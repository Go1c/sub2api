//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestIsOpenAIAccountQuotaExhausted(t *testing.T) {
	require.True(t, isOpenAIAccountQuotaExhausted(nil, []byte(`{"error":{"type":"usage_limit_reached"}}`)))
	require.False(t, isOpenAIAccountQuotaExhausted(nil, []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)))

	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "12")
	headers.Set("x-codex-primary-reset-after-seconds", "600")
	headers.Set("x-codex-primary-window-minutes", "300")
	headers.Set("x-codex-secondary-used-percent", "34")
	headers.Set("x-codex-secondary-reset-after-seconds", "86400")
	headers.Set("x-codex-secondary-window-minutes", "10080")
	require.False(t, isOpenAIAccountQuotaExhausted(headers, []byte(`{"error":{"type":"rate_limit_error"}}`)))

	headers.Set("x-codex-primary-used-percent", "100")
	require.True(t, isOpenAIAccountQuotaExhausted(headers, []byte(`{"error":{"type":"rate_limit_error"}}`)))
}

func TestOpenAIIPGroupRetryStateTwoPasses(t *testing.T) {
	st := newOpenAIIPGroupRetryState()
	live := []int64{2, 3}

	st.advanceAfterTransient(live, 2)
	require.False(t, st.done)
	require.Equal(t, 1, st.pass)
	require.True(t, st.isTried(2))
	require.False(t, st.isTried(3))

	st.advanceAfterTransient(live, 3)
	require.False(t, st.done)
	require.Equal(t, 2, st.pass)
	require.False(t, st.isTried(2))
	require.False(t, st.isTried(3))

	st.advanceAfterTransient(live, 2)
	st.advanceAfterTransient(live, 3)
	require.True(t, st.done)
	require.Equal(t, 3, st.extraRetries())
}

func setupIPGroupRotateTest(t *testing.T, accountID int64) (*OpenAIGatewayService, *Account, *openAIIPGroupResolver, *rateLimit429AccountRepoStub, context.Context) {
	t.Helper()
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
	account := &Account{ID: accountID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	repo := &rateLimit429AccountRepoStub{}
	svc := &OpenAIGatewayService{
		rateLimitService: NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
		ipGroupResolver:  resolver,
	}
	ctx := withOpenAIIPGroupSession(context.Background(), "conv-1")
	return svc, account, resolver, repo, ctx
}

func TestIPGroupSoft429DoesNotMarkAccountAndRebinds(t *testing.T) {
	svc, account, resolver, repo, ctx := setupIPGroupRotateTest(t, 42)

	first, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)

	body := []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)
	shouldDisable := svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusTooManyRequests, http.Header{}, body)
	require.False(t, shouldDisable)
	require.Zero(t, repo.rateLimitCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	again, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), again.Proxy.ID)
}

func TestIPGroupQuota429StillMarksAccount(t *testing.T) {
	gid := int64(1)
	account := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ProxyIPGroupID: &gid}
	repo := &rateLimit429AccountRepoStub{}
	svc := &OpenAIGatewayService{
		rateLimitService: NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
	}
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "18000")
	headers.Set("x-codex-primary-window-minutes", "300")
	body := []byte(`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached"}}`)

	shouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, headers, body)
	require.False(t, shouldDisable)
	require.Equal(t, 1, repo.rateLimitCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestSingleIPAccountSoft429StillMarksAccount(t *testing.T) {
	account := &Account{ID: 46, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	repo := &rateLimit429AccountRepoStub{}
	svc := &OpenAIGatewayService{
		rateLimitService: NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
	}
	body := []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)
	shouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, body)
	require.False(t, shouldDisable)
	require.Equal(t, 1, repo.rateLimitCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestIPGroupSoft429FailoverRetriesSameAccount(t *testing.T) {
	svc, account, resolver, repo, ctx := setupIPGroupRotateTest(t, 44)
	_, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)

	body := []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)
	require.False(t, svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusTooManyRequests, http.Header{}, body))
	failoverErr := newOpenAIUpstreamFailoverError(
		ctx,
		http.StatusTooManyRequests,
		http.Header{},
		body,
		"slow down",
		false,
		account,
	)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.True(t, failoverErr.RequestScopedTransient)
	require.Equal(t, NextAccountRetry, failoverErr.NextAccountAction)
	require.Equal(t, 3, failoverErr.SameAccountRetryMax)
	require.Zero(t, repo.rateLimitCalls)
}

func TestIPGroupOverloadRotatesWithoutMarkingAccount(t *testing.T) {
	svc, account, resolver, repo, ctx := setupIPGroupRotateTest(t, 45)

	first, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), first.Proxy.ID)

	body := []byte(`{"error":{"type":"server_error","message":"Our servers are currently overloaded. Please try again later."}}`)
	shouldDisable := svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusBadRequest, http.Header{}, body)
	require.False(t, shouldDisable)
	require.Zero(t, repo.rateLimitCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	again, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), again.Proxy.ID)
}

func TestIPGroupTwoPass429ThenSwitchAccountWithoutRateLimit(t *testing.T) {
	svc, account, resolver, repo, ctx := setupIPGroupRotateTest(t, 47)
	body := []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)
	wantOrder := []int64{2, 3, 2, 3}

	for i, wantID := range wantOrder {
		resolved, err := resolveOpenAIAccountProxy(ctx, resolver, account, "conv-1")
		require.NoError(t, err, "attempt %d", i+1)
		require.Equal(t, wantID, resolved.Proxy.ID, "attempt %d", i+1)
		if resolved.Release != nil {
			resolved.Release()
		}

		shouldDisable := svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusTooManyRequests, http.Header{}, body)
		require.False(t, shouldDisable)
		require.Zero(t, repo.rateLimitCalls)
		require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

		failoverErr := newOpenAIUpstreamFailoverError(
			ctx,
			http.StatusTooManyRequests,
			http.Header{},
			body,
			"slow down",
			false,
			account,
		)
		require.True(t, failoverErr.RequestScopedTransient)
		require.Equal(t, NextAccountRetry, failoverErr.NextAccountAction)
		if i < len(wantOrder)-1 {
			require.True(t, failoverErr.RetryableOnSameAccount, "attempt %d should stay on the account", i+1)
			require.Equal(t, 3, failoverErr.EffectiveSameAccountRetryLimit(3))
			continue
		}
		require.False(t, failoverErr.RetryableOnSameAccount, "after two full IP passes, switch account")
		require.Equal(t, 3, failoverErr.EffectiveSameAccountRetryLimit(3), "done path keeps the caller pool default")
	}
}

func TestEffectiveSameAccountRetryLimitOverridesPoolDefault(t *testing.T) {
	err := &UpstreamFailoverError{SameAccountRetryMax: 5}
	require.Equal(t, 5, err.EffectiveSameAccountRetryLimit(3))
	require.Equal(t, 3, (&UpstreamFailoverError{}).EffectiveSameAccountRetryLimit(3))
}
