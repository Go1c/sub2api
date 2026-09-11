package service

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	openAIIPGroupTransientCooldown = 30 * time.Second
	ginKeyOpenAIIPGroupSession     = "openai_ip_group_session_hash"
)

// errOpenAIIPGroupNoProxy means the account is assigned an IP group but no
// live, non-cooling member can carry this conversation. Callers must fail
// closed instead of dialing OpenAI from the server's own IP.
var errOpenAIIPGroupNoProxy = errors.New("openai ip group has no available egress")

type openaiIPGroupSessionCtxKey struct{}
type openaiIPGroupRetryStateCtxKey struct{}

const openAIIPGroupPassLimit = 2

type openAIIPGroupRetryState struct {
	tried map[int64]struct{}
	pass  int
	size  int
	done  bool
}

func newOpenAIIPGroupRetryState() *openAIIPGroupRetryState {
	return &openAIIPGroupRetryState{tried: map[int64]struct{}{}, pass: 1}
}

func (st *openAIIPGroupRetryState) isTried(id int64) bool {
	if st == nil || id <= 0 {
		return false
	}
	_, ok := st.tried[id]
	return ok
}

func (st *openAIIPGroupRetryState) markTried(id int64) {
	if st == nil || id <= 0 {
		return
	}
	if st.tried == nil {
		st.tried = map[int64]struct{}{}
	}
	st.tried[id] = struct{}{}
}

func (st *openAIIPGroupRetryState) extraRetries() int {
	if st == nil || st.size <= 0 {
		return defaultPoolModeRetryCount
	}
	extra := openAIIPGroupPassLimit*st.size - 1
	if extra < 1 {
		return 1
	}
	return extra
}

func (st *openAIIPGroupRetryState) advanceAfterTransient(liveIDs []int64, proxyID int64) {
	if st == nil {
		return
	}
	st.markTried(proxyID)
	if n := len(liveIDs); n > st.size {
		st.size = n
	}
	if st.done {
		return
	}
	if len(liveIDs) == 0 {
		if st.pass < openAIIPGroupPassLimit {
			st.pass++
			st.tried = map[int64]struct{}{}
			return
		}
		st.done = true
		return
	}
	for _, id := range liveIDs {
		if !st.isTried(id) {
			return
		}
	}
	if st.pass < openAIIPGroupPassLimit {
		st.pass++
		st.tried = map[int64]struct{}{}
		return
	}
	st.done = true
}

func withOpenAIIPGroupSession(ctx context.Context, sessionHash string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(sessionHash) != "" {
		ctx = context.WithValue(ctx, openaiIPGroupSessionCtxKey{}, sessionHash)
	}
	if openAIIPGroupRetryStateFrom(ctx) == nil {
		ctx = context.WithValue(ctx, openaiIPGroupRetryStateCtxKey{}, newOpenAIIPGroupRetryState())
	}
	return ctx
}

func openAIIPGroupRetryStateFrom(ctx context.Context) *openAIIPGroupRetryState {
	if ctx == nil {
		return nil
	}
	st, _ := ctx.Value(openaiIPGroupRetryStateCtxKey{}).(*openAIIPGroupRetryState)
	return st
}

func openAIIPGroupSessionFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	hash, _ := ctx.Value(openaiIPGroupSessionCtxKey{}).(string)
	return strings.TrimSpace(hash)
}

func openAIErrorContext(ctx context.Context, c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// OpenAIIPGroupBindStore persists conversation -> proxy bindings for an account.
type OpenAIIPGroupBindStore interface {
	GetBoundProxyID(ctx context.Context, accountID int64, sessionHash string) (int64, bool, error)
	SetBoundProxyID(ctx context.Context, accountID int64, sessionHash string, proxyID int64, ttl time.Duration) error
	DeleteBoundProxyID(ctx context.Context, accountID int64, sessionHash string) error
	MarkProxyCooldown(ctx context.Context, accountID, proxyID int64, ttl time.Duration) error
	IsProxyCoolingDown(ctx context.Context, accountID, proxyID int64) (bool, error)
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
	if st := openAIIPGroupRetryStateFrom(ctx); st != nil && len(members) > st.size {
		st.size = len(members)
	}
	occupySlot := sessionHash != ""
	skipTried := occupySlot
	if sessionHash != "" && r.bind != nil {
		if boundID, ok, err := r.bind.GetBoundProxyID(ctx, account.ID, sessionHash); err != nil {
			return nil, err
		} else if ok {
			tried := openAIIPGroupRetryStateFrom(ctx)
			if bound, live := members[boundID]; live && (tried == nil || !tried.isTried(boundID)) {
				release, _, err := r.tryOccupy(ctx, account.ID, bound.ID, group.PerIPConcurrency, occupySlot)
				if err != nil {
					return nil, err
				}
				// Slot full still keeps the bound IP (no rebind).
				return &resolvedOpenAIProxy{Proxy: bound, Release: release}, nil
			}
			// Bound proxy is dead or already 429'd this pass: pick another live member and rebind.
			if next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, occupySlot, skipTried); err != nil {
				return nil, err
			} else if next != nil {
				if err := r.bind.SetBoundProxyID(ctx, account.ID, sessionHash, next.ID, r.ttl); err != nil {
					release()
					return nil, err
				}
				return &resolvedOpenAIProxy{Proxy: next, Release: release}, nil
			}
			if occupySlot {
				return nil, errOpenAIIPGroupNoProxy
			}
			return &resolvedOpenAIProxy{Release: func() {}}, nil
		}
	}

	if sessionHash == "" {
		// Quota / OAuth: any live member, no conversation bind.
		if next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, false, false); err != nil {
			return nil, err
		} else if next != nil {
			return &resolvedOpenAIProxy{Proxy: next, Release: release}, nil
		}
		return &resolvedOpenAIProxy{Release: func() {}}, nil
	}

	next, release, err := r.pickAndOccupy(ctx, account.ID, group, members, occupySlot, skipTried)
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, errOpenAIIPGroupNoProxy
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

func (r *openAIIPGroupResolver) pickAndOccupy(ctx context.Context, accountID int64, group *ProxyIPGroup, members map[int64]*Proxy, occupySlot bool, skipTried bool) (*Proxy, func(), error) {
	ids := make([]int64, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	tried := openAIIPGroupRetryStateFrom(ctx)
	for _, id := range ids {
		if skipTried && tried != nil && tried.isTried(id) {
			continue
		}
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

func finishOpenAIProxyLookup(account *Account, url string, release func(), err error) (string, func(), error) {
	if release == nil {
		release = func() {}
	}
	if err != nil && !errors.Is(err, errOpenAIIPGroupNoProxy) {
		release()
		return "", func() {}, err
	}
	if err != nil || (accountUsesOpenAIIPGroup(account) && strings.TrimSpace(url) == "") {
		release()
		return "", func() {}, newOpenAIIPGroupNoProxyFailoverError()
	}
	return url, release, nil
}

func newOpenAIIPGroupNoProxyFailoverError() *UpstreamFailoverError {
	return &UpstreamFailoverError{
		StatusCode:             http.StatusServiceUnavailable,
		RequestScopedTransient: true,
		RetryableOnSameAccount: false,
		NextAccountAction:      NextAccountRetry,
		Reason:                 OpenAIIPGroupRotateReason,
		ClientStatusCode:       http.StatusServiceUnavailable,
		ClientMessage:          "Upstream request failed",
	}
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

func (r *openAIIPGroupResolver) liveMemberIDs(ctx context.Context, account *Account) []int64 {
	if r == nil || account == nil || account.ProxyIPGroupID == nil {
		return nil
	}
	group, err := r.groups.GetByID(ctx, *account.ProxyIPGroupID)
	if err != nil || group == nil {
		return nil
	}
	members, err := r.loadLiveMembers(ctx, group)
	if err != nil {
		return nil
	}
	ids := make([]int64, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	return ids
}

func (s *OpenAIGatewayService) rotateOpenAIIPGroupAfterTransient(ctx context.Context, account *Account) {
	if s == nil || !accountUsesOpenAIIPGroup(account) {
		return
	}
	st := openAIIPGroupRetryStateFrom(ctx)
	sessionHash := openAIIPGroupSessionFromCtx(ctx)
	proxyID := int64(0)
	var liveIDs []int64
	if r := s.ipGroupResolver; r != nil {
		liveIDs = r.liveMemberIDs(ctx, account)
		if r.bind != nil && sessionHash != "" {
			if id, ok, err := r.bind.GetBoundProxyID(ctx, account.ID, sessionHash); err == nil && ok {
				proxyID = id
			}
			_ = r.bind.DeleteBoundProxyID(ctx, account.ID, sessionHash)
		}
	}
	st.advanceAfterTransient(liveIDs, proxyID)
}

func (s *OpenAIGatewayService) mustOpenAIAccountProxyURL(ctx context.Context, account *Account, sessionHash string) (string, func(), error) {
	url, release, err := s.resolveOpenAIAccountProxyURL(ctx, account, sessionHash)
	return finishOpenAIProxyLookup(account, url, release, err)
}

func (s *OpenAIGatewayService) lookupOpenAIProxyURL(ctx context.Context, c *gin.Context, account *Account, body []byte) (string, func(), error) {
	sessionHash := ""
	if s != nil && c != nil {
		sessionHash = s.GenerateSessionHash(c, body)
		c.Set(ginKeyOpenAIIPGroupSession, sessionHash)
		if c.Request != nil {
			reqCtx := withOpenAIIPGroupSession(c.Request.Context(), sessionHash)
			c.Request = c.Request.WithContext(reqCtx)
			ctx = reqCtx
		}
	}
	ctx = withOpenAIIPGroupSession(ctx, sessionHash)
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
