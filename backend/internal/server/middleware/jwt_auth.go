package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// Auth method values stored under ContextKeyAuthMethod.
const (
	AuthMethodJWT             = "jwt"
	AuthMethodUserAccessToken = "user_access_token"
)

// NewJWTAuthMiddleware 创建 JWT / User Access Token 统一认证中间件
func NewJWTAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
	accessTokenService *service.UserAccessTokenService,
) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuth(authService, userService, userService, settingService, auditService, accessTokenService))
}

type jwtUserReader interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

type userActivityToucher interface {
	TouchLastActiveForUser(ctx context.Context, user *service.User)
}

// jwtAuth JWT / opaque user access token 认证中间件实现
func jwtAuth(
	authService *service.AuthService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
	accessTokenService *service.UserAccessTokenService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""
		tokenFromCookie := false
		// WebSocket browsers cannot set Authorization; allow jwt.<token> via Sec-WebSocket-Protocol.
		if isWebSocketUpgradeRequest(c) {
			if t := extractJWTFromWebSocketSubprotocol(c); t != "" {
				tokenString = t
			}
		}
		if tokenString == "" {
			// 从Authorization header中提取token
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				if cookieToken, err := c.Cookie(service.LumioWebSessionCookieName); err == nil {
					tokenString = strings.TrimSpace(cookieToken)
					tokenFromCookie = tokenString != ""
				}
				if tokenString == "" {
					AbortWithError(c, 401, "UNAUTHORIZED", "Authorization header is required")
					return
				}
			} else {
				// 验证Bearer scheme
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
					AbortWithError(c, 401, "INVALID_AUTH_HEADER", "Authorization header format must be 'Bearer {token}'")
					return
				}

				tokenString = strings.TrimSpace(parts[1])
				if tokenString == "" {
					AbortWithError(c, 401, "EMPTY_TOKEN", "Token cannot be empty")
					return
				}
			}
		}

		// Opaque user access token path (not a JWT: no two '.' segments).
		if !tokenFromCookie && looksLikeUserAccessToken(tokenString) {
			if !authenticateUserAccessToken(c, accessTokenService, userService, activityToucher, tokenString) {
				return
			}
			if !enforceUserAccessTokenPathScope(c) {
				return
			}
			c.Next()
			return
		}

		// 验证 JWT token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
				return
			}
			AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
			return
		}

		// 从数据库获取最新的用户信息
		user, err := userService.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
			return
		}

		// 检查用户状态
		if !user.IsActive() {
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}

		// Security: Validate TokenVersion to ensure token hasn't been invalidated
		// This check ensures tokens issued before a password change are rejected
		if claims.TokenVersion != user.TokenVersion {
			AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
			return
		}

		// 会话绑定校验：IP/UA 任一变化即撤销会话（功能可在系统设置中关闭）
		if !enforceSessionBinding(c, authService, settingService, auditService, claims) {
			return
		}

		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Set(ContextKeyAuthEmail, user.Email)
		c.Set(ContextKeySessionID, claims.SessionID)
		c.Set(string(ContextKeyAuthMethod), AuthMethodJWT)
		if activityToucher != nil {
			activityToucher.TouchLastActiveForUser(c.Request.Context(), user)
		}

		c.Next()
	}
}

// looksLikeUserAccessToken detects opaque user access tokens.
// Prefer uat_ prefix; also treat non-JWT-shaped tokens with that prefix only.
func looksLikeUserAccessToken(token string) bool {
	return service.IsUserAccessTokenShape(token)
}

func authenticateUserAccessToken(
	c *gin.Context,
	accessTokenService *service.UserAccessTokenService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	tokenString string,
) bool {
	if accessTokenService == nil {
		AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
		return false
	}

	rec, err := accessTokenService.ValidateToken(c.Request.Context(), tokenString)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAccessTokenExpired):
			AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
		case errors.Is(err, service.ErrUserAccessTokenRevoked):
			AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked")
		default:
			AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
		}
		return false
	}

	user, err := userService.GetByID(c.Request.Context(), rec.UserID)
	if err != nil {
		AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
		return false
	}
	if !user.IsActive() {
		AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set(ContextKeyAuthEmail, user.Email)
	c.Set(string(ContextKeyAuthMethod), AuthMethodUserAccessToken)
	c.Set(string(ContextKeyUserAccessTokenID), rec.ID)

	if activityToucher != nil {
		activityToucher.TouchLastActiveForUser(c.Request.Context(), user)
	}
	// best-effort last_used_at
	go accessTokenService.TouchLastUsed(context.Background(), rec.ID)

	return true
}

// enforceUserAccessTokenPathScope restricts opaque-token auth to the API whitelist
// (key management + read-only usage logs / wallet balance / subscription credit).
func enforceUserAccessTokenPathScope(c *gin.Context) bool {
	if !IsUserAccessTokenAuth(c) {
		return true
	}
	path := c.Request.URL.Path
	method := c.Request.Method
	if method == "POST" && path == "/api/v1/user/balance/debit" {
		AbortWithError(c, 401, "INVALID_TOKEN", "invalid token")
		return false
	}
	if isUserAccessTokenAllowedPath(method, path) {
		return true
	}
	AbortWithError(c, 403, "ACCESS_TOKEN_SCOPE_DENIED", "Access token is not allowed for this endpoint")
	return false
}

// isUserAccessTokenAllowedPath returns true for whitelist paths under /api/v1.
// Write operations outside key management remain denied (profile update, access-token
// management, subscription weekly-reset, payment, admin, etc.).
func isUserAccessTokenAllowedPath(method, path string) bool {
	// Normalize: strip trailing slash except root
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// GET/POST /api/v1/keys
	if path == "/api/v1/keys" {
		return method == "GET" || method == "POST"
	}
	// GET/PUT/DELETE /api/v1/keys/:id
	if strings.HasPrefix(path, "/api/v1/keys/") {
		rest := strings.TrimPrefix(path, "/api/v1/keys/")
		if rest != "" && !strings.Contains(rest, "/") {
			return method == "GET" || method == "PUT" || method == "DELETE"
		}
		return false
	}
	// GET /api/v1/groups/available
	if path == "/api/v1/groups/available" && method == "GET" {
		return true
	}
	// GET /api/v1/groups/rates
	if path == "/api/v1/groups/rates" && method == "GET" {
		return true
	}

	// GET /api/v1/user/profile — wallet balance (balance / frozen_balance) among profile fields
	if path == "/api/v1/user/profile" && method == "GET" {
		return true
	}
	// GET /api/v1/auth/me — same wallet balance fields as profile (frontend primary path)
	if path == "/api/v1/auth/me" && method == "GET" {
		return true
	}
	if path == "/api/v1/user/balance/transactions" && method == "GET" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/user/balance/transactions/") && method == "GET" {
		rest := strings.TrimPrefix(path, "/api/v1/user/balance/transactions/")
		return rest != "" && !strings.Contains(rest, "/")
	}

	// Usage / request logs (read-only; includes dashboard stats and batch query POST)
	if path == "/api/v1/usage" && method == "GET" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/usage/") {
		rest := strings.TrimPrefix(path, "/api/v1/usage/")
		switch rest {
		case "stats", "dashboard/stats", "dashboard/trend", "dashboard/models":
			return method == "GET"
		case "dashboard/api-keys-usage":
			return method == "POST"
		default:
			// GET /api/v1/usage/:id (numeric-ish single segment; reject nested)
			if rest != "" && !strings.Contains(rest, "/") {
				return method == "GET"
			}
		}
		return false
	}

	// Subscription credit / remaining quotas (read-only; no weekly-limit reset)
	if path == "/api/v1/subscriptions" && method == "GET" {
		return true
	}
	if path == "/api/v1/subscriptions/active" && method == "GET" {
		return true
	}
	if path == "/api/v1/subscriptions/progress" && method == "GET" {
		return true
	}
	if path == "/api/v1/subscriptions/summary" && method == "GET" {
		return true
	}

	return false
}

// IsUserAccessTokenAuth reports whether the request was authenticated via user access token.
func IsUserAccessTokenAuth(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(string(ContextKeyAuthMethod))
	if !ok {
		return false
	}
	s, _ := v.(string)
	return s == AuthMethodUserAccessToken
}

// Deprecated: prefer GetAuthSubjectFromContext in auth_subject.go.
