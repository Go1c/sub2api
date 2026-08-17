//go:build unit

package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type uatRepoStub struct {
	byHash map[string]*service.UserAccessToken
	byID   map[int64]*service.UserAccessToken
	nextID int64
}

func newUATRepoStub() *uatRepoStub {
	return &uatRepoStub{
		byHash: make(map[string]*service.UserAccessToken),
		byID:   make(map[int64]*service.UserAccessToken),
		nextID: 1,
	}
}

func (r *uatRepoStub) Create(_ context.Context, token *service.UserAccessToken) error {
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

func (r *uatRepoStub) GetByID(_ context.Context, id int64) (*service.UserAccessToken, error) {
	t, ok := r.byID[id]
	if !ok {
		return nil, service.ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *uatRepoStub) GetByTokenHash(_ context.Context, hash string) (*service.UserAccessToken, error) {
	t, ok := r.byHash[hash]
	if !ok {
		return nil, service.ErrUserAccessTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *uatRepoStub) ListByUserID(_ context.Context, userID int64) ([]service.UserAccessToken, error) {
	out := make([]service.UserAccessToken, 0)
	for _, t := range r.byID {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *uatRepoStub) CountActiveByUserID(_ context.Context, userID int64, now time.Time) (int, error) {
	n := 0
	for _, t := range r.byID {
		if t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			n++
		}
	}
	return n, nil
}

func (r *uatRepoStub) RevokeByIDForUser(_ context.Context, userID, id int64, revokedAt time.Time) error {
	t, ok := r.byID[id]
	if !ok || t.UserID != userID {
		return service.ErrUserAccessTokenNotFound
	}
	t.RevokedAt = &revokedAt
	return nil
}

func (r *uatRepoStub) TouchLastUsedAt(_ context.Context, id int64, usedAt time.Time) error {
	if t, ok := r.byID[id]; ok {
		t.LastUsedAt = &usedAt
	}
	return nil
}

func newUATAuthEnv(t *testing.T, user *service.User) (*gin.Engine, *service.UserAccessTokenService, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.JWT.Secret = "test-jwt-secret-32bytes-long!!!"
	cfg.JWT.AccessTokenExpireMinutes = 60

	users := map[int64]*service.User{}
	if user != nil {
		users[user.ID] = user
	}
	userRepo := &stubJWTUserRepo{users: users}
	authSvc := service.NewAuthService(nil, userRepo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	userSvc := service.NewUserService(userRepo, nil, nil, nil)

	repo := newUATRepoStub()
	uatSvc := service.NewUserAccessTokenService(repo)

	created, err := uatSvc.Create(context.Background(), user.ID, service.CreateUserAccessTokenInput{Name: "test"})
	require.NoError(t, err)

	mw := NewJWTAuthMiddleware(authSvc, userSvc, nil, nil, uatSvc)
	r := gin.New()
	r.Use(gin.HandlerFunc(mw))
	r.Any("/*path", func(c *gin.Context) {
		subject, _ := GetAuthSubjectFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":     subject.UserID,
			"auth_method": c.GetString(string(ContextKeyAuthMethod)),
			"path":        c.Request.URL.Path,
		})
	})
	return r, uatSvc, created.Token
}

func TestUserAccessTokenAuth_AllowsKeysAndGroups(t *testing.T) {
	user := &service.User{
		ID: 10, Email: "a@example.com", Role: "user",
		Status: service.StatusActive, Concurrency: 3, TokenVersion: 1,
	}
	router, _, token := newUATAuthEnv(t, user)

	for _, path := range []string{
		"/api/v1/keys",
		"/api/v1/keys/42",
		"/api/v1/groups/available",
		"/api/v1/groups/rates",
		"/api/v1/user/profile",
		"/api/v1/auth/me",
		"/api/v1/usage",
		"/api/v1/usage/1",
		"/api/v1/usage/stats",
		"/api/v1/usage/dashboard/stats",
		"/api/v1/subscriptions",
		"/api/v1/subscriptions/active",
		"/api/v1/subscriptions/progress",
		"/api/v1/subscriptions/summary",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, path)
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, float64(10), body["user_id"])
		require.Equal(t, AuthMethodUserAccessToken, body["auth_method"])
	}

	// POST keys allowed
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// POST usage dashboard batch query allowed (read-only query via POST)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/usage/dashboard/api-keys-usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestUserAccessTokenAuth_DeniesOutOfScope(t *testing.T) {
	user := &service.User{
		ID: 10, Email: "a@example.com", Role: "user",
		Status: service.StatusActive, Concurrency: 3, TokenVersion: 1,
	}
	router, _, token := newUATAuthEnv(t, user)

	// Mutations and unrelated resources stay denied
	type caseSpec struct {
		method string
		path   string
	}
	for _, tc := range []caseSpec{
		{http.MethodPut, "/api/v1/user/profile"},
		{http.MethodGet, "/api/v1/user/access-tokens"},
		{http.MethodPost, "/api/v1/user/access-tokens"},
		{http.MethodPost, "/api/v1/auth/revoke-all-sessions"},
		{http.MethodPost, "/api/v1/auth/bridge"},
		{http.MethodPost, "/api/v1/auth/oauth/bind-token"},
		{http.MethodPost, "/api/v1/subscriptions/1/reset-weekly-limit"},
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/channels/available"},
		{http.MethodGet, "/api/v1/user/aff"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code, tc.method+" "+tc.path)
		var body ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "ACCESS_TOKEN_SCOPE_DENIED", body.Code)
	}
}

func TestUserAccessTokenAuth_RevokedAndExpired(t *testing.T) {
	user := &service.User{
		ID: 10, Email: "a@example.com", Role: "user",
		Status: service.StatusActive, Concurrency: 3, TokenVersion: 1,
	}
	router, uatSvc, token := newUATAuthEnv(t, user)

	// revoke
	list, err := uatSvc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NoError(t, uatSvc.Revoke(context.Background(), 10, list[0].ID))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "TOKEN_REVOKED", body.Code)
}

func TestUserAccessTokenAuth_JWTStillWorks(t *testing.T) {
	user := &service.User{
		ID: 1, Email: "j@example.com", Role: "user",
		Status: service.StatusActive, Concurrency: 5, TokenVersion: 1,
	}
	// env with access token service nil is fine for JWT
	router, authSvc := newJWTTestEnv(map[int64]*service.User{1: user})
	token, err := authSvc.GenerateToken(context.Background(), user)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIsUserAccessTokenAllowedPath(t *testing.T) {
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/keys"))
	require.True(t, isUserAccessTokenAllowedPath("POST", "/api/v1/keys"))
	require.True(t, isUserAccessTokenAllowedPath("PUT", "/api/v1/keys/1"))
	require.True(t, isUserAccessTokenAllowedPath("DELETE", "/api/v1/keys/99"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/groups/available"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/groups/rates"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/user/profile"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/auth/me"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/usage"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/usage/stats"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/usage/dashboard/trend"))
	require.True(t, isUserAccessTokenAllowedPath("POST", "/api/v1/usage/dashboard/api-keys-usage"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/subscriptions"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/subscriptions/active"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/subscriptions/progress"))
	require.True(t, isUserAccessTokenAllowedPath("GET", "/api/v1/subscriptions/summary"))
	require.False(t, isUserAccessTokenAllowedPath("PUT", "/api/v1/user/profile"))
	require.False(t, isUserAccessTokenAllowedPath("POST", "/api/v1/user/access-tokens"))
	require.False(t, isUserAccessTokenAllowedPath("POST", "/api/v1/auth/revoke-all-sessions"))
	require.False(t, isUserAccessTokenAllowedPath("POST", "/api/v1/auth/bridge"))
	require.False(t, isUserAccessTokenAllowedPath("POST", "/api/v1/subscriptions/1/reset-weekly-limit"))
	require.False(t, isUserAccessTokenAllowedPath("GET", "/api/v1/keys/1/extra"))
}
