package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const openAIIPGroupBindKeyPrefix = "openai:ip_group_bind:"

// OpenAIIPGroupBindStore persists conversation -> proxy bindings for an account.
type OpenAIIPGroupBindStore interface {
	GetBoundProxyID(ctx context.Context, accountID int64, sessionHash string) (int64, bool, error)
	SetBoundProxyID(ctx context.Context, accountID int64, sessionHash string, proxyID int64, ttl time.Duration) error
}

// AccountProxySlotCache occupies per-account-per-proxy concurrency slots.
type AccountProxySlotCache interface {
	AcquireAccountProxySlot(ctx context.Context, accountID, proxyID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseAccountProxySlot(ctx context.Context, accountID, proxyID int64, requestID string) error
	GetAccountProxyConcurrency(ctx context.Context, accountID, proxyID int64) (int, error)
}

type resolvedOpenAIProxy struct {
	Proxy   *Proxy
	Release func()
}

type proxyByIDsReader interface {
	ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error)
}

type openAIIPGroupResolver struct {
	groups  ProxyIPGroupRepository
	proxies proxyByIDsReader
	bind    OpenAIIPGroupBindStore
	slots   AccountProxySlotCache
	now     func() time.Time
	ttl     time.Duration
}

func newOpenAIIPGroupResolver(
	groups ProxyIPGroupRepository,
	proxies proxyByIDsReader,
	bind OpenAIIPGroupBindStore,
	slots AccountProxySlotCache,
	ttl time.Duration,
) *openAIIPGroupResolver {
	if ttl <= 0 {
		ttl = openaiStickySessionTTL
	}
	return &openAIIPGroupResolver{
		groups:  groups,
		proxies: proxies,
		bind:    bind,
		slots:   slots,
		now:     time.Now,
		ttl:     ttl,
	}
}

func accountDefaultProxyURL(account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func accountUsesOpenAIIPGroup(account *Account) bool {
	return account != nil && account.IsOpenAIOAuth() && account.ProxyIPGroupID != nil && *account.ProxyIPGroupID > 0
}

func resolveOpenAIAccountProxy(ctx context.Context, resolver *openAIIPGroupResolver, account *Account, sessionHash string) (*resolvedOpenAIProxy, error) {
	if account == nil {
		return &resolvedOpenAIProxy{Release: func() {}}, nil
	}
	if resolver == nil || !accountUsesOpenAIIPGroup(account) {
		return &resolvedOpenAIProxy{Proxy: account.Proxy, Release: func() {}}, nil
	}
	return resolver.resolve(ctx, account, sessionHash)
}

func (r *openAIIPGroupResolver) resolve(ctx context.Context, account *Account, sessionHash string) (*resolvedOpenAIProxy, error) {
	group, err := r.groups.GetByID(ctx, *account.ProxyIPGroupID)
	if err != nil {
		return nil, err
	}
	members, err := r.loadLiveMembers(ctx, group)
	if err != nil {
		return nil, err
	}
	occupySlot := sessionHash != ""
	if sessionHash != "" && r.bind != nil {
		if boundID, ok, err := r.bind.GetBoundProxyID(ctx, account.ID, sessionHash); err != nil {
			return nil, err
		} else if ok {
			if bound, live := members[boundID]; live {
				release, _, err := r.tryOccupy(ctx, account.ID, bound.ID, group.PerIPConcurrency, occupySlot)
				if err != nil {
					return nil, err
				}
				// Slot full still keeps the bound IP (no rebind).
				return &resolvedOpenAIProxy{Proxy: bound, Release: release}, nil
			}
			// Bound proxy is dead: pick another live member with capacity and rebind.
			if next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, occupySlot); err != nil {
				return nil, err
			} else if next != nil {
				if err := r.bind.SetBoundProxyID(ctx, account.ID, sessionHash, next.ID, r.ttl); err != nil {
					release()
					return nil, err
				}
				return &resolvedOpenAIProxy{Proxy: next, Release: release}, nil
			}
			return &resolvedOpenAIProxy{Release: func() {}}, nil
		}
	}

	if sessionHash == "" {
		// Quota / OAuth: any live member, no conversation bind.
		if next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, false); err != nil {
			return nil, err
		} else if next != nil {
			return &resolvedOpenAIProxy{Proxy: next, Release: release}, nil
		}
		return &resolvedOpenAIProxy{Release: func() {}}, nil
	}

	next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, occupySlot)
	if err != nil {
		return nil, err
	}
	if next == nil {
		return &resolvedOpenAIProxy{Release: func() {}}, nil
	}
	if r.bind != nil {
		if err := r.bind.SetBoundProxyID(ctx, account.ID, sessionHash, next.ID, r.ttl); err != nil {
			release()
			return nil, err
		}
	}
	return &resolvedOpenAIProxy{Proxy: next, Release: release}, nil
}

func (r *openAIIPGroupResolver) loadLiveMembers(ctx context.Context, group *ProxyIPGroup) (map[int64]*Proxy, error) {
	if len(group.ProxyIDs) == 0 {
		return map[int64]*Proxy{}, nil
	}
	proxies, err := r.proxies.ListByIDs(ctx, group.ProxyIDs)
	if err != nil {
		return nil, err
	}
	now := r.now()
	live := make(map[int64]*Proxy, len(proxies))
	for i := range proxies {
		p := proxies[i]
		if proxyIsLive(&p, now) {
			cp := p
			live[p.ID] = &cp
		}
	}
	return live, nil
}

func (r *openAIIPGroupResolver) pickAndOccupy(ctx context.Context, accountID int64, group *ProxyIPGroup, members map[int64]*Proxy, occupySlot bool) (*Proxy, func(), error) {
	ids := make([]int64, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		release, ok, err := r.tryOccupy(ctx, accountID, id, group.PerIPConcurrency, occupySlot)
		if err != nil {
			return nil, nil, err
		}
		if !occupySlot || ok {
			return members[id], release, nil
		}
		release()
	}
	return nil, func() {}, nil
}

func (r *openAIIPGroupResolver) tryOccupy(ctx context.Context, accountID, proxyID int64, max int, occupy bool) (func(), bool, error) {
	noop := func() {}
	if !occupy || r.slots == nil || max <= 0 {
		return noop, true, nil
	}
	reqID := generateRequestID()
	ok, err := r.slots.AcquireAccountProxySlot(ctx, accountID, proxyID, max, reqID)
	if err != nil {
		return noop, false, err
	}
	if !ok {
		return noop, false, nil
	}
	return func() {
		_ = r.slots.ReleaseAccountProxySlot(context.Background(), accountID, proxyID, reqID)
	}, true, nil
}

func openAIIPGroupBindKey(accountID int64, sessionHash string) string {
	return openAIIPGroupBindKeyPrefix + strconv.FormatInt(accountID, 10) + ":" + sessionHash
}

func (s *OpenAIGatewayService) resolveOpenAIAccountProxy(ctx context.Context, account *Account, sessionHash string) (*resolvedOpenAIProxy, error) {
	return resolveOpenAIAccountProxy(ctx, s.ipGroupResolver, account, sessionHash)
}

func (s *OpenAIGatewayService) resolveOpenAIAccountProxyURL(ctx context.Context, account *Account, sessionHash string) (string, func(), error) {
	resolved, err := s.resolveOpenAIAccountProxy(ctx, account, sessionHash)
	if err != nil {
		return "", func() {}, err
	}
	if resolved == nil {
		return "", func() {}, nil
	}
	if resolved.Release == nil {
		resolved.Release = func() {}
	}
	if resolved.Proxy == nil {
		return "", resolved.Release, nil
	}
	return resolved.Proxy.URL(), resolved.Release, nil
}

func resolveOpenAIProxyURL(ctx context.Context, resolver *openAIIPGroupResolver, account *Account, sessionHash string) (string, func(), error) {
	resolved, err := resolveOpenAIAccountProxy(ctx, resolver, account, sessionHash)
	if err != nil {
		return "", func() {}, err
	}
	if resolved == nil || resolved.Proxy == nil {
		release := func() {}
		if resolved != nil && resolved.Release != nil {
			release = resolved.Release
		}
		return "", release, nil
	}
	if resolved.Release == nil {
		resolved.Release = func() {}
	}
	return resolved.Proxy.URL(), resolved.Release, nil
}

func (s *OpenAIGatewayService) SetIPGroupResolver(resolver *openAIIPGroupResolver) {
	if s == nil {
		return
	}
	s.ipGroupResolver = resolver
}

func (s *OpenAIGatewayService) IPGroupResolver() *openAIIPGroupResolver {
	if s == nil {
		return nil
	}
	return s.ipGroupResolver
}

func (s *OpenAIGatewayService) mustOpenAIAccountProxyURL(ctx context.Context, account *Account, sessionHash string) (string, func()) {
	url, release, err := s.resolveOpenAIAccountProxyURL(ctx, account, sessionHash)
	if err != nil {
		if release != nil {
			release()
		}
		return accountDefaultProxyURL(account), func() {}
	}
	if release == nil {
		release = func() {}
	}
	return url, release
}

func (s *OpenAIGatewayService) lookupOpenAIProxyURL(ctx context.Context, c *gin.Context, account *Account, body []byte) (string, func()) {
	sessionHash := ""
	if s != nil && c != nil {
		sessionHash = s.GenerateSessionHash(c, body)
	}
	return s.mustOpenAIAccountProxyURL(ctx, account, sessionHash)
}

func (s *ConcurrencyService) AcquireAccountProxySlot(ctx context.Context, accountID, proxyID int64, maxConcurrency int) (*AcquireResult, error) {
	if s == nil || maxConcurrency <= 0 {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	cache, ok := s.cache.(AccountProxySlotCache)
	if !ok {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	requestID := generateRequestID()
	acquired, err := cache.AcquireAccountProxySlot(ctx, accountID, proxyID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return &AcquireResult{Acquired: false}, nil
	}
	return &AcquireResult{
		Acquired: true,
		ReleaseFunc: func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := cache.ReleaseAccountProxySlot(bgCtx, accountID, proxyID, requestID); err != nil {
				_ = err
			}
		},
	}, nil
}

func accountProxySlotKey(accountID, proxyID int64) string {
	return fmt.Sprintf("conc:acct-proxy:%d:%d", accountID, proxyID)
}

type redisOpenAIIPGroupBindStore struct {
	rdb *redis.Client
}

func NewRedisOpenAIIPGroupBindStore(rdb *redis.Client) OpenAIIPGroupBindStore {
	if rdb == nil {
		return nil
	}
	return &redisOpenAIIPGroupBindStore{rdb: rdb}
}

func (s *redisOpenAIIPGroupBindStore) GetBoundProxyID(ctx context.Context, accountID int64, sessionHash string) (int64, bool, error) {
	if s == nil || s.rdb == nil || sessionHash == "" {
		return 0, false, nil
	}
	val, err := s.rdb.Get(ctx, openAIIPGroupBindKey(accountID, sessionHash)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil || id <= 0 {
		return 0, false, nil
	}
	return id, true, nil
}

func (s *redisOpenAIIPGroupBindStore) SetBoundProxyID(ctx context.Context, accountID int64, sessionHash string, proxyID int64, ttl time.Duration) error {
	if s == nil || s.rdb == nil || sessionHash == "" || proxyID <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = openaiStickySessionTTL
	}
	return s.rdb.Set(ctx, openAIIPGroupBindKey(accountID, sessionHash), strconv.FormatInt(proxyID, 10), ttl).Err()
}
