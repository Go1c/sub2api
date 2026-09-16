package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type memoryRequestHealthStore struct {
	mu       sync.Mutex
	events   map[string][]RequestHealthEvent
	current  map[string]int
	cooldown map[string]time.Time
}

func newMemoryRequestHealthStore() *memoryRequestHealthStore {
	return &memoryRequestHealthStore{
		events:   map[string][]RequestHealthEvent{},
		current:  map[string]int{},
		cooldown: map[string]time.Time{},
	}
}

func requestHealthMemoryKey(accountID, proxyID int64) string {
	if proxyID > 0 {
		return fmt.Sprintf("a:%d:p:%d", accountID, proxyID)
	}
	return fmt.Sprintf("a:%d", accountID)
}

func (s *memoryRequestHealthStore) Append(_ context.Context, ev RequestHealthEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.push(requestHealthMemoryKey(ev.AccountID, 0), ev)
	if ev.ProxyID > 0 {
		s.push(requestHealthMemoryKey(ev.AccountID, ev.ProxyID), ev)
	}
	return nil
}

func (s *memoryRequestHealthStore) push(key string, ev RequestHealthEvent) {
	list := append(s.events[key], ev)
	if len(list) > RequestHealthMaxEvents {
		list = list[len(list)-RequestHealthMaxEvents:]
	}
	s.events[key] = list
}

func (s *memoryRequestHealthStore) List(_ context.Context, accountID, proxyID int64, limit int) ([]RequestHealthEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.events[requestHealthMemoryKey(accountID, proxyID)]
	if limit > 0 && len(list) > limit {
		list = list[len(list)-limit:]
	}
	out := make([]RequestHealthEvent, len(list))
	copy(out, list)
	return out, nil
}

func (s *memoryRequestHealthStore) Runtime(_ context.Context, accountID, proxyID int64) (int, *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := requestHealthMemoryKey(accountID, proxyID)
	current := s.current[key]
	if until, ok := s.cooldown[key]; ok && until.After(time.Now()) {
		cp := until
		return current, &cp
	}
	return current, nil
}

type requestHealthDirStub struct {
	accounts []*Account
	groups   []ProxyIPGroup
	proxies  []Proxy
}

func (s *requestHealthDirStub) GetAccountsByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	want := map[int64]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]*Account, 0, len(ids))
	for _, acc := range s.accounts {
		if acc == nil {
			continue
		}
		if _, ok := want[acc.ID]; ok {
			out = append(out, acc)
		}
	}
	return out, nil
}

func (s *requestHealthDirStub) ListProxyIPGroups(context.Context) ([]ProxyIPGroup, error) {
	return s.groups, nil
}

func (s *requestHealthDirStub) GetProxiesByIDs(context.Context, []int64) ([]Proxy, error) {
	return s.proxies, nil
}

func TestClampRequestHealthWindow(t *testing.T) {
	require.Equal(t, 12, ClampRequestHealthWindow(0))
	require.Equal(t, 12, ClampRequestHealthWindow(9))
	require.Equal(t, 8, ClampRequestHealthWindow(8))
	require.Equal(t, 20, ClampRequestHealthWindow(20))
}

func TestMaskRequestHealthHost(t *testing.T) {
	require.Equal(t, "—", MaskRequestHealthHost(""))
	require.Equal(t, "1.2.*.*", MaskRequestHealthHost("1.2.3.4"))
	require.Equal(t, "1.2.*.*", MaskRequestHealthHost("socks5://user:pass@1.2.3.4:1080"))
	require.Equal(t, "proxy.***.com", MaskRequestHealthHost("proxy.example.com"))
}

func TestEgressProxyIDFromContextAndAccount(t *testing.T) {
	proxyID := int64(9)
	account := &Account{ProxyID: &proxyID}
	require.Equal(t, int64(9), EgressProxyIDFrom(context.Background(), account))

	ctx := context.WithValue(context.Background(), ctxkey.EgressProxyID, int64(44))
	require.Equal(t, int64(44), EgressProxyIDFrom(ctx, account))
}

func TestAccountRequestHealthServiceListsSingleIPOldestLeft(t *testing.T) {
	store := newMemoryRequestHealthStore()
	proxyID := int64(7)
	svc := NewAccountRequestHealthService(store, &requestHealthDirStub{
		accounts: []*Account{{ID: 11, Name: "claude-1", ProxyID: &proxyID, Concurrency: 3}},
	})

	require.NoError(t, svc.recordSync(context.Background(), RequestHealthRecordInput{
		AccountID: 11, ProxyID: 7, Slot: RequestHealthSlotOK, Model: "claude-sonnet", Endpoint: "/v1/messages",
	}))
	require.NoError(t, svc.recordSync(context.Background(), RequestHealthRecordInput{
		AccountID: 11, ProxyID: 7, Slot: RequestHealthSlotFail, StatusCode: 429, Message: "rate limited", Model: "claude-sonnet",
	}))

	items, err := svc.ListForAccounts(context.Background(), []int64{11}, 8)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "single", items[0].Mode)
	require.Len(t, items[0].Lines, 1)
	require.Equal(t, []string{"ok", "fail"}, outcomeSlots(items[0].Lines[0].Outcomes))
	require.True(t, items[0].Lines[0].RateLimited)
	require.False(t, items[0].Lines[0].Overloaded)
	require.Equal(t, 3, items[0].Lines[0].Max)
}

func TestAccountRequestHealthServiceListsIPGroupPerProxy(t *testing.T) {
	store := newMemoryRequestHealthStore()
	groupID := int64(3)
	store.current[requestHealthMemoryKey(22, 101)] = 2
	store.cooldown[requestHealthMemoryKey(22, 102)] = time.Now().Add(90 * time.Second)
	svc := NewAccountRequestHealthService(store, &requestHealthDirStub{
		accounts: []*Account{{ID: 22, Name: "codex-pool", ProxyIPGroupID: &groupID, Concurrency: 1}},
		groups:   []ProxyIPGroup{{ID: 3, Name: "france-pool", PerIPConcurrency: 10, ProxyIDs: []int64{101, 102}}},
		proxies:  []Proxy{{ID: 101, Host: "10.0.1.8"}, {ID: 102, Host: "10.0.1.9"}},
	})

	require.NoError(t, svc.recordSync(context.Background(), RequestHealthRecordInput{
		AccountID: 22, ProxyID: 101, Slot: RequestHealthSlotFail, StatusCode: 529, Message: "overloaded",
	}))

	items, err := svc.ListForAccounts(context.Background(), []int64{22}, 12)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "ip_group", items[0].Mode)
	require.Equal(t, "france-pool", items[0].IPGroupName)
	require.Len(t, items[0].Lines, 2)
	require.Equal(t, "10.0.*.*", items[0].Lines[0].IP)
	require.Equal(t, []string{"fail"}, outcomeSlots(items[0].Lines[0].Outcomes))
	require.True(t, items[0].Lines[0].Overloaded)
	require.Equal(t, 2, items[0].Lines[0].Current)
	require.Equal(t, 10, items[0].Lines[0].Max)
	require.Empty(t, items[0].Lines[1].Outcomes)
	require.NotNil(t, items[0].Lines[1].CooldownUntil)
}

func TestAccountRequestHealthServiceRecordNilSafe(t *testing.T) {
	var svc *AccountRequestHealthService
	svc.Record(context.Background(), RequestHealthRecordInput{AccountID: 1, Slot: RequestHealthSlotOK})
}

func outcomeSlots(items []RequestHealthOutcomeDTO) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Slot)
	}
	return out
}
