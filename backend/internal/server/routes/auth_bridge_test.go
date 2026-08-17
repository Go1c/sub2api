//go:build unit

package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type authBridgeUserRepoStub struct {
	service.UserRepository
	user *service.User
}

func (s *authBridgeUserRepoStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, service.ErrInvalidCredentials
	}
	cloned := *s.user
	return &cloned, nil
}

func (s *authBridgeUserRepoStub) GetUserAvatar(context.Context, int64) (*service.UserAvatar, error) {
	return nil, nil
}

func (s *authBridgeUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}

type authBridgeRefreshTokenCacheStub struct{}

func (s *authBridgeRefreshTokenCacheStub) StoreRefreshToken(context.Context, string, *service.RefreshTokenData, time.Duration) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) GetRefreshToken(context.Context, string) (*service.RefreshTokenData, error) {
	return nil, service.ErrRefreshTokenNotFound
}
func (s *authBridgeRefreshTokenCacheStub) DeleteRefreshToken(context.Context, string) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}
func (s *authBridgeRefreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (s *authBridgeRefreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}
func (s *authBridgeRefreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}

type authBridgeUATRepoStub struct {
	byHash map[string]*service.UserAccessToken
	byID   map[int64]*service.UserAccessToken
	nextID int64
}

func newAuthBridgeUATRepoStub() *authBridgeUATRepoStub {
	return &authBridgeUATRepoStub{
		byHash: make(map[string]*service.UserAccessToken),
		byID:   make(map[int64]*service.UserAccessToken),
		nextID: 1,
	}
}

func (r *authBridgeUATRepoStub) Create(_ context.Context, token *service.UserAccessToken) error {
	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	cp := *token
	cp.ID = id
	cp.CreatedAt = now
	cp.UpdatedAt = now
	r.byID[id] = &cp
	r.byHash[cp.TokenHash] = &cp
	token.ID = id
	token.CreatedAt = now
	token.UpdatedAt = now
	return nil
}

func (r *authBridgeUATRepoStub) GetByID(_ context.Context, id int64) (*service.UserAccessToken, error) {
	t, ok := r.byID[id]
	if !ok {
		return nil, service.ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *authBridgeUATRepoStub) GetByTokenHash(_ context.Context, hash string) (*service.UserAccessToken, error) {
	t, ok := r.byHash[hash]
	if !ok {
		return nil, service.ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *authBridgeUATRepoStub) ListByUserID(_ context.Context, userID int64) ([]service.UserAccessToken, error) {
	out := make([]service.UserAccessToken, 0)
	for _, t := range r.byID {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *authBridgeUATRepoStub) CountActiveByUserID(_ context.Context, userID int64, now time.Time) (int, error) {
	n := 0
	for _, t := range r.byID {
		if t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			n++
		}
	}
	return n, nil
}

func (r *authBridgeUATRepoStub) RevokeByIDForUser(_ context.Context, userID, id int64, revokedAt time.Time) error {
	t, ok := r.byID[id]
	if !ok || t.UserID != userID {
		return service.ErrUserAccessTokenNotFound
	}
	t.RevokedAt = &revokedAt
	return nil
}

func (r *authBridgeUATRepoStub) TouchLastUsedAt(_ context.Context, id int64, usedAt time.Time) error {
	if t, ok := r.byID[id]; ok {
		t.LastUsedAt = &usedAt
	}
	return nil
}

type authBridgeTestEnv struct {
	router    *gin.Engine
	authSvc   *service.AuthService
	user      *service.User
	uatToken  string
	jwtSecret string
}

func newAuthBridgeRouteEnv(t *testing.T) authBridgeTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	user := &service.User{
		ID:           77,
		Email:        "bridge-route@example.com",
		Username:     "bridge-route",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  3,
		TokenVersion: 1,
	}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-jwt-secret-32bytes-long!!!",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
	}
	userRepo := &authBridgeUserRepoStub{user: user}
	authSvc := service.NewAuthService(nil, userRepo, nil, &authBridgeRefreshTokenCacheStub{}, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	userSvc := service.NewUserService(userRepo, nil, nil, nil)
	uatSvc := service.NewUserAccessTokenService(newAuthBridgeUATRepoStub())
	created, err := uatSvc.Create(context.Background(), user.ID, service.CreateUserAccessTokenInput{Name: "bridge-uat"})
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterAuthRoutes(
		v1,
		&handler.Handlers{
			Auth:    handler.NewAuthHandler(cfg, authSvc, userSvc, nil, nil, nil, nil, nil),
			Setting: &handler.SettingHandler{},
		},
		servermiddleware.NewJWTAuthMiddleware(authSvc, userSvc, nil, nil, uatSvc),
		servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() }),
		rdb,
		nil,
	)

	return authBridgeTestEnv{
		router:    router,
		authSvc:   authSvc,
		user:      user,
		uatToken:  created.Token,
		jwtSecret: cfg.JWT.Secret,
	}
}

func postAuthBridge(router http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bridge", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAuthBridgeRouteIsRegistered(t *testing.T) {
	env := newAuthBridgeRouteEnv(t)
	found := false
	for _, route := range env.router.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/auth/bridge" {
			found = true
			break
		}
	}
	require.True(t, found, "POST /api/v1/auth/bridge must be registered")
}

func TestAuthBridgeRequiresAccessJWT(t *testing.T) {
	env := newAuthBridgeRouteEnv(t)

	missing := postAuthBridge(env.router, "")
	require.Equal(t, http.StatusUnauthorized, missing.Code)

	forged := postAuthBridge(env.router, "not-a-jwt")
	require.Equal(t, http.StatusUnauthorized, forged.Code)

	refresh := postAuthBridge(env.router, "rt_"+("aa"+t.Name()))
	require.Equal(t, http.StatusUnauthorized, refresh.Code)
}

func TestAuthBridgeRejectsExpiredAccessJWT(t *testing.T) {
	env := newAuthBridgeRouteEnv(t)
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":       env.user.ID,
		"email":         env.user.Email,
		"role":          env.user.Role,
		"token_version": env.user.TokenVersion,
		"exp":           time.Now().Add(-time.Hour).Unix(),
		"iat":           time.Now().Add(-2 * time.Hour).Unix(),
	})
	token, err := expired.SignedString([]byte(env.jwtSecret))
	require.NoError(t, err)

	w := postAuthBridge(env.router, token)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body servermiddleware.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "TOKEN_EXPIRED", body.Code)
}

func TestAuthBridgeRejectsUserAccessToken(t *testing.T) {
	env := newAuthBridgeRouteEnv(t)
	w := postAuthBridge(env.router, env.uatToken)
	require.Equal(t, http.StatusForbidden, w.Code)
	var body servermiddleware.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ACCESS_TOKEN_SCOPE_DENIED", body.Code)
}

func TestAuthBridgeExchangesValidAccessJWT(t *testing.T) {
	env := newAuthBridgeRouteEnv(t)
	inbound, err := env.authSvc.GenerateToken(context.Background(), env.user)
	require.NoError(t, err)

	w := postAuthBridge(env.router, inbound)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			TokenType    string `json:"token_type"`
			User         struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "Bearer", resp.Data.TokenType)
	require.Equal(t, env.user.ID, resp.Data.User.ID)
	require.NotEmpty(t, resp.Data.AccessToken)
	require.NotEmpty(t, resp.Data.RefreshToken)
	require.Greater(t, resp.Data.ExpiresIn, 0)
	require.NotEqual(t, inbound, resp.Data.AccessToken)
	require.NotEqual(t, inbound, resp.Data.RefreshToken)
}
