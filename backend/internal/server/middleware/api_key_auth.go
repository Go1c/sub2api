package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const insufficientBalanceMessage = "账户余额不足，请先充值后再使用。"

// NewAPIKeyAuthMiddleware 创建 API Key 认证中间件
func NewAPIKeyAuthMiddleware(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) APIKeyAuthMiddleware {
	return APIKeyAuthMiddleware(apiKeyAuthWithSubscription(apiKeyService, subscriptionService, cfg))
}

// apiKeyAuthWithSubscription API Key认证中间件（支持订阅验证）
//
// 中间件职责分为两层：
//   - 鉴权（Authentication）：验证 Key 有效性、用户状态、IP 限制 —— 始终执行
//   - 计费执行（Billing Enforcement）：过期/配额/订阅/余额检查 —— skipBilling 时整块跳过
//
// /v1/usage 端点只需鉴权，不需要计费执行（允许过期/配额耗尽的 Key 查询自身用量）。
func apiKeyAuthWithSubscription(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── 1. 提取 API Key ──────────────────────────────────────────

		queryKey := strings.TrimSpace(c.Query("key"))
		queryApiKey := strings.TrimSpace(c.Query("api_key"))
		if queryKey != "" || queryApiKey != "" {
			AbortWithError(c, 400, "api_key_in_query_deprecated", "API key in query parameter is deprecated. Please use Authorization header instead.")
			return
		}

		// 尝试从Authorization header中提取API key (Bearer scheme)
		authHeader := c.GetHeader("Authorization")
		var apiKeyString string

		if authHeader != "" {
			// 验证Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				apiKeyString = strings.TrimSpace(parts[1])
			}
		}

		// 如果Authorization header中没有，尝试从x-api-key header中提取
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-api-key")
		}

		// 如果x-api-key header中没有，尝试从x-goog-api-key header中提取（Gemini CLI兼容）
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-goog-api-key")
		}

		// 如果所有header都没有API key
		if apiKeyString == "" {
			AbortWithError(c, 401, "API_KEY_REQUIRED", "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header")
			return
		}

		// ── 2. 验证 Key 存在 ─────────────────────────────────────────

		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				AbortWithError(c, 401, "INVALID_API_KEY", "Invalid API key")
				return
			}
			AbortWithError(c, 500, "INTERNAL_ERROR", "Failed to validate API key")
			return
		}

		// ── 3. 基础鉴权（始终执行） ─────────────────────────────────

		// disabled / 未知状态 → 无条件拦截（expired 和 quota_exhausted 留给计费阶段）
		if !apiKey.IsActive() &&
			apiKey.Status != service.StatusAPIKeyExpired &&
			apiKey.Status != service.StatusAPIKeyQuotaExhausted {
			AbortWithError(c, 401, "API_KEY_DISABLED", "API key is disabled")
			return
		}

		// 检查 IP 限制（白名单/黑名单）
		// 注意：错误信息故意模糊，避免暴露具体的 IP 限制机制
		if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
			clientIP := ip.GetTrustedClientIP(c)
			allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
			if !allowed {
				AbortWithError(c, 403, "ACCESS_DENIED", "Access denied")
				return
			}
		}

		// 检查关联的用户
		if apiKey.User == nil {
			AbortWithError(c, 401, "USER_NOT_FOUND", "User associated with API key not found")
			return
		}

		// 检查用户状态
		if !apiKey.User.IsActive() {
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}

		// 独占分组授权守卫：分组被停用/删除，或 API Key 所属独占分组已不再授权当前用户时拒绝。
		if abortIfAPIKeyGroupUnavailable(c, apiKey) {
			return
		}
		if abortIfAPIKeyGroupNotAllowed(c, apiKey) {
			return
		}
		ctx := context.WithValue(c.Request.Context(), ctxkey.UserID, apiKey.User.ID)
		c.Request = c.Request.WithContext(ctx)

		// ── 4. SimpleMode → early return ─────────────────────────────

		if cfg.RunMode == config.RunModeSimple {
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setGroupContext(c, apiKey.Group)
			_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
			c.Next()
			return
		}

		// ── 5. 加载可消费订阅（用户级额度池，不再依赖分组 subscription_type） ──

		// skipBilling: usage 查询只需鉴权，跳过所有计费执行
		skipBilling := isGatewayUsagePath(c.Request.URL.Path)

		var subscription *service.UserSubscription
		var subscriptionCandidateIDs []int64
		var subscriptionErr error

		if subscriptionService != nil {
			sub, candidateIDs, subErr := loadUsableCreditSubscriptionForAuth(
				c.Request.Context(),
				subscriptionService,
				apiKey.User.ID,
				apiKey.Group,
				apiKey.User,
			)
			if subErr != nil {
				if !skipBilling && !isFallbackableSubscriptionAuthError(subErr) {
					status, code := subscriptionAuthErrorStatus(subErr)
					AbortWithError(c, status, code, subErr.Error())
					return
				}
				subscriptionErr = subErr
			} else {
				subscription = sub
				subscriptionCandidateIDs = candidateIDs
			}
		}

		// ── 6. 计费执行（skipBilling 时整块跳过） ────────────────────

		if !skipBilling {
			// Key 状态检查
			switch apiKey.Status {
			case service.StatusAPIKeyQuotaExhausted:
				AbortWithError(c, 403, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
				return
			case service.StatusAPIKeyExpired:
				AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
				return
			}

			// 运行时过期/配额检查（即使状态是 active，也要检查时间和用量）
			if apiKey.IsExpired() {
				AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
				return
			}
			if apiKey.IsQuotaExhausted() {
				AbortWithError(c, 403, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
				return
			}

			if subscription == nil {
				if apiKey.User.Balance <= 0 {
					if subscriptionErr != nil {
						status, code := subscriptionAuthErrorStatus(subscriptionErr)
						AbortWithError(c, status, code, subscriptionErr.Error())
						return
					}
					AbortWithError(c, 403, "INSUFFICIENT_BALANCE", insufficientBalanceMessage)
					return
				}
			}
		}

		// ── 7. 设置上下文 → Next ─────────────────────────────────────

		if subscription != nil {
			c.Set(string(ContextKeySubscription), subscription)
		}
		if len(subscriptionCandidateIDs) > 0 {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.SubscriptionCandidateIDs, subscriptionCandidateIDs))
		}
		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContext(c, apiKey.Group)
		_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)

		c.Next()
	}
}

// GetAPIKeyFromContext 从上下文中获取API key
func GetAPIKeyFromContext(c *gin.Context) (*service.APIKey, bool) {
	value, exists := c.Get(string(ContextKeyAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// GetSubscriptionFromContext 从上下文中获取订阅信息
func GetSubscriptionFromContext(c *gin.Context) (*service.UserSubscription, bool) {
	value, exists := c.Get(string(ContextKeySubscription))
	if !exists {
		return nil, false
	}
	subscription, ok := value.(*service.UserSubscription)
	return subscription, ok
}

func setGroupContext(c *gin.Context, group *service.Group) {
	if !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group); ok && existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	ctx := context.WithValue(c.Request.Context(), ctxkey.Group, group)
	c.Request = c.Request.WithContext(ctx)
}

func loadUsableCreditSubscriptionForAuth(ctx context.Context, subscriptionService *service.SubscriptionService, userID int64, group *service.Group, user *service.User) (*service.UserSubscription, []int64, error) {
	if subscriptionService == nil {
		return nil, nil, nil
	}
	subs, err := subscriptionService.ListUsableCreditSubscriptions(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var firstErr error
	var selected *service.UserSubscription
	candidateIDs := make([]int64, 0, len(subs))
	for i := range subs {
		sub := subs[i]
		if !service.SubscriptionCoversGroup(&sub, group, user) {
			if firstErr == nil {
				firstErr = service.ErrSubscriptionInvalid
			}
			continue
		}
		candidateIDs = append(candidateIDs, sub.ID)
		if _, err := subscriptionService.ValidateAndCheckLimits(&sub, group); err != nil {
			if !isFallbackableSubscriptionAuthError(err) {
				return nil, nil, err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if selected == nil {
			selected = &sub
		}
	}
	if selected != nil {
		return selected, candidateIDs, nil
	}
	if firstErr != nil {
		return nil, candidateIDs, firstErr
	}
	return nil, nil, nil
}

func isFallbackableSubscriptionAuthError(err error) bool {
	return errors.Is(err, service.ErrSubscriptionInvalid) ||
		errors.Is(err, service.ErrSubscriptionExpired) ||
		errors.Is(err, service.ErrSubscriptionSuspended) ||
		errors.Is(err, service.ErrDailyLimitExceeded) ||
		errors.Is(err, service.ErrWeeklyLimitExceeded) ||
		errors.Is(err, service.ErrMonthlyLimitExceeded)
}

func subscriptionAuthErrorStatus(err error) (int, string) {
	if isFallbackableSubscriptionAuthError(err) {
		return 403, "SUBSCRIPTION_INVALID"
	}
	return 500, "INTERNAL_ERROR"
}

// abortIfAPIKeyGroupUnavailable 在 API Key 所属分组已删除/停用时以 403 拒绝。
func abortIfAPIKeyGroupUnavailable(c *gin.Context, apiKey *service.APIKey) bool {
	code, message, ok := validateAPIKeyGroupAvailable(apiKey)
	if ok {
		return false
	}
	AbortWithError(c, 403, code, message)
	return true
}

// abortIfAPIKeyGroupNotAllowed 在 API Key 所属独占分组不再授权当前用户时以 403 拒绝（越权防护）。
func abortIfAPIKeyGroupNotAllowed(c *gin.Context, apiKey *service.APIKey) bool {
	if validateAPIKeyGroupAllowed(apiKey) {
		return false
	}
	AbortWithError(c, 403, "GROUP_NOT_ALLOWED", "API Key 所属专属分组不再允许当前用户使用")
	return true
}

// validateAPIKeyGroupAllowed 判断 API Key 所属分组是否仍允许其所属用户访问。
// 订阅型分组直接放行；独占分组要求用户当前仍被授权绑定该分组。
func validateAPIKeyGroupAllowed(apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.GroupID == nil || apiKey.User == nil || apiKey.Group == nil {
		return true
	}
	group := apiKey.Group
	if group.IsSubscriptionType() {
		return true
	}
	return apiKey.User.CanBindGroup(group.ID, group.IsExclusive)
}

// validateAPIKeyGroupAvailable 判断 API Key 所属分组是否可用（未删除、未停用）。
func validateAPIKeyGroupAvailable(apiKey *service.APIKey) (string, string, bool) {
	if apiKey == nil || apiKey.GroupID == nil {
		return "", "", true
	}
	group := apiKey.Group
	if group == nil || strings.EqualFold(group.Status, "deleted") {
		return "GROUP_DELETED", "API Key 所属分组已删除", false
	}
	if !group.IsActive() {
		return "GROUP_DISABLED", "API Key 所属分组已停用", false
	}
	return "", "", true
}
