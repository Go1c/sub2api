//go:build integration

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageBillingRepositoryApply_DeduplicatesBalanceBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-" + uuid.NewString(),
		Name:   "billing",
		Quota:  1,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:           requestID,
		APIKeyID:            apiKey.ID,
		UserID:              user.ID,
		AccountID:           account.ID,
		AccountType:         service.AccountTypeAPIKey,
		BalanceCost:         1.25,
		APIKeyQuotaCost:     1.25,
		APIKeyRateLimitCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, result1.Applied)
	require.True(t, result1.APIKeyQuotaExhausted)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used FROM api_keys WHERE id = $1", apiKey.ID).Scan(&quotaUsed))
	require.InDelta(t, 1.25, quotaUsed, 0.000001)

	var usage5h float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT usage_5h FROM api_keys WHERE id = $1", apiKey.ID).Scan(&usage5h))
	require.InDelta(t, 1.25, usage5h, 0.000001)

	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status FROM api_keys WHERE id = $1", apiKey.ID).Scan(&status))
	require.Equal(t, service.StatusAPIKeyQuotaExhausted, status)

	var dedupCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&dedupCount))
	require.Equal(t, 1, dedupCount)
}

func TestUsageBillingRepositoryApply_DeduplicatesSubscriptionBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-sub-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "usage-billing-group-" + uuid.NewString(),
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeSubscription,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		GroupID: &group.ID,
		Key:     "sk-usage-billing-sub-" + uuid.NewString(),
		Name:    "billing-sub",
	})
	subscriptionID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD: 100,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:        requestID,
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        0,
		SubscriptionID:   &subscriptionID,
		SubscriptionCost: 2.5,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var dailyUsage float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT daily_usage_usd FROM user_subscriptions WHERE id = $1", subscriptionID).Scan(&dailyUsage))
	require.InDelta(t, 2.5, dailyUsage, 0.000001)
}

func TestUsageBillingRepositoryApply_RequestFingerprintConflict(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-conflict-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-conflict-" + uuid.NewString(),
		Name:   "billing-conflict",
	})

	requestID := uuid.NewString()
	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	})
	require.NoError(t, err)

	_, err = repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 2.50,
	})
	require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
}

func TestUsageBillingRepositoryApply_UpdatesAccountQuota(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-account-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-account-" + uuid.NewString(),
		Name:   "billing-account",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-quota-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_limit": 100.0,
		},
	})

	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        account.ID,
		AccountType:      service.AccountTypeAPIKey,
		AccountQuotaCost: 3.5,
	})
	require.NoError(t, err)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COALESCE((extra->>'quota_used')::numeric, 0) FROM accounts WHERE id = $1", account.ID).Scan(&quotaUsed))
	require.InDelta(t, 3.5, quotaUsed, 0.000001)
}

func TestUsageBillingRepositoryApply_EnqueuesSchedulerOutboxOnQuotaCrossing(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	newFixture := func(t *testing.T, extra map[string]any) (int64, int64) {
		t.Helper()
		user := mustCreateUser(t, client, &service.User{
			Email:        fmt.Sprintf("usage-billing-outbox-user-%d-%s@example.com", time.Now().UnixNano(), uuid.NewString()),
			PasswordHash: "hash",
		})
		apiKey := mustCreateApiKey(t, client, &service.APIKey{
			UserID: user.ID,
			Key:    "sk-usage-billing-outbox-" + uuid.NewString(),
			Name:   "billing-outbox",
		})
		account := mustCreateAccount(t, client, &service.Account{
			Name:  "usage-billing-outbox-" + uuid.NewString(),
			Type:  service.AccountTypeAPIKey,
			Extra: extra,
		})
		return apiKey.ID, account.ID
	}

	outboxCountFor := func(t *testing.T, accountID int64) int {
		t.Helper()
		var count int
		require.NoError(t, integrationDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM scheduler_outbox WHERE event_type = $1 AND account_id = $2",
			service.SchedulerOutboxEventAccountChanged, accountID,
		).Scan(&count))
		return count
	}

	t.Run("daily_first_crossing_enqueues", func(t *testing.T) {
		apiKeyID, accountID := newFixture(t, map[string]any{
			"quota_daily_limit": 10.0,
		})
		// 第一次低于日限额：不应入队 outbox
		_, err := repo.Apply(ctx, &service.UsageBillingCommand{
			RequestID:        uuid.NewString(),
			APIKeyID:         apiKeyID,
			AccountID:        accountID,
			AccountType:      service.AccountTypeAPIKey,
			AccountQuotaCost: 4,
		})
		require.NoError(t, err)
		require.Equal(t, 0, outboxCountFor(t, accountID), "below limit should not enqueue")

		// 第二次跨越日限额：应入队一次 outbox
		_, err = repo.Apply(ctx, &service.UsageBillingCommand{
			RequestID:        uuid.NewString(),
			APIKeyID:         apiKeyID,
			AccountID:        accountID,
			AccountType:      service.AccountTypeAPIKey,
			AccountQuotaCost: 8,
		})
		require.NoError(t, err)
		require.Equal(t, 1, outboxCountFor(t, accountID), "crossing daily limit should enqueue once")

		// 再次递增（已超）：不应重复入队
		_, err = repo.Apply(ctx, &service.UsageBillingCommand{
			RequestID:        uuid.NewString(),
			APIKeyID:         apiKeyID,
			AccountID:        accountID,
			AccountType:      service.AccountTypeAPIKey,
			AccountQuotaCost: 2,
		})
		require.NoError(t, err)
		require.Equal(t, 1, outboxCountFor(t, accountID), "subsequent increments beyond limit should not re-enqueue")
	})

	t.Run("weekly_first_crossing_enqueues", func(t *testing.T) {
		apiKeyID, accountID := newFixture(t, map[string]any{
			"quota_weekly_limit": 10.0,
		})
		_, err := repo.Apply(ctx, &service.UsageBillingCommand{
			RequestID:        uuid.NewString(),
			APIKeyID:         apiKeyID,
			AccountID:        accountID,
			AccountType:      service.AccountTypeAPIKey,
			AccountQuotaCost: 15, // 单次即跨越
		})
		require.NoError(t, err)
		require.Equal(t, 1, outboxCountFor(t, accountID), "single-shot crossing weekly limit should enqueue once")
	})
}

func TestDashboardAggregationRepositoryCleanupUsageBillingDedup_BatchDeletesOldRows(t *testing.T) {
	ctx := context.Background()
	repo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	oldRequestID := "dedup-old-" + uuid.NewString()
	newRequestID := "dedup-new-" + uuid.NewString()
	oldCreatedAt := time.Now().UTC().AddDate(0, 0, -400)
	newCreatedAt := time.Now().UTC().Add(-time.Hour)

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint, created_at)
		VALUES ($1, 1, $2, $3), ($4, 1, $5, $6)
	`,
		oldRequestID, strings.Repeat("a", 64), oldCreatedAt,
		newRequestID, strings.Repeat("b", 64), newCreatedAt,
	)
	require.NoError(t, err)

	require.NoError(t, repo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	var oldCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", oldRequestID).Scan(&oldCount))
	require.Equal(t, 0, oldCount)

	var newCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", newRequestID).Scan(&newCount))
	require.Equal(t, 1, newCount)

	var archivedCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup_archive WHERE request_id = $1", oldRequestID).Scan(&archivedCount))
	require.Equal(t, 1, archivedCount)
}

func TestUsageBillingRepositoryApply_DeduplicatesAgainstArchivedKey(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	aggRepo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-archive-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-archive-" + uuid.NewString(),
		Name:   "billing-archive",
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	_, err = integrationDB.ExecContext(ctx, `
		UPDATE usage_billing_dedup
		SET created_at = $1
		WHERE request_id = $2 AND api_key_id = $3
	`, time.Now().UTC().AddDate(0, 0, -400), requestID, apiKey.ID)
	require.NoError(t, err)
	require.NoError(t, aggRepo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_MixesSubscriptionAndBalanceWhenTotalQuotaIsLow(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-mixed-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-mixed-" + uuid.NewString(),
		Name:   "billing-mixed",
	})
	subID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  8,
	})

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		SubscriptionID:   &subID,
		SubscriptionCost: 5,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 2.0, result.SubscriptionCost, 0.000001)
	require.InDelta(t, 3.0, result.BalanceCost, 0.000001)
	require.ElementsMatch(t, []string{service.SubscriptionLimitReachedTotal}, result.LimitReachedKinds)

	var quotaUsed float64
	var exhaustedAt sql.NullTime
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT quota_used_usd, exhausted_at
		FROM user_subscriptions
		WHERE id = $1
	`, subID).Scan(&quotaUsed, &exhaustedAt))
	require.InDelta(t, 10.0, quotaUsed, 0.000001)
	require.True(t, exhaustedAt.Valid, "total quota crossing should set exhausted_at")

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 97.0, balance, 0.000001)

	var consumeDelta, consumeBalanceDelta, remainingAfter float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT delta_usd, balance_delta_usd, remaining_after_usd
		FROM subscription_credit_ledger
		WHERE subscription_id = $1 AND type = $2
		ORDER BY id DESC
		LIMIT 1
	`, subID, service.SubscriptionCreditLedgerConsume).Scan(&consumeDelta, &consumeBalanceDelta, &remainingAfter))
	require.InDelta(t, -2.0, consumeDelta, 0.000001)
	require.InDelta(t, -3.0, consumeBalanceDelta, 0.000001)
	require.InDelta(t, 0.0, remainingAfter, 0.000001)

	requireSubscriptionNotifyOutbox(t, ctx, user.ID, subID, "limit_reached_"+service.SubscriptionLimitReachedTotal)
}

func TestUsageBillingRepositoryApply_ConsumesMultipleSubscriptionsByCreationTime(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-multi-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-multi-" + uuid.NewString(),
		Name:   "billing-multi",
	})
	firstSubID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  8,
	})
	secondSubID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD: 10,
	})
	setUsageBillingSubscriptionCreatedAt(t, firstSubID, time.Now().UTC().Add(-2*time.Hour))
	setUsageBillingSubscriptionCreatedAt(t, secondSubID, time.Now().UTC().Add(-time.Hour))

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		SubscriptionIDs:  []int64{secondSubID, firstSubID},
		SubscriptionCost: 5,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 5.0, result.SubscriptionCost, 0.000001)
	require.InDelta(t, 0.0, result.BalanceCost, 0.000001)
	require.ElementsMatch(t, []string{service.SubscriptionLimitReachedTotal}, result.LimitReachedKinds)

	var firstUsed, secondUsed, balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used_usd FROM user_subscriptions WHERE id = $1", firstSubID).Scan(&firstUsed))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used_usd FROM user_subscriptions WHERE id = $1", secondSubID).Scan(&secondUsed))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 10.0, firstUsed, 0.000001)
	require.InDelta(t, 3.0, secondUsed, 0.000001)
	require.InDelta(t, 100.0, balance, 0.000001)

	var firstDelta, secondDelta float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT delta_usd
		FROM subscription_credit_ledger
		WHERE subscription_id = $1 AND type = $2
		ORDER BY id DESC
		LIMIT 1
	`, firstSubID, service.SubscriptionCreditLedgerConsume).Scan(&firstDelta))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT delta_usd
		FROM subscription_credit_ledger
		WHERE subscription_id = $1 AND type = $2
		ORDER BY id DESC
		LIMIT 1
	`, secondSubID, service.SubscriptionCreditLedgerConsume).Scan(&secondDelta))
	require.InDelta(t, -2.0, firstDelta, 0.000001)
	require.InDelta(t, -3.0, secondDelta, 0.000001)
}

func TestUsageBillingRepositoryApply_FallsThroughWhenEarliestSubscriptionWindowLimitReached(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-multi-limit-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-multi-limit-" + uuid.NewString(),
		Name:   "billing-multi-limit",
	})
	dailyLimit := 5.0
	today := time.Now().UTC().Truncate(24 * time.Hour)
	firstSubID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD:    50,
		DailyLimitUSD:    &dailyLimit,
		DailyUsageUSD:    5,
		DailyWindowStart: &today,
	})
	secondSubID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD: 50,
	})
	setUsageBillingSubscriptionCreatedAt(t, firstSubID, time.Now().UTC().Add(-2*time.Hour))
	setUsageBillingSubscriptionCreatedAt(t, secondSubID, time.Now().UTC().Add(-time.Hour))

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		SubscriptionIDs:  []int64{firstSubID, secondSubID},
		SubscriptionCost: 4,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 4.0, result.SubscriptionCost, 0.000001)
	require.InDelta(t, 0.0, result.BalanceCost, 0.000001)

	var firstUsed, secondUsed, balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used_usd FROM user_subscriptions WHERE id = $1", firstSubID).Scan(&firstUsed))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used_usd FROM user_subscriptions WHERE id = $1", secondSubID).Scan(&secondUsed))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0.0, firstUsed, 0.000001)
	require.InDelta(t, 4.0, secondUsed, 0.000001)
	require.InDelta(t, 100.0, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_RecordsDailyLimitReachedOnce(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-daily-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-daily-" + uuid.NewString(),
		Name:   "billing-daily",
	})
	dailyLimit := 5.0
	today := time.Now().UTC().Truncate(24 * time.Hour)
	subID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD:    100,
		QuotaUsedUSD:     10,
		DailyLimitUSD:    &dailyLimit,
		DailyUsageUSD:    4,
		DailyWindowStart: &today,
	})

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		SubscriptionID:   &subID,
		SubscriptionCost: 3,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 1.0, result.SubscriptionCost, 0.000001)
	require.InDelta(t, 2.0, result.BalanceCost, 0.000001)
	require.ElementsMatch(t, []string{service.SubscriptionLimitReachedDaily}, result.LimitReachedKinds)

	var dailyUsage float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT daily_usage_usd
		FROM user_subscriptions
		WHERE id = $1
	`, subID).Scan(&dailyUsage))
	require.InDelta(t, 5.0, dailyUsage, 0.000001)

	requireSubscriptionNotifyOutbox(t, ctx, user.ID, subID, "limit_reached_"+service.SubscriptionLimitReachedDaily)
}

func TestUsageBillingRepositoryApply_LogsDailyWindowResetWasteBeforeConsuming(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-reset-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-reset-" + uuid.NewString(),
		Name:   "billing-reset",
	})
	dailyLimit := 10.0
	oldWindowStart := time.Now().UTC().Add(-25 * time.Hour).Truncate(24 * time.Hour)
	subID := mustCreateUsageBillingCreditSubscription(t, user.ID, usageBillingCreditSubSpec{
		QuotaLimitUSD:    100,
		QuotaUsedUSD:     5,
		DailyLimitUSD:    &dailyLimit,
		DailyUsageUSD:    4,
		DailyWindowStart: &oldWindowStart,
	})

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		SubscriptionID:   &subID,
		SubscriptionCost: 3,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 3.0, result.SubscriptionCost, 0.000001)
	require.InDelta(t, 0.0, result.BalanceCost, 0.000001)

	var dailyUsage float64
	var dailyWindowStart time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT daily_usage_usd, daily_window_start
		FROM user_subscriptions
		WHERE id = $1
	`, subID).Scan(&dailyUsage, &dailyWindowStart))
	require.InDelta(t, 3.0, dailyUsage, 0.000001)
	require.True(t, dailyWindowStart.After(oldWindowStart), "daily window should advance")

	var metadataRaw []byte
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT metadata
		FROM subscription_credit_ledger
		WHERE subscription_id = $1 AND type = $2 AND event_key LIKE 'window_reset_daily:%'
		ORDER BY id DESC
		LIMIT 1
	`, subID, service.SubscriptionCreditLedgerWindowReset).Scan(&metadataRaw))
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(metadataRaw, &metadata))
	require.Equal(t, "daily", metadata["window"])
	require.InDelta(t, 6.0, metadata["wasted_usd"].(float64), 0.000001)
	require.InDelta(t, 0.6, metadata["wasted_ratio"].(float64), 0.000001)
}

type usageBillingCreditSubSpec struct {
	QuotaLimitUSD     float64
	QuotaUsedUSD      float64
	DailyLimitUSD     *float64
	WeeklyLimitUSD    *float64
	DailyUsageUSD     float64
	WeeklyUsageUSD    float64
	DailyWindowStart  *time.Time
	WeeklyWindowStart *time.Time
}

func mustCreateUsageBillingCreditSubscription(t *testing.T, userID int64, spec usageBillingCreditSubSpec) int64 {
	t.Helper()
	now := time.Now().UTC()
	var id int64
	err := integrationDB.QueryRowContext(context.Background(), `
		INSERT INTO user_subscriptions (
			user_id, scope_type, scope_config,
			quota_limit_usd, quota_used_usd,
			daily_limit_usd, weekly_limit_usd,
			starts_at, expires_at, status,
			daily_window_start, weekly_window_start,
			daily_usage_usd, weekly_usage_usd,
			assigned_at, notes
		) VALUES (
			$1, $2, '{}'::jsonb,
			$3, $4,
			$5, $6,
			$7, $8, $9,
			$10, $11,
			$12, $13,
			$14, ''
		)
		RETURNING id
	`,
		userID,
		service.SubscriptionScopeAllAvailableGroups,
		spec.QuotaLimitUSD,
		spec.QuotaUsedUSD,
		spec.DailyLimitUSD,
		spec.WeeklyLimitUSD,
		now.Add(-time.Hour),
		now.Add(24*time.Hour),
		service.SubscriptionStatusActive,
		spec.DailyWindowStart,
		spec.WeeklyWindowStart,
		spec.DailyUsageUSD,
		spec.WeeklyUsageUSD,
		now,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func setUsageBillingSubscriptionCreatedAt(t *testing.T, subID int64, createdAt time.Time) {
	t.Helper()
	_, err := integrationDB.ExecContext(context.Background(), `
		UPDATE user_subscriptions
		SET created_at = $2, updated_at = $2
		WHERE id = $1
	`, subID, createdAt.UTC())
	require.NoError(t, err)
}

func requireSubscriptionNotifyOutbox(t *testing.T, ctx context.Context, userID int64, subID int64, kind string) {
	t.Helper()
	var payloadRaw []byte
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT payload
		FROM scheduler_outbox
		WHERE event_type = $1
			AND payload->>'kind' = $2
			AND (payload->>'user_id')::bigint = $3
			AND (payload->>'subscription_id')::bigint = $4
		ORDER BY id DESC
		LIMIT 1
	`, service.SchedulerOutboxEventSubscriptionNotify, kind, userID, subID).Scan(&payloadRaw))
}
