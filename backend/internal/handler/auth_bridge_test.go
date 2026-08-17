//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAuthBridgeHandler(t *testing.T) (*AuthHandler, *service.User, *service.AuthService) {
	t.Helper()

	user := &service.User{
		ID:           41,
		Email:        "bridge@example.com",
		Username:     "bridge-user",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		TokenVersion: 3,
	}
	repo := &userHandlerRepoStub{user: user}
	refreshTokenCache := &userHandlerRefreshTokenCacheStub{}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-secret",
			ExpireHour:               1,
			RefreshTokenExpireDays:   7,
			AccessTokenExpireMinutes: 60,
		},
	}
	authService := service.NewAuthService(nil, repo, nil, refreshTokenCache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	handler := &AuthHandler{
		authService: authService,
		userService: service.NewUserService(repo, nil, nil, nil),
	}
	return handler, user, authService
}

func TestAuthHandlerBridgeRequiresAuthenticatedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newAuthBridgeHandler(t)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/bridge", nil)

	handler.Bridge(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuthHandlerBridgeIssuesNewTokenPairWithoutEchoingInbound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, user, authService := newAuthBridgeHandler(t)

	inbound, err := authService.GenerateToken(t.Context(), user)
	require.NoError(t, err)
	require.NotEmpty(t, inbound)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/bridge", nil)
	c.Request.Header.Set("Authorization", "Bearer "+inbound)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.Bridge(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			TokenType    string `json:"token_type"`
			User         struct {
				ID    int64  `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "success", resp.Message)
	require.Equal(t, "Bearer", resp.Data.TokenType)
	require.Equal(t, user.ID, resp.Data.User.ID)
	require.Equal(t, user.Email, resp.Data.User.Email)
	require.NotEmpty(t, resp.Data.AccessToken)
	require.NotEmpty(t, resp.Data.RefreshToken)
	require.Greater(t, resp.Data.ExpiresIn, 0)
	require.NotEqual(t, inbound, resp.Data.AccessToken)
	require.NotEqual(t, inbound, resp.Data.RefreshToken)
	require.True(t, len(resp.Data.RefreshToken) > 3 && resp.Data.RefreshToken[:3] == "rt_")
}
