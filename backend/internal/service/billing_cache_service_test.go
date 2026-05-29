package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates      int64
	subscriptionUpdates int64
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) (*UserPlatformQuotaCacheEntry, bool, error) {
	return nil, false, nil
}

func (b *billingCacheWorkerStub) SetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string, entry *UserPlatformQuotaCacheEntry, ttl time.Duration) error {
	return nil
}

func (b *billingCacheWorkerStub) DeleteUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) error {
	return nil
}

func (b *billingCacheWorkerStub) IncrUserPlatformQuotaUsageCache(ctx context.Context, userID int64, platform string, cost float64, ttl time.Duration, markDirty bool) error {
	return nil
}

func (b *billingCacheWorkerStub) PopDirtyUserPlatformQuotaKeys(ctx context.Context, n int) ([]UserPlatformQuotaKey, error) {
	return nil, nil
}

func (b *billingCacheWorkerStub) ReaddDirtyUserPlatformQuotaKeys(ctx context.Context, keys []UserPlatformQuotaKey) error {
	return nil
}

func (b *billingCacheWorkerStub) BatchGetUserPlatformQuotaCache(ctx context.Context, keys []UserPlatformQuotaKey) ([]*UserPlatformQuotaCacheEntry, error) {
	return nil, nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

type balanceGateCacheStub struct {
	balance float64
	subData *SubscriptionCacheData
}

type balanceGateRechargeReaderStub struct {
	amount float64
	err    error
}

func (s balanceGateRechargeReaderStub) SumBalanceUsageGateRechargeAmount(context.Context, int64) (float64, error) {
	return s.amount, s.err
}

func (b *balanceGateCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	return b.balance, nil
}

func (b *balanceGateCacheStub) SetUserBalance(context.Context, int64, float64) error {
	return nil
}

func (b *balanceGateCacheStub) DeductUserBalance(context.Context, int64, float64) error {
	return nil
}

func (b *balanceGateCacheStub) InvalidateUserBalance(context.Context, int64) error {
	return nil
}

func (b *balanceGateCacheStub) GetSubscriptionCache(context.Context, int64, int64) (*SubscriptionCacheData, error) {
	if b.subData != nil {
		return b.subData, nil
	}
	return nil, errors.New("not implemented")
}

func (b *balanceGateCacheStub) SetSubscriptionCache(context.Context, int64, int64, *SubscriptionCacheData) error {
	return nil
}

func (b *balanceGateCacheStub) UpdateSubscriptionUsage(context.Context, int64, int64, float64) error {
	return nil
}

func (b *balanceGateCacheStub) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	return nil
}

func (b *balanceGateCacheStub) GetAPIKeyRateLimit(context.Context, int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *balanceGateCacheStub) SetAPIKeyRateLimit(context.Context, int64, *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *balanceGateCacheStub) UpdateAPIKeyRateLimitUsage(context.Context, int64, float64) error {
	return nil
}

func (b *balanceGateCacheStub) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	return nil
}

func (b *balanceGateCacheStub) GetUserPlatformQuotaCache(context.Context, int64, string) (*UserPlatformQuotaCacheEntry, bool, error) {
	return nil, false, nil
}

func (b *balanceGateCacheStub) SetUserPlatformQuotaCache(context.Context, int64, string, *UserPlatformQuotaCacheEntry, time.Duration) error {
	return nil
}

func (b *balanceGateCacheStub) DeleteUserPlatformQuotaCache(context.Context, int64, string) error {
	return nil
}

func (b *balanceGateCacheStub) IncrUserPlatformQuotaUsageCache(context.Context, int64, string, float64, time.Duration, bool) error {
	return nil
}

func (b *balanceGateCacheStub) PopDirtyUserPlatformQuotaKeys(context.Context, int) ([]UserPlatformQuotaKey, error) {
	return nil, nil
}

func (b *balanceGateCacheStub) ReaddDirtyUserPlatformQuotaKeys(context.Context, []UserPlatformQuotaKey) error {
	return nil
}

func (b *balanceGateCacheStub) BatchGetUserPlatformQuotaCache(context.Context, []UserPlatformQuotaKey) ([]*UserPlatformQuotaCacheEntry, error) {
	return nil, nil
}

func TestBillingCacheServiceCheckBillingEligibility_BalanceUsageGateRequiresRecharge(t *testing.T) {
	resetBalanceUsageGateCacheForTest()
	settingSvc := NewSettingService(&balanceGateSettingRepoStub{values: map[string]string{
		"balance_usage_gate_enabled":      "true",
		"balance_usage_gate_min_balance":  "1",
		"balance_usage_gate_min_recharge": "5",
	}}, nil)
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 3}, nil, nil, nil, nil, nil, &config.Config{}, settingSvc)
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1, TotalRecharged: 0}, nil, &Group{}, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "为防止批量注册")
	require.Contains(t, err.Error(), "最低充值 5.00 元")
	require.Contains(t, err.Error(), "正常使用赠送余额")
	require.Contains(t, err.Error(), "无理由退款")
	require.Contains(t, err.Error(), "5.00")
	require.NotContains(t, err.Error(), "当前历史充值")
	require.NotContains(t, err.Error(), "0.00")
}

func TestBillingCacheServiceCheckBillingEligibility_BalanceUsageGateRequiresMinBalance(t *testing.T) {
	resetBalanceUsageGateCacheForTest()
	settingSvc := NewSettingService(&balanceGateSettingRepoStub{values: map[string]string{
		"balance_usage_gate_enabled":      "true",
		"balance_usage_gate_min_balance":  "2",
		"balance_usage_gate_min_recharge": "5",
	}}, nil)
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 1.25}, nil, nil, nil, nil, nil, &config.Config{}, settingSvc)
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1, TotalRecharged: 6}, nil, &Group{}, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "余额需大于 2.00")
	require.NotContains(t, err.Error(), "当前余额")
	require.NotContains(t, err.Error(), "1.25")
}

func TestBillingCacheServiceCheckBillingEligibility_BalanceUsageGateAllowsQualifiedUser(t *testing.T) {
	resetBalanceUsageGateCacheForTest()
	settingSvc := NewSettingService(&balanceGateSettingRepoStub{values: map[string]string{
		"balance_usage_gate_enabled":      "true",
		"balance_usage_gate_min_balance":  "2",
		"balance_usage_gate_min_recharge": "5",
	}}, nil)
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 3}, nil, nil, nil, nil, nil, &config.Config{}, settingSvc)
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1, TotalRecharged: 6}, nil, &Group{}, nil)

	require.NoError(t, err)
}

func TestBillingCacheServiceCheckBillingEligibility_BalanceUsageGateCountsExternalSubscriptionPayments(t *testing.T) {
	resetBalanceUsageGateCacheForTest()
	settingSvc := NewSettingService(&balanceGateSettingRepoStub{values: map[string]string{
		"balance_usage_gate_enabled":      "true",
		"balance_usage_gate_min_balance":  "2",
		"balance_usage_gate_min_recharge": "10",
	}}, nil)
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 12}, nil, nil, nil, nil, nil, &config.Config{}, settingSvc)
	svc.SetBalanceUsageGateRechargeReader(balanceGateRechargeReaderStub{amount: 79})
	t.Cleanup(svc.Stop)

	limit := 1.0
	sub := &UserSubscription{
		ID:            10,
		UserID:        1,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		ScopeType:     SubscriptionScopeAllAvailableGroups,
		ScopeConfig:   map[string]any{},
		DailyLimitUSD: &limit,
		DailyUsageUSD: 1,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  1,
	}

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1, TotalRecharged: 0},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.NoError(t, err)
}

func TestBillingCacheServiceCheckBillingEligibility_SubscriptionDoesNotRequireBalance(t *testing.T) {
	limit := 1.0
	sub := &UserSubscription{
		ID:            10,
		UserID:        1,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		ScopeType:     SubscriptionScopeAllAvailableGroups,
		ScopeConfig:   map[string]any{},
		DailyLimitUSD: &limit,
		DailyUsageUSD: 0.25,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  1,
	}
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 0}, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.NoError(t, err)
}

func TestBillingCacheServiceCheckBillingEligibility_SubscriptionLimitFallsBackToBalance(t *testing.T) {
	limit := 1.0
	sub := &UserSubscription{
		ID:            10,
		UserID:        1,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		ScopeType:     SubscriptionScopeAllAvailableGroups,
		ScopeConfig:   map[string]any{},
		DailyLimitUSD: &limit,
		DailyUsageUSD: 1,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  1,
	}
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 3}, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.NoError(t, err)
}

func TestBillingCacheServiceCheckBillingEligibility_SubscriptionLimitWithoutBalanceReturnsLimitError(t *testing.T) {
	limit := 1.0
	sub := &UserSubscription{
		ID:            10,
		UserID:        1,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		ScopeType:     SubscriptionScopeAllAvailableGroups,
		ScopeConfig:   map[string]any{},
		DailyLimitUSD: &limit,
		DailyUsageUSD: 1,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  1,
	}
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 0}, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}

// 回归：订阅总额度池耗尽（已用 >= 上限）但日/周窗口仍未触顶时，
// 订阅必须被判为不可消费并回落到余额闸门——余额不足则直接拒绝，
// 不能因为窗口未触顶而误判订阅可用、绕过余额检查导致余额被扣成负数。
func TestBillingCacheServiceCheckBillingEligibility_ExhaustedPoolWithoutBalanceBlocks(t *testing.T) {
	dailyLimit := 35.0
	weeklyLimit := 35.0
	sub := &UserSubscription{
		ID:             10,
		UserID:         1,
		Status:         SubscriptionStatusActive,
		ExpiresAt:      time.Now().Add(time.Hour),
		ScopeType:      SubscriptionScopeAllAvailableGroups,
		ScopeConfig:    map[string]any{},
		DailyLimitUSD:  &dailyLimit,
		DailyUsageUSD:  33, // 窗口冻结在上限以下
		WeeklyLimitUSD: &weeklyLimit,
		WeeklyUsageUSD: 33,
		QuotaLimitUSD:  93,
		QuotaUsedUSD:   93, // 总池耗尽
	}
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 0}, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.Error(t, err)
}

// 回归：总池耗尽时若用户仍有正余额，则回落到余额计费（允许）。
func TestBillingCacheServiceCheckBillingEligibility_ExhaustedPoolFallsBackToBalance(t *testing.T) {
	dailyLimit := 35.0
	sub := &UserSubscription{
		ID:            10,
		UserID:        1,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		ScopeType:     SubscriptionScopeAllAvailableGroups,
		ScopeConfig:   map[string]any{},
		DailyLimitUSD: &dailyLimit,
		DailyUsageUSD: 33,
		QuotaLimitUSD: 93,
		QuotaUsedUSD:  93,
	}
	svc := NewBillingCacheService(&balanceGateCacheStub{balance: 5}, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		nil,
		&Group{ID: 2, Status: StatusActive, Platform: PlatformAnthropic},
		sub,
	)

	require.NoError(t, err)
}

func TestSettingServiceGetBalanceUsageGateSettingsUsesProcessCache(t *testing.T) {
	resetBalanceUsageGateCacheForTest()
	repo := &balanceGateSettingRepoStub{values: map[string]string{
		"balance_usage_gate_enabled":      "true",
		"balance_usage_gate_min_balance":  "2",
		"balance_usage_gate_min_recharge": "5",
	}}
	settingSvc := NewSettingService(repo, nil)

	enabled, minBalance, minRecharge := settingSvc.GetBalanceUsageGateSettings(context.Background())
	require.True(t, enabled)
	require.Equal(t, 2.0, minBalance)
	require.Equal(t, 5.0, minRecharge)

	enabled, minBalance, minRecharge = settingSvc.GetBalanceUsageGateSettings(context.Background())
	require.True(t, enabled)
	require.Equal(t, 2.0, minBalance)
	require.Equal(t, 5.0, minRecharge)
	require.Equal(t, 1, repo.getMultipleCalls)
}

func resetBalanceUsageGateCacheForTest() {
	balanceUsageGateSF.Forget("balance_usage_gate")
	balanceUsageGateCache.Store(&cachedBalanceUsageGateSettings{expiresAt: 0})
}

type balanceGateSettingRepoStub struct {
	values           map[string]string
	getMultipleCalls int
}

func (s *balanceGateSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *balanceGateSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", errors.New("setting not found")
	}
	return value, nil
}

func (s *balanceGateSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *balanceGateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	s.getMultipleCalls++
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *balanceGateSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *balanceGateSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *balanceGateSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}
