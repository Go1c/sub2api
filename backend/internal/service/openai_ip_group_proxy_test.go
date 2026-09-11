//go:build unit

package service

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const openAIIPGroupBindKeyPrefix = "openai:ip_group_bind:"

func openAIIPGroupBindKey(accountID int64, sessionHash string) string {
	return openAIIPGroupBindKeyPrefix + strconv.FormatInt(accountID, 10) + ":" + sessionHash
}

type memoryIPGroupBindStore struct {
	mu       sync.Mutex
	data     map[string]int64
	cooldown map[string]time.Time
	now      func() time.Time
}

func newMemoryIPGroupBindStore() *memoryIPGroupBindStore {
	return &memoryIPGroupBindStore{data: map[string]int64{}, cooldown: map[string]time.Time{}, now: time.Now}
}

func memoryIPGroupCooldownKey(accountID, proxyID int64) string {
	return "openai:ip_group_cooldown:" + strconv.FormatInt(accountID, 10) + ":" + strconv.FormatInt(proxyID, 10)
}

func (s *memoryIPGroupBindStore) GetBoundProxyID(_ context.Context, accountID int64, sessionHash string) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.data[openAIIPGroupBindKey(accountID, sessionHash)]
	return id, ok, nil
}

func (s *memoryIPGroupBindStore) SetBoundProxyID(_ context.Context, accountID int64, sessionHash string, proxyID int64, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[openAIIPGroupBindKey(accountID, sessionHash)] = proxyID
	return nil
}

func (s *memoryIPGroupBindStore) DeleteBoundProxyID(_ context.Context, accountID int64, sessionHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, openAIIPGroupBindKey(accountID, sessionHash))
	return nil
}

func (s *memoryIPGroupBindStore) MarkProxyCooldown(_ context.Context, accountID, proxyID int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if ttl <= 0 {
		ttl = openAIIPGroupTransientCooldown
	}
	s.cooldown[memoryIPGroupCooldownKey(accountID, proxyID)] = now.Add(ttl)
	return nil
}

func (s *memoryIPGroupBindStore) IsProxyCoolingDown(_ context.Context, accountID, proxyID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.cooldown[memoryIPGroupCooldownKey(accountID, proxyID)]
	if !ok {
		return false, nil
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if !now.Before(until) {
		delete(s.cooldown, memoryIPGroupCooldownKey(accountID, proxyID))
		return false, nil
	}
	return true, nil
}

func accountProxySlotKey(accountID, proxyID int64) string {
	return "conc:acct-proxy:" + strconv.FormatInt(accountID, 10) + ":" + strconv.FormatInt(proxyID, 10)
}

type memoryAccountProxySlots struct {
	mu    sync.Mutex
	count map[string]int
}

func newMemoryAccountProxySlots() *memoryAccountProxySlots {
	return &memoryAccountProxySlots{count: map[string]int{}}
}

func (s *memoryAccountProxySlots) AcquireAccountProxySlot(_ context.Context, accountID, proxyID int64, maxConcurrency int, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := accountProxySlotKey(accountID, proxyID)
	if s.count[key] >= maxConcurrency {
		return false, nil
	}
	s.count[key]++
	return true, nil
}

func (s *memoryAccountProxySlots) ReleaseAccountProxySlot(_ context.Context, accountID, proxyID int64, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := accountProxySlotKey(accountID, proxyID)
	if s.count[key] > 0 {
		s.count[key]--
	}
	return nil
}

func (s *memoryAccountProxySlots) GetAccountProxyConcurrency(_ context.Context, accountID, proxyID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count[accountProxySlotKey(accountID, proxyID)], nil
}

type staticProxyRepo struct {
	proxies map[int64]Proxy
}

func (s *staticProxyRepo) ListByIDs(_ context.Context, ids []int64) ([]Proxy, error) {
	out := make([]Proxy, 0, len(ids))
	for _, id := range ids {
		if p, ok := s.proxies[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func liveProxy(id int64, host string) Proxy {
	return Proxy{ID: id, Name: host, Protocol: "http", Host: host, Port: 8080, Status: StatusActive}
}

func testResolver(groups *proxyIPGroupRepoStub, proxies map[int64]Proxy, bind *memoryIPGroupBindStore, slots *memoryAccountProxySlots) *openAIIPGroupResolver {
	r := newOpenAIIPGroupResolver(groups, &staticProxyRepo{proxies: proxies}, bind, slots, time.Hour)
	r.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }
	return r
}

func TestResolveOpenAIAccountProxy_NoGroupUsesAccountProxy(t *testing.T) {
	proxy := liveProxy(4, "single.example")
	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		ProxyID:  &proxy.ID,
		Proxy:    &proxy,
	}
	resolved, err := resolveOpenAIAccountProxy(context.Background(), nil, account, "sess")
	require.NoError(t, err)
	require.Equal(t, proxy.URL(), resolved.Proxy.URL())
	require.Equal(t, accountDefaultProxyURL(account), resolved.Proxy.URL())
}

func TestResolveOpenAIAccountProxy_NonOpenAIIgnoresGroup(t *testing.T) {
	proxy := liveProxy(4, "single.example")
	gid := int64(1)
	account := &Account{
		ID:             1,
		Platform:       PlatformAnthropic,
		Type:           AccountTypeOAuth,
		ProxyID:        &proxy.ID,
		Proxy:          &proxy,
		ProxyIPGroupID: &gid,
	}
	resolved, err := resolveOpenAIAccountProxy(context.Background(), &openAIIPGroupResolver{}, account, "sess")
	require.NoError(t, err)
	require.Equal(t, proxy.URL(), resolved.Proxy.URL())
}

func TestResolveOpenAIAccountProxy_StickySameSession(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 10,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, bind, slots)
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)
	second, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, first.Proxy.ID, second.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_NewSessionMovesWhenFull(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 1,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, bind, slots)
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)
	// Keep the first slot occupied.
	next, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-2")
	require.NoError(t, err)
	require.Equal(t, int64(3), next.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_SlotFullDoesNotRebind(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 1,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, bind, slots)
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)
	again, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), again.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_DeadBoundRebinds(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 10,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, bind, slots)
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)

	dead := proxies[2]
	dead.Status = StatusDisabled
	proxies[2] = dead
	again, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), again.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_RemovedFromGroupRebinds(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 10,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	resolver := testResolver(groups, proxies, bind, newMemoryAccountProxySlots())
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)

	require.NoError(t, groups.SetMembers(context.Background(), 1, []int64{3}))
	again, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), again.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_SharedIPIndependentCaps(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 1,
		ProxyIDs:         []int64{2},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "fr.example")}
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, newMemoryIPGroupBindStore(), slots)
	a := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	b := &Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, a, "a-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)
	second, err := resolveOpenAIAccountProxy(context.Background(), resolver, b, "b-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), second.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_ExpiredIsDead(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 10,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	expired := liveProxy(2, "a.example")
	past := time.Unix(1_000, 0).UTC()
	expired.ExpiresAt = &past
	proxies := map[int64]Proxy{2: expired, 3: liveProxy(3, "b.example")}
	resolver := testResolver(groups, proxies, newMemoryIPGroupBindStore(), newMemoryAccountProxySlots())
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	resolved, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), resolved.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_TriedIPRebinds(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 10,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	bind := newMemoryIPGroupBindStore()
	resolver := testResolver(groups, proxies, bind, newMemoryAccountProxySlots())
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	ctx := withOpenAIIPGroupSession(context.Background(), "conv-1")

	first, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)

	st := openAIIPGroupRetryStateFrom(ctx)
	require.NotNil(t, st)
	st.markTried(2)
	require.NoError(t, bind.DeleteBoundProxyID(ctx, account.ID, "conv-1"))

	again, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), again.Proxy.ID)
}

func TestResolveOpenAIAccountProxy_AllSlotsFullReturnsError(t *testing.T) {
	groups := newProxyIPGroupRepoStub()
	require.NoError(t, groups.Create(context.Background(), &ProxyIPGroup{
		Name:             "eu",
		PerIPConcurrency: 1,
		ProxyIDs:         []int64{2, 3},
	}))
	gid := int64(1)
	proxies := map[int64]Proxy{2: liveProxy(2, "a.example"), 3: liveProxy(3, "b.example")}
	slots := newMemoryAccountProxySlots()
	resolver := testResolver(groups, proxies, newMemoryIPGroupBindStore(), slots)
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}

	first, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)
	second, err := resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-2")
	require.NoError(t, err)
	require.Equal(t, int64(3), second.Proxy.ID)

	_, err = resolveOpenAIAccountProxy(context.Background(), resolver, account, "conv-3")
	require.ErrorIs(t, err, errOpenAIIPGroupNoProxy)
}
