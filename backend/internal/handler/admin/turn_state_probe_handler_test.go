package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerMemoryTurnStateStore struct {
	mu      sync.Mutex
	tickets map[int64]service.TurnStateTicketRecord
	binds   map[string]string
	locks   map[int64]time.Time
	rpm     map[string]int
}

func newHandlerMemoryTurnStateStore() *handlerMemoryTurnStateStore {
	return &handlerMemoryTurnStateStore{
		tickets: map[int64]service.TurnStateTicketRecord{},
		binds:   map[string]string{},
		locks:   map[int64]time.Time{},
		rpm:     map[string]int{},
	}
}

func (s *handlerMemoryTurnStateStore) Get(_ context.Context, accountID int64) (*service.TurnStateTicketRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.tickets[accountID]
	if !ok {
		return nil, nil
	}
	cp := rec
	return &cp, nil
}

func (s *handlerMemoryTurnStateStore) Put(_ context.Context, rec service.TurnStateTicketRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = time.Now()
	}
	s.tickets[rec.AccountID] = rec
	return nil
}

func (s *handlerMemoryTurnStateStore) Delete(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tickets, accountID)
	return nil
}

func (s *handlerMemoryTurnStateStore) BindTurn(_ context.Context, _ int64, _, state string, _ time.Duration) (string, error) {
	return state, nil
}

func (s *handlerMemoryTurnStateStore) GetTurnBind(context.Context, int64, string) (string, error) {
	return "", nil
}

func (s *handlerMemoryTurnStateStore) TryLock(_ context.Context, accountID int64, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expiry, ok := s.locks[accountID]; ok && expiry.After(time.Now()) {
		return false, nil
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	s.locks[accountID] = time.Now().Add(ttl)
	return true, nil
}

func (s *handlerMemoryTurnStateStore) Unlock(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.locks, accountID)
	return nil
}

func (s *handlerMemoryTurnStateStore) AllowRPM(context.Context, string, int) (bool, error) {
	return true, nil
}

type handlerTurnStateSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *handlerTurnStateSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r *handlerTurnStateSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return v, nil
}
func (r *handlerTurnStateSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}
func (r *handlerTurnStateSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *handlerTurnStateSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *handlerTurnStateSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *handlerTurnStateSettingRepo) Delete(context.Context, string) error { return nil }

type handlerTurnStateAccountRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *handlerTurnStateAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, service.ErrAccountNotFound
	}
	cp := *r.account
	return &cp, nil
}

func (r *handlerTurnStateAccountRepo) ListByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	if r.account == nil || r.account.Platform != platform {
		return nil, nil
	}
	return []service.Account{*r.account}, nil
}

func (r *handlerTurnStateAccountRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if r.account == nil || r.account.ID != id {
		return service.ErrAccountNotFound
	}
	if r.account.Extra == nil {
		r.account.Extra = map[string]any{}
	}
	for k, v := range updates {
		r.account.Extra[k] = v
	}
	return nil
}

type handlerTurnStateHTTPUpstream struct {
	lastBody []byte
}

func (u *handlerTurnStateHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if req != nil && req.Body != nil {
		u.lastBody, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	header := http.Header{}
	header.Set("x-codex-turn-state", strings.Repeat("A", 160))
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"model":"gpt-6-astra"}}`,
		`data: {"type":"response.output_text.done","text":"21"}`,
		`data: [DONE]`,
		"",
	}, "\n")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (u *handlerTurnStateHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, conc int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, conc)
}

func newTurnStateProbeTestHandler(t *testing.T) (*TurnStateProbeHandler, *handlerMemoryTurnStateStore, *handlerTurnStateHTTPUpstream) {
	t.Helper()
	account := &service.Account{
		ID:       21,
		Name:     "codex-21",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "tok-secret",
		},
		Extra: map[string]any{
			service.TurnStateProbeExtraKey: map[string]any{"enabled": true},
		},
		Concurrency: 1,
	}
	tickets := newHandlerMemoryTurnStateStore()
	upstream := &handlerTurnStateHTTPUpstream{}
	svc := service.NewTurnStateProbeService(
		&handlerTurnStateSettingRepo{values: map[string]string{}},
		&handlerTurnStateAccountRepo{account: account},
		nil,
		tickets,
		upstream,
		nil,
	)
	return NewTurnStateProbeHandler(svc), tickets, upstream
}

func TestTurnStateProbeHandlerPolicyAndHarvest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tickets, upstream := newTurnStateProbeTestHandler(t)
	router := gin.New()
	router.GET("/admin/channels/turn-state-probe", h.Overview)
	router.PUT("/admin/channels/turn-state-probe", h.UpdatePolicy)
	router.GET("/admin/channels/turn-state-probe/accounts", h.Overview)
	router.POST("/admin/accounts/:id/turn-state-probe/enabled", h.SetEnabled)
	router.POST("/admin/accounts/:id/turn-state-probe/run", h.RunOne)
	router.DELETE("/admin/accounts/:id/turn-state-probe", h.Clear)

	policyBody := `{
		"enabled": true,
		"dynamic": {"host":"us.lajiaohttp.net:2000","username":"user1","password":"secret","region":"Random","session_minutes":5},
		"model":"gpt-6-astra",
		"length_filter_enabled": true,
		"min_state_length": 160,
		"answer":"21",
		"fuzzy_match": true,
		"recheck_minutes": 10,
		"rpm": 6
	}`
	put := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/channels/turn-state-probe", bytes.NewBufferString(policyBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(put, req)
	require.Equal(t, http.StatusOK, put.Code)
	var putPayload struct {
		Data service.TurnStateProbePolicy `json:"data"`
	}
	require.NoError(t, json.Unmarshal(put.Body.Bytes(), &putPayload))
	require.Equal(t, int64(1), putPayload.Data.Revision)
	require.Empty(t, putPayload.Data.Dynamic.Password)
	require.True(t, putPayload.Data.Dynamic.PasswordSet)

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/admin/channels/turn-state-probe", nil))
	require.Equal(t, http.StatusOK, get.Code)
	var overview struct {
		Data service.TurnStateProbeOverview `json:"data"`
	}
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &overview))
	require.Empty(t, overview.Data.Policy.Dynamic.Password)
	require.Len(t, overview.Data.Accounts, 1)
	require.Equal(t, int64(21), overview.Data.Accounts[0].AccountID)

	run := httptest.NewRecorder()
	router.ServeHTTP(run, httptest.NewRequest(http.MethodPost, "/admin/accounts/21/turn-state-probe/run", nil))
	require.Equal(t, http.StatusOK, run.Code, run.Body.String())
	require.NotContains(t, string(upstream.lastBody), "max_output_tokens")
	ticket, err := tickets.Get(context.Background(), 21)
	require.NoError(t, err)
	require.NotNil(t, ticket)
	require.Equal(t, "holding", ticket.Status)
	require.Equal(t, 160, ticket.StateLength)

	clear := httptest.NewRecorder()
	router.ServeHTTP(clear, httptest.NewRequest(http.MethodDelete, "/admin/accounts/21/turn-state-probe", nil))
	require.Equal(t, http.StatusOK, clear.Code)
	ticket, err = tickets.Get(context.Background(), 21)
	require.NoError(t, err)
	require.Nil(t, ticket)

	disable := httptest.NewRecorder()
	disReq := httptest.NewRequest(http.MethodPost, "/admin/accounts/21/turn-state-probe/enabled", bytes.NewBufferString(`{"enabled":false}`))
	disReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(disable, disReq)
	require.Equal(t, http.StatusOK, disable.Code)
}

func TestTurnStateProbeHandlerInvalidAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newTurnStateProbeTestHandler(t)
	router := gin.New()
	router.POST("/admin/accounts/:id/turn-state-probe/run", h.RunOne)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/accounts/abc/turn-state-probe/run", nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
