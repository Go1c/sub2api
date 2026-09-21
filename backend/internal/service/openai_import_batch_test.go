package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func batchTestService(t *testing.T, advanced string, accounts []Account, cache schedulerTestConcurrencyCache) *OpenAIGatewayService {
	t.Helper()
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
	cfg := newSchedulerTestSubscriptionPriorityConfig()
	cfg.Gateway.Scheduling = config.GatewaySchedulingConfig{StickySessionMaxWaiting: 3, StickySessionWaitTimeout: time.Second, FallbackWaitTimeout: time.Second, FallbackMaxWaiting: 100, LoadBatchEnabled: true}
	repo := &openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{"openai_import_batch_minutes": "30", openAIAdvancedSchedulerSettingKey: advanced}}
	return &OpenAIGatewayService{cfg: cfg, accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts}, cache: &schedulerTestGatewayCache{}, concurrencyService: NewConcurrencyService(cache), rateLimitService: &RateLimitService{settingService: NewSettingService(repo, cfg)}}
}

func batchTestAccounts() []Account {
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	return []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 8, CreatedAt: start.Add(time.Minute)},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 8, CreatedAt: start.Add(31 * time.Minute)},
	}
}

func batchSelect(t *testing.T, svc *OpenAIGatewayService, session string) *AccountSelectionResult {
	t.Helper()
	result, _, err := svc.SelectAccountWithScheduler(context.Background(), nil, "", session, "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	if result.ReleaseFunc != nil {
		t.Cleanup(result.ReleaseFunc)
	}
	return result
}

func TestImportBatchOlderBeforeLowerLoad(t *testing.T) {
	for _, advanced := range []string{"false", "true"} {
		t.Run(advanced, func(t *testing.T) {
			svc := batchTestService(t, advanced, batchTestAccounts(), schedulerTestConcurrencyCache{loadMap: map[int64]*AccountLoadInfo{1: {AccountID: 1, LoadRate: 75, CurrentConcurrency: 6}, 2: {AccountID: 2}}})
			require.Equal(t, int64(1), batchSelect(t, svc, "").Account.ID)
		})
	}
}

func TestImportBatchSameWindowUsesLoadAndBoundaryStartsNewBatch(t *testing.T) {
	for _, delta := range []time.Duration{28 * time.Minute, 29 * time.Minute} {
		t.Run(delta.String(), func(t *testing.T) {
			accounts := batchTestAccounts()
			accounts[1].CreatedAt = accounts[0].CreatedAt.Add(delta)
			svc := batchTestService(t, "true", accounts, schedulerTestConcurrencyCache{loadMap: map[int64]*AccountLoadInfo{1: {AccountID: 1, LoadRate: 75, CurrentConcurrency: 6}, 2: {AccountID: 2}}})
			want := int64(2)
			if delta == 29*time.Minute {
				want = 1
			}
			require.Equal(t, want, batchSelect(t, svc, "").Account.ID)
		})
	}
}

func TestImportBatchOverflowAndPriority(t *testing.T) {
	t.Run("full old batch spills", func(t *testing.T) {
		svc := batchTestService(t, "true", batchTestAccounts(), schedulerTestConcurrencyCache{acquireResults: map[int64]bool{1: false, 2: true}})
		require.Equal(t, int64(2), batchSelect(t, svc, "").Account.ID)
	})
	t.Run("manual priority before age", func(t *testing.T) {
		accounts := batchTestAccounts()
		accounts[0].Priority = 10
		svc := batchTestService(t, "true", accounts, schedulerTestConcurrencyCache{})
		require.Equal(t, int64(2), batchSelect(t, svc, "").Account.ID)
	})
}

func TestImportBatchStickyIgnoresLatencyAndKeepsBindingOnOverflow(t *testing.T) {
	svc := batchTestService(t, "true", batchTestAccounts(), schedulerTestConcurrencyCache{acquireResults: map[int64]bool{1: true, 2: false}, waitCounts: map[int64]int{2: 3}})
	cache := svc.cache.(*schedulerTestGatewayCache)
	cache.sessionBindings = map[string]int64{"openai:session": 2}
	require.Equal(t, int64(1), batchSelect(t, svc, "session").Account.ID)
	require.Equal(t, int64(2), cache.sessionBindings["openai:session"])
	svc.concurrencyService = NewConcurrencyService(schedulerTestConcurrencyCache{})
	slow := 60000
	svc.ReportOpenAIAccountScheduleResult(&batchTestAccounts()[1], "gpt-5", true, &slow)
	require.Equal(t, int64(2), batchSelect(t, svc, "session").Account.ID)
}

func TestImportBatchRepeatedFailuresDemoteOldAccount(t *testing.T) {
	svc := batchTestService(t, "false", batchTestAccounts(), schedulerTestConcurrencyCache{})
	account := batchTestAccounts()[0]
	for i := 0; i < 5; i++ {
		svc.ReportOpenAIAccountScheduleResult(&account, "gpt-5", false, nil, &UpstreamFailoverError{StatusCode: 503})
	}
	require.Equal(t, int64(2), batchSelect(t, svc, "").Account.ID)
}

func TestImportBatchClientErrorsDoNotDemote(t *testing.T) {
	svc := batchTestService(t, "true", batchTestAccounts(), schedulerTestConcurrencyCache{})
	account := batchTestAccounts()[0]
	for i := 0; i < 20; i++ {
		svc.ReportOpenAIAccountScheduleResult(&account, "gpt-5", false, nil, &UpstreamFailoverError{StatusCode: 400})
	}
	require.Equal(t, int64(1), batchSelect(t, svc, "").Account.ID)
}
