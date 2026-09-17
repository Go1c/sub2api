package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type memoryTurnStateTicketStore struct {
	mu      sync.Mutex
	tickets map[int64]TurnStateTicketRecord
	binds   map[string]string
	locks   map[int64]time.Time
	rpm     map[string]int
}

func newMemoryTurnStateTicketStore() *memoryTurnStateTicketStore {
	return &memoryTurnStateTicketStore{
		tickets: make(map[int64]TurnStateTicketRecord),
		binds:   make(map[string]string),
		locks:   make(map[int64]time.Time),
		rpm:     make(map[string]int),
	}
}

func (s *memoryTurnStateTicketStore) Get(_ context.Context, accountID int64) (*TurnStateTicketRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.tickets[accountID]
	if !ok {
		return nil, nil
	}
	cp := rec
	return &cp, nil
}

func (s *memoryTurnStateTicketStore) Put(_ context.Context, rec TurnStateTicketRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.StateHash == "" {
		rec.StateHash = TurnStateProbeStateHash(rec.State)
	}
	if rec.StateLength == 0 {
		rec.StateLength = len(strings.TrimSpace(rec.State))
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = time.Now()
	}
	s.tickets[rec.AccountID] = rec
	return nil
}

func (s *memoryTurnStateTicketStore) Delete(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tickets, accountID)
	return nil
}

func (s *memoryTurnStateTicketStore) BindTurn(_ context.Context, accountID int64, turnKey, state string, _ time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	turnKey = strings.TrimSpace(turnKey)
	state = strings.TrimSpace(state)
	if turnKey == "" || state == "" {
		return state, nil
	}
	key := strconv.FormatInt(accountID, 10) + ":" + turnKey
	if existing, ok := s.binds[key]; ok && existing != "" {
		return existing, nil
	}
	s.binds[key] = state
	return state, nil
}

func (s *memoryTurnStateTicketStore) GetTurnBind(_ context.Context, accountID int64, turnKey string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.binds[strconv.FormatInt(accountID, 10)+":"+strings.TrimSpace(turnKey)], nil
}

func (s *memoryTurnStateTicketStore) TryLock(_ context.Context, accountID int64, ttl time.Duration) (bool, error) {
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

func (s *memoryTurnStateTicketStore) Unlock(_ context.Context, accountID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.locks, accountID)
	return nil
}

func (s *memoryTurnStateTicketStore) AllowRPM(_ context.Context, bucket string, rpm int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rpm < 1 {
		rpm = 1
	}
	s.rpm[bucket]++
	return s.rpm[bucket] <= rpm, nil
}

type turnStateProbeSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newTurnStateProbeSettingRepo() *turnStateProbeSettingRepo {
	return &turnStateProbeSettingRepo{values: map[string]string{}}
}

func (r *turnStateProbeSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (r *turnStateProbeSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (r *turnStateProbeSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *turnStateProbeSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *turnStateProbeSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (r *turnStateProbeSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *turnStateProbeSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

type turnStateProbeAccountRepo struct {
	AccountRepository
	mu       sync.Mutex
	accounts map[int64]*Account
}

func newTurnStateProbeAccountRepo(accounts ...*Account) *turnStateProbeAccountRepo {
	r := &turnStateProbeAccountRepo{accounts: map[int64]*Account{}}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		cp := *account
		r.accounts[account.ID] = &cp
	}
	return r
}

func (r *turnStateProbeAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account := r.accounts[id]
	if account == nil {
		return nil, ErrAccountNotFound
	}
	cp := *account
	return &cp, nil
}

func (r *turnStateProbeAccountRepo) ListByPlatform(_ context.Context, platform string) ([]Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform != platform {
			continue
		}
		out = append(out, *account)
	}
	return out, nil
}

func (r *turnStateProbeAccountRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	account := r.accounts[id]
	if account == nil {
		return ErrAccountNotFound
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	for k, v := range updates {
		account.Extra[k] = v
	}
	return nil
}

type recordingTurnStateHTTPUpstream struct {
	handler   http.Handler
	mu        sync.Mutex
	lastReq   *http.Request
	lastBody  []byte
	lastProxy string
}

func (u *recordingTurnStateHTTPUpstream) Do(req *http.Request, proxyURL string, _ int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	var body []byte
	if req != nil && req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	u.lastReq = req
	u.lastBody = append([]byte(nil), body...)
	u.lastProxy = proxyURL
	if u.handler == nil {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}
	inner := req.Clone(req.Context())
	inner.Body = io.NopCloser(bytes.NewReader(body))
	rec := httptest.NewRecorder()
	u.handler.ServeHTTP(rec, inner)
	return rec.Result(), nil
}

func (u *recordingTurnStateHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func newOAuthProbeAccount(id int64, enabled bool) *Account {
	return &Account{
		ID:       id,
		Name:     "codex-" + strconv.FormatInt(id, 10),
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "tok-secret",
		},
		Extra: map[string]any{
			TurnStateProbeExtraKey: map[string]any{"enabled": enabled},
		},
		Concurrency: 1,
	}
}

func newTurnStateProbeServiceForTest(t *testing.T, account *Account, tickets TurnStateTicketStore, upstream HTTPUpstream) *TurnStateProbeService {
	t.Helper()
	if tickets == nil {
		tickets = newMemoryTurnStateTicketStore()
	}
	return NewTurnStateProbeService(
		newTurnStateProbeSettingRepo(),
		newTurnStateProbeAccountRepo(account),
		nil,
		tickets,
		upstream,
		nil,
	)
}

func TestTurnStateProbeSavePolicyIncrementsRevision(t *testing.T) {
	svc := newTurnStateProbeServiceForTest(t, newOAuthProbeAccount(1, true), nil, nil)
	first := DefaultTurnStateProbePolicy()
	first.Enabled = true
	first.Dynamic.Host = "us.lajiaohttp.net:2000"
	first.Dynamic.Username = "user1"
	first.Dynamic.Password = "secret"
	saved, err := svc.SavePolicy(context.Background(), first)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.Revision)
	require.Empty(t, saved.Dynamic.Password)
	require.True(t, saved.Dynamic.PasswordSet)

	again, err := svc.SavePolicy(context.Background(), saved)
	require.NoError(t, err)
	require.Equal(t, int64(1), again.Revision)

	saved.Model = "gpt-6-astra-high"
	bumped, err := svc.SavePolicy(context.Background(), saved)
	require.NoError(t, err)
	require.Equal(t, int64(2), bumped.Revision)

	stored, err := svc.GetPolicy(context.Background())
	require.NoError(t, err)
	require.Equal(t, "secret", stored.Dynamic.Password)
	require.Equal(t, int64(2), stored.Revision)
}

func TestTurnStateProbeGetPolicyPublicRedactsPassword(t *testing.T) {
	svc := newTurnStateProbeServiceForTest(t, newOAuthProbeAccount(1, true), nil, nil)
	next := DefaultTurnStateProbePolicy()
	next.Enabled = true
	next.Dynamic.Host = "proxy.example:2000"
	next.Dynamic.Username = "user1"
	next.Dynamic.Password = "secret"
	_, err := svc.SavePolicy(context.Background(), next)
	require.NoError(t, err)
	pub, err := svc.GetPolicyPublic(context.Background())
	require.NoError(t, err)
	require.Empty(t, pub.Dynamic.Password)
	require.True(t, pub.Dynamic.PasswordSet)
}

func TestTurnStateProbeBindCurrentNoopsIfDisabled(t *testing.T) {
	tickets := newMemoryTurnStateTicketStore()
	require.NoError(t, tickets.Put(context.Background(), TurnStateTicketRecord{
		AccountID:      7,
		State:          stringsRepeat("A", 160),
		Status:         turnStateProbeStatusHolding,
		PolicyRevision: 1,
	}))
	account := newOAuthProbeAccount(7, false)
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	policy.Dynamic.Host = "proxy.example:2000"
	policy.Dynamic.Username = "user1"
	policy.Dynamic.Password = "secret"
	_, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)

	state, ok := svc.BindCurrent(context.Background(), account, "", "", "gpt-6-astra")
	require.False(t, ok)
	require.Empty(t, state)
	require.False(t, svc.HasHolding(context.Background(), account))

	account.Extra[TurnStateProbeExtraKey] = map[string]any{"enabled": true}
	disabled := DefaultTurnStateProbePolicy()
	disabled.Enabled = false
	disabled.Dynamic.Host = "proxy.example:2000"
	disabled.Dynamic.Username = "user1"
	disabled.Dynamic.Password = "secret"
	_, err = svc.SavePolicy(context.Background(), disabled)
	require.NoError(t, err)
	state, ok = svc.BindCurrent(context.Background(), account, "", "", "gpt-6-astra")
	require.False(t, ok)
	require.Empty(t, state)
}

func TestTurnStateProbeBindCurrentReturnsTicketAndPinsTurn(t *testing.T) {
	tickets := newMemoryTurnStateTicketStore()
	first := stringsRepeat("A", 160)
	require.NoError(t, tickets.Put(context.Background(), TurnStateTicketRecord{
		AccountID:      9,
		State:          first,
		Status:         turnStateProbeStatusHolding,
		PolicyRevision: 1,
	}))
	account := newOAuthProbeAccount(9, true)
	svc := newTurnStateProbeServiceForTest(t, account, tickets, nil)
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	policy.Dynamic.Host = "proxy.example:2000"
	policy.Dynamic.Username = "user1"
	policy.Dynamic.Password = "secret"
	saved, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)

	ticket, err := tickets.Get(context.Background(), 9)
	require.NoError(t, err)
	ticket.PolicyRevision = saved.Revision
	require.NoError(t, tickets.Put(context.Background(), *ticket))

	state, ok := svc.BindCurrent(context.Background(), account, "ns", "", "gpt-6-astra")
	require.True(t, ok)
	require.Equal(t, first, state)
	require.True(t, svc.HasHolding(context.Background(), account))

	bound, ok := svc.BindCurrent(context.Background(), account, "ns", "turn-1", "gpt-6-astra")
	require.True(t, ok)
	require.Equal(t, first, bound)

	second := stringsRepeat("B", 160)
	ticket.State = second
	ticket.StateHash = TurnStateProbeStateHash(second)
	require.NoError(t, tickets.Put(context.Background(), *ticket))
	pinned, ok := svc.BindCurrent(context.Background(), account, "ns", "turn-1", "gpt-6-astra")
	require.True(t, ok)
	require.Equal(t, first, pinned)
}

func TestTurnStateProbeAccountHarvestStoresHoldingTicket(t *testing.T) {
	stateBlob := stringsRepeat("A", 160)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NotContains(t, string(body), "max_output_tokens")
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		_, hasMax := payload["max_output_tokens"]
		require.False(t, hasMax)
		require.Equal(t, "gpt-6-astra", payload["model"])
		w.Header().Set(openAICodexTurnStateHeader, stateBlob)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, strings.Join([]string{
			`data: {"type":"response.created","response":{"model":"gpt-6-astra"}}`,
			`data: {"type":"response.output_text.done","text":"21"}`,
			`data: [DONE]`,
			"",
		}, "\n"))
	})
	upstream := &recordingTurnStateHTTPUpstream{handler: handler}
	tickets := newMemoryTurnStateTicketStore()
	account := newOAuthProbeAccount(11, true)
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	policy.Dynamic.Host = "us.lajiaohttp.net:2000"
	policy.Dynamic.Username = "user1"
	policy.Dynamic.Password = "secret"
	policy.LengthFilterEnabled = true
	policy.MinStateLength = 160
	saved, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)

	require.NoError(t, svc.ProbeAccount(context.Background(), account.ID))
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, http.MethodPost, upstream.lastReq.Method)
	require.Equal(t, chatgptCodexURL, upstream.lastReq.URL.String())
	require.Empty(t, upstream.lastReq.Header.Get("OpenAI-Beta"))
	require.NotContains(t, string(upstream.lastBody), "max_output_tokens")
	require.Contains(t, upstream.lastProxy, "user1-region-")
	require.Contains(t, upstream.lastProxy, "-sid-")
	require.NotContains(t, upstream.lastProxy, "secret?")

	ticket, err := tickets.Get(context.Background(), account.ID)
	require.NoError(t, err)
	require.NotNil(t, ticket)
	require.Equal(t, turnStateProbeStatusHolding, ticket.Status)
	require.Equal(t, stateBlob, ticket.State)
	require.Equal(t, TurnStateProbeStateHash(stateBlob), ticket.StateHash)
	require.Equal(t, 160, ticket.StateLength)
	require.Equal(t, "gpt-6-astra", ticket.Model)
	require.Equal(t, saved.Revision, ticket.PolicyRevision)
	require.False(t, ticket.RecheckAt.IsZero())
	require.Empty(t, ticket.LastError)
	require.NotContains(t, ticket.Summary().StateHash, stateBlob)
}

func enableTurnStateProbePolicy(t *testing.T, svc *TurnStateProbeService) TurnStateProbePolicy {
	t.Helper()
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	policy.Dynamic.Host = "us.lajiaohttp.net:2000"
	policy.Dynamic.Username = "user1"
	policy.Dynamic.Password = "secret"
	policy.RPM = 60
	saved, err := svc.SavePolicy(context.Background(), policy)
	require.NoError(t, err)
	return saved
}

func turnStateProbeSuccessSSE(state string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(openAICodexTurnStateHeader, state)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, strings.Join([]string{
			`data: {"type":"response.created","response":{"model":"gpt-6-astra"}}`,
			`data: {"type":"response.output_text.done","text":"21"}`,
			`data: [DONE]`,
			"",
		}, "\n"))
	}
}

type concurrentTurnStateHTTPUpstream struct {
	handler  http.Handler
	inflight atomic.Int64
	max      atomic.Int64
	calls    atomic.Int64
}

func (u *concurrentTurnStateHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	n := u.inflight.Add(1)
	u.calls.Add(1)
	for {
		old := u.max.Load()
		if n <= old || u.max.CompareAndSwap(old, n) {
			break
		}
	}
	defer u.inflight.Add(-1)
	if u.handler == nil {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}
	inner := req.Clone(req.Context())
	rec := httptest.NewRecorder()
	u.handler.ServeHTTP(rec, inner)
	return rec.Result(), nil
}

func (u *concurrentTurnStateHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, conc int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, conc)
}

func TestTurnStateProbeMarksRunningWhileHarvesting(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	upstream := &concurrentTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		turnStateProbeSuccessSSE(stringsRepeat("A", 160)).ServeHTTP(w, r)
	})}
	tickets := newMemoryTurnStateTicketStore()
	account := newOAuthProbeAccount(31, true)
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	enableTurnStateProbePolicy(t, svc)

	errCh := make(chan error, 1)
	go func() { errCh <- svc.ProbeAccount(context.Background(), account.ID) }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("harvest did not start")
	}
	out, err := svc.GetOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Accounts, 1)
	require.Equal(t, turnStateProbeStatusRunning, out.Accounts[0].Status)

	close(release)
	require.NoError(t, <-errCh)
	ticket, err := tickets.Get(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, turnStateProbeStatusHolding, ticket.Status)
}

func TestTurnStateProbeGivesEachAccountTenAttemptsThenMovesOn(t *testing.T) {
	upstream := &concurrentTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})}
	first := newOAuthProbeAccount(41, true)
	second := newOAuthProbeAccount(42, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := NewTurnStateProbeService(
		newTurnStateProbeSettingRepo(),
		newTurnStateProbeAccountRepo(first, second),
		nil,
		tickets,
		upstream,
		nil,
	)
	enableTurnStateProbePolicy(t, svc)

	require.Error(t, svc.ProbeAccount(context.Background(), first.ID))
	ticket, err := tickets.Get(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, turnStateProbeStatusCooldown, ticket.Status)
	require.Equal(t, turnStateProbeMaxAttempts, ticket.Attempts)
	require.False(t, ticket.RecheckAt.IsZero())
	require.Equal(t, int64(turnStateProbeMaxAttempts), upstream.calls.Load())

	require.NoError(t, svc.HarvestDue(context.Background()))
	require.Eventually(t, func() bool {
		secondTicket, getErr := tickets.Get(context.Background(), second.ID)
		return getErr == nil && secondTicket != nil && secondTicket.Attempts == turnStateProbeMaxAttempts && secondTicket.Status == turnStateProbeStatusCooldown
	}, 3*time.Second, 20*time.Millisecond)
	require.Equal(t, int64(turnStateProbeMaxAttempts*2), upstream.calls.Load())

	firstAgain, err := tickets.Get(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, turnStateProbeMaxAttempts, firstAgain.Attempts)
}

func TestTurnStateProbeHarvestDueRunsTenAccountsInParallel(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 16)
	upstream := &concurrentTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-release
		turnStateProbeSuccessSSE(stringsRepeat("A", 160)).ServeHTTP(w, r)
	})}
	accounts := make([]*Account, 0, 12)
	for i := 1; i <= 12; i++ {
		accounts = append(accounts, newOAuthProbeAccount(int64(50+i), true))
	}
	tickets := newMemoryTurnStateTicketStore()
	svc := NewTurnStateProbeService(
		newTurnStateProbeSettingRepo(),
		newTurnStateProbeAccountRepo(accounts...),
		nil,
		tickets,
		upstream,
		nil,
	)
	enableTurnStateProbePolicy(t, svc)

	done := make(chan error, 1)
	go func() { done <- svc.HarvestDue(context.Background()) }()

	deadline := time.After(2 * time.Second)
	for i := 0; i < turnStateProbeHarvestConcurrency; i++ {
		select {
		case <-started:
		case <-deadline:
			t.Fatalf("started %d workers, want %d", i, turnStateProbeHarvestConcurrency)
		}
	}
	select {
	case <-started:
		t.Fatal("started more than 10 harvests at once")
	case <-time.After(150 * time.Millisecond):
	}
	require.Equal(t, int64(turnStateProbeHarvestConcurrency), upstream.max.Load())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("HarvestDue should return after filling 10 slots")
	}

	close(release)
}

func TestTurnStateProbeRunOneResetsExhaustedAccount(t *testing.T) {
	stateBlob := stringsRepeat("A", 160)
	upstream := &concurrentTurnStateHTTPUpstream{handler: turnStateProbeSuccessSSE(stateBlob)}
	account := newOAuthProbeAccount(61, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	saved := enableTurnStateProbePolicy(t, svc)
	require.NoError(t, tickets.Put(context.Background(), TurnStateTicketRecord{
		AccountID:      account.ID,
		Status:         turnStateProbeStatusFailed,
		Attempts:       turnStateProbeMaxAttempts,
		PolicyRevision: saved.Revision,
	}))

	require.NoError(t, svc.HarvestDue(context.Background()))
	require.Equal(t, int64(0), upstream.calls.Load())

	require.NoError(t, svc.RunOne(context.Background(), account.ID))
	ticket, err := tickets.Get(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, turnStateProbeStatusHolding, ticket.Status)
	require.Equal(t, 1, ticket.Attempts)
}

func TestTurnStateProbeSkipsAccountOnUnauthorized(t *testing.T) {
	upstream := &concurrentTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})}
	first := newOAuthProbeAccount(71, true)
	second := newOAuthProbeAccount(72, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := NewTurnStateProbeService(
		newTurnStateProbeSettingRepo(),
		newTurnStateProbeAccountRepo(first, second),
		nil,
		tickets,
		upstream,
		nil,
	)
	enableTurnStateProbePolicy(t, svc)

	err := svc.ProbeAccount(context.Background(), first.ID)
	require.Error(t, err)
	ticket, getErr := tickets.Get(context.Background(), first.ID)
	require.NoError(t, getErr)
	require.Equal(t, turnStateProbeStatusSkipped, ticket.Status)
	require.Equal(t, 1, ticket.Attempts)
	require.Equal(t, int64(1), upstream.calls.Load())

	require.NoError(t, svc.HarvestDue(context.Background()))
	require.Eventually(t, func() bool {
		secondTicket, secondErr := tickets.Get(context.Background(), second.ID)
		return secondErr == nil && secondTicket != nil && secondTicket.Status == turnStateProbeStatusSkipped
	}, 3*time.Second, 20*time.Millisecond)
	require.Equal(t, int64(2), upstream.calls.Load())
	firstAgain, getErr := tickets.Get(context.Background(), first.ID)
	require.NoError(t, getErr)
	require.Equal(t, 1, firstAgain.Attempts)
	require.Equal(t, turnStateProbeStatusSkipped, firstAgain.Status)
}

func TestTurnStateProbeCooldownAfterTenFailuresThenRetries(t *testing.T) {
	stateBlob := stringsRepeat("A", 160)
	upstream := &concurrentTurnStateHTTPUpstream{handler: turnStateProbeSuccessSSE(stateBlob)}
	account := newOAuthProbeAccount(81, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	saved := enableTurnStateProbePolicy(t, svc)
	require.NoError(t, tickets.Put(context.Background(), TurnStateTicketRecord{
		AccountID:      account.ID,
		Status:         turnStateProbeStatusCooldown,
		Attempts:       turnStateProbeMaxAttempts,
		PolicyRevision: saved.Revision,
		RecheckAt:      time.Now().Add(time.Hour),
	}))

	require.NoError(t, svc.HarvestDue(context.Background()))
	require.Equal(t, int64(0), upstream.calls.Load())

	ticket, err := tickets.Get(context.Background(), account.ID)
	require.NoError(t, err)
	ticket.RecheckAt = time.Now().Add(-time.Second)
	require.NoError(t, tickets.Put(context.Background(), *ticket))

	require.NoError(t, svc.HarvestDue(context.Background()))
	require.Eventually(t, func() bool {
		got, getErr := tickets.Get(context.Background(), account.ID)
		return getErr == nil && got != nil && got.Status == turnStateProbeStatusHolding
	}, 3*time.Second, 20*time.Millisecond)
	require.Equal(t, int64(1), upstream.calls.Load())
}
