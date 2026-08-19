package middleware

import (
	"net/http"
	"strings"

	responsepkg "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RequireWalletHeaderJWT rejects cookie and opaque-token authorization for debit.
// Signature, expiry, token version, session binding and user status have already
// been checked by JWTAuthMiddleware before this guard runs.
func RequireWalletHeaderJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" || IsUserAccessTokenAuth(c) {
			responsepkg.WalletError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token", nil)
			c.Abort()
			return
		}
		method := c.GetString(string(ContextKeyAuthMethod))
		if method != AuthMethodJWT {
			responsepkg.WalletError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func WalletBackendModeGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsBackendModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if role == "admin" {
			c.Next()
			return
		}
		responsepkg.WalletError(c, http.StatusForbidden, "BACKEND_MODE_ACTIVE", "backend mode is active", nil)
		c.Abort()
	}
}
