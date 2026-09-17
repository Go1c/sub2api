package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type trafficAdminStub struct {
	service.AdminService
	account       *service.Account
	reads, writes int
}

func (s *trafficAdminStub) GetAccount(context.Context, int64) (*service.Account, error) {
	s.reads++
	return s.account, nil
}
func (s *trafficAdminStub) UpdateAccountExtra(_ context.Context, _ int64, extra map[string]any) error {
	s.writes++
	for k, v := range extra {
		s.account.Extra[k] = v
	}
	return nil
}

func TestAccountTrafficHandlerIndependentSwitches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &trafficAdminStub{account: &service.Account{ID: 42, Concurrency: 128, Extra: map[string]any{"keep": "value"}}}
	h := &AccountHandler{adminService: repo}
	r := gin.New()
	r.GET("/:id/traffic-control", h.GetTrafficControl)
	r.PUT("/:id/traffic-control", h.UpdateTrafficControl)
	policy := service.DefaultAccountTrafficPolicy()
	policy.StrictRPMEnabled = true
	raw, _ := json.Marshal(policy)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/42/traffic-control", strings.NewReader(string(raw))))
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, 1, repo.writes)
	require.Equal(t, 128, repo.account.Concurrency)
	require.Equal(t, "value", repo.account.Extra["keep"])
	stored, err := service.ParseAccountTrafficPolicy(repo.account.Extra)
	require.NoError(t, err)
	require.True(t, stored.StrictRPMEnabled)
	require.False(t, stored.AdaptiveEnabled)
	policy.StrictRPMEnabled = false
	raw, _ = json.Marshal(policy)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/42/traffic-control", strings.NewReader(string(raw))))
	require.Equal(t, 200, w.Code)
	stored, err = service.ParseAccountTrafficPolicy(repo.account.Extra)
	require.NoError(t, err)
	require.False(t, stored.Enabled())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/42/traffic-control", nil))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"state_available":false`)
	require.Contains(t, w.Body.String(), `"hard_limit":128`)
}
