package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	responsepkg "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ContextKey 定义上下文键类型
type ContextKey string

const (
	// ContextKeyUser 用户上下文键
	ContextKeyUser ContextKey = "user"
	// ContextKeyUserRole 当前用户角色（string）
	ContextKeyUserRole ContextKey = "user_role"
	// ContextKeyAPIKey API密钥上下文键
	ContextKeyAPIKey ContextKey = "api_key"
	// ContextKeySubscription 订阅上下文键
	ContextKeySubscription ContextKey = "subscription"
	// ContextKeyForcePlatform 强制平台（用于 /antigravity 路由）
	ContextKeyForcePlatform ContextKey = "force_platform"
	// ContextKeyOpsFallbackAPIKey 运维错误日志专用回退键。
	// 鉴权早退（分组停用/删除、Key 停用/过期/额度、用户停用、IP 限制等）时，
	// apiKey 已加载但尚未写入 ContextKeyAPIKey；该键让 Ops 错误日志仍能取到
	// user/group/platform。仅供 Ops 错误日志读取，不代表请求已通过鉴权。
	ContextKeyOpsFallbackAPIKey ContextKey = "ops_fallback_api_key"
	// ContextKeyAuthMethod 认证方式：jwt | user_access_token
	ContextKeyAuthMethod ContextKey = "auth_method"
	// ContextKeyUserAccessTokenID opaque access token 记录 ID（仅 user_access_token 认证时设置）
	ContextKeyUserAccessTokenID ContextKey = "user_access_token_id"
	// ContextKeyBalanceClientID 通过钱包消费方认证后的稳定 client_id（不含 secret/hash）。
	ContextKeyBalanceClientID ContextKey = "balance_client_id"
)

// ForcePlatform 返回设置强制平台的中间件
// 同时设置 request.Context（供 Service 使用）和 gin.Context（供 Handler 快速检查）
func ForcePlatform(platform string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置到 request.Context，使用 ctxkey.ForcePlatform 供 Service 层读取
		ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, platform)
		c.Request = c.Request.WithContext(ctx)
		// 同时设置到 gin.Context，供 Handler 快速检查
		c.Set(string(ContextKeyForcePlatform), platform)
		c.Next()
	}
}

// HasForcePlatform 检查是否有强制平台（用于 Handler 跳过分组检查）
func HasForcePlatform(c *gin.Context) bool {
	_, exists := c.Get(string(ContextKeyForcePlatform))
	return exists
}

// GetForcePlatformFromContext 从 gin.Context 获取强制平台
func GetForcePlatformFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyForcePlatform))
	if !exists {
		return "", false
	}
	platform, ok := value.(string)
	return platform, ok
}

// ErrorResponse 标准错误响应结构
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// AbortWithError 中断请求并返回JSON错误
func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	if isBalanceWalletUserPath(c.Request.URL.Path) {
		switch code {
		case "TOKEN_EXPIRED":
			statusCode = http.StatusUnauthorized
		case "USER_INACTIVE":
			statusCode = http.StatusForbidden
		default:
			statusCode = http.StatusUnauthorized
			code = "INVALID_TOKEN"
			message = "invalid token"
		}
		responsepkg.WalletError(c, statusCode, code, message, nil)
		c.Abort()
		return
	}
	if isBalanceClientAdminPath(c.Request.URL.Path) {
		responsepkg.WalletError(c, statusCode, code, message, nil)
		c.Abort()
		return
	}
	c.JSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

func isBalanceWalletUserPath(path string) bool {
	return path == "/api/v1/user/balance" || strings.HasPrefix(path, "/api/v1/user/balance/")
}

func isBalanceClientAdminPath(path string) bool {
	return path == "/api/v1/admin/balance-clients" || strings.HasPrefix(path, "/api/v1/admin/balance-clients/")
}

// abortWithOpenAIQuotaError writes the OpenAI-compatible insufficient quota response.
func abortWithOpenAIQuotaError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "insufficient_quota",
			"param":   nil,
			"code":    "insufficient_quota",
		},
	})
	c.Abort()
}

// ──────────────────────────────────────────────────────────
// RequireGroupAssignment — 未分组 Key 拦截中间件
// ──────────────────────────────────────────────────────────

// GatewayErrorWriter 定义网关错误响应格式（不同协议使用不同格式）
type GatewayErrorWriter func(c *gin.Context, status int, message string)

// AnthropicErrorWriter 按 Anthropic API 规范输出错误
func AnthropicErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"type":  "error",
		"error": gin.H{"type": "permission_error", "message": message},
	})
}

// GoogleErrorWriter 按 Google API 规范输出错误
func GoogleErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
}

// RequireGroupAssignment 检查 API Key 是否已分配到分组，
// 如果未分组且系统设置不允许未分组 Key 调度则返回 403。
func RequireGroupAssignment(settingService *service.SettingService, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isGatewayUsagePath(c.Request.URL.Path) {
			c.Next()
			return
		}
		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey.GroupID != nil {
			c.Next()
			return
		}
		// 未分组 Key — 检查系统设置
		if settingService.IsUngroupedKeySchedulingAllowed(c.Request.Context()) {
			c.Next()
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnassigned)
		MarkIngressRejected(c, IngressRejectGroupUnassigned)
		writeError(c, http.StatusForbidden, "API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group.")
		c.Abort()
	}
}

func isGatewayUsagePath(path string) bool {
	return path == "/v1/usage" || path == "/antigravity/v1/usage"
}

// isAsyncImageTaskRead reports whether the request only polls an existing
// asynchronous image task. Task results already belong to the authenticated
// key and must remain readable after generation has consumed remaining balance.
func isAsyncImageTaskRead(method, path string) bool {
	if method != "GET" {
		return false
	}
	return strings.HasPrefix(path, "/v1/images/tasks/") || strings.HasPrefix(path, "/images/tasks/")
}

// isGatewayModelsListPath reports whether the request only lists or inspects
// gateway model catalogs. Catalogs are discovery, not billed consumption.
func isGatewayModelsListPath(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	path = strings.TrimRight(path, "/")
	switch path {
	case "/v1/models",
		"/models",
		"/backend-api/codex/models",
		"/v1beta/models",
		"/antigravity/models",
		"/antigravity/v1/models",
		"/antigravity/v1beta/models":
		return true
	}
	return isSingleSegmentModelsLookup(path, "/v1beta/models/") ||
		isSingleSegmentModelsLookup(path, "/antigravity/v1beta/models/")
}

func isSingleSegmentModelsLookup(path, prefix string) bool {
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok || rest == "" || strings.Contains(rest, "/") {
		return false
	}
	return true
}
