package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewayUsageUserRepoStub struct {
	service.UserRepository
	users map[int64]*service.User
}

func (r *gatewayUsageUserRepoStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cloned := *user
	return &cloned, nil
}

func (r *gatewayUsageUserRepoStub) GetUserAvatar(_ context.Context, _ int64) (*service.UserAvatar, error) {
	return nil, nil
}

func TestGatewayUsageQuotaLimitedIncludesWalletBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := int64(42)
	userRepo := &gatewayUsageUserRepoStub{
		users: map[int64]*service.User{
			userID: {
				ID:      userID,
				Balance: 5.5,
				Status:  service.StatusActive,
			},
		},
	}
	h := &GatewayHandler{
		userService: service.NewUserService(userRepo, nil, nil, nil),
	}

	r := gin.New()
	r.GET("/v1/usage", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:        7,
			UserID:    userID,
			Status:    service.StatusAPIKeyActive,
			Quota:     20,
			QuotaUsed: 7,
		})
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		h.Usage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "quota_limited", body["mode"])
	floatValue := func(key string) float64 {
		value, ok := body[key].(float64)
		require.Truef(t, ok, "%s should be a number", key)
		return value
	}
	require.InDelta(t, 13.0, floatValue("remaining"), 1e-9)
	require.InDelta(t, 5.5, floatValue("balance"), 1e-9)
	require.InDelta(t, 5.5, floatValue("wallet_balance"), 1e-9)
	require.InDelta(t, 5.5, floatValue("account_balance"), 1e-9)
}
