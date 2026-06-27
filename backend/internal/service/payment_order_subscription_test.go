//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionOrderSnapshotFromPlanCopiesCreditFields(t *testing.T) {
	daily := 3.5
	weekly := 12.5
	plan := &dbent.SubscriptionPlan{
		ID:             42,
		QuotaUsd:       25,
		DailyLimitUsd:  &daily,
		WeeklyLimitUsd: &weekly,
		ScopeType:      SubscriptionScopePlatforms,
		ScopeConfig:    map[string]any{"platforms": []any{PlatformAnthropic}},
		ValidityDays:   2,
		ValidityUnit:   validityUnitWeek,
	}

	snapshot, err := subscriptionOrderSnapshotFromPlan(plan)

	require.NoError(t, err)
	require.Equal(t, int64(42), snapshot.PlanID)
	require.Equal(t, 14, snapshot.ValidityDays)
	require.InDelta(t, 25.0, snapshot.QuotaUSD, 1e-9)
	require.Equal(t, &daily, snapshot.DailyLimitUSD)
	require.Equal(t, &weekly, snapshot.WeeklyLimitUSD)
	require.Equal(t, SubscriptionScopePlatforms, snapshot.ScopeType)
	require.Equal(t, plan.ScopeConfig, snapshot.ScopeConfig)
}

func TestSubscriptionOrderSnapshotFromPlanRejectsInvalidQuota(t *testing.T) {
	_, err := subscriptionOrderSnapshotFromPlan(&dbent.SubscriptionPlan{
		ID:           42,
		QuotaUsd:     0,
		ScopeType:    SubscriptionScopeAllAvailableGroups,
		ScopeConfig:  map[string]any{},
		ValidityDays: 30,
		ValidityUnit: "day",
	})

	require.Error(t, err)
	require.Equal(t, "PLAN_QUOTA_INVALID", infraerrors.Reason(err))
}

func TestSubscriptionRenewalNotAllowedErrorIncludesRemainingAndExpiry(t *testing.T) {
	expiresAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	err := subscriptionRenewalNotAllowedError(RenewalEligibility{
		Allowed: false,
		Reason:  RenewalReasonNotExhausted,
		Subscription: &UserSubscription{
			ID:            99,
			QuotaLimitUSD: 10,
			QuotaUsedUSD:  4.25,
			ExpiresAt:     expiresAt,
		},
	})

	appErr := infraerrors.FromError(err)
	require.Equal(t, "SUBSCRIPTION_RENEWAL_NOT_ALLOWED", appErr.Reason)
	require.Equal(t, "99", appErr.Metadata["subscription_id"])
	require.Equal(t, "5.7500000000", appErr.Metadata["quota_remaining_usd"])
	require.Equal(t, expiresAt.Format(time.RFC3339), appErr.Metadata["expires_at"])
}

type paymentOrderRenewalRepoStub struct {
	userSubRepoNoop
	called      bool
	eligibility RenewalEligibility
}

func (r *paymentOrderRenewalRepoStub) GetRenewalEligibility(ctx context.Context, userID int64) (RenewalEligibility, error) {
	r.called = true
	return r.eligibility, nil
}

func TestValidateSubOrderAllowsUsableSubscriptionWhenMultiplePurchasesEnabled(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	plan, err := client.SubscriptionPlan.Create().
		SetName("Starter").
		SetDescription("starter plan").
		SetPrice(10).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetFeatures("[]").
		SetProductName("Starter Plan").
		SetQuotaUsd(25).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	repo := &paymentOrderRenewalRepoStub{
		eligibility: RenewalEligibility{
			Allowed: false,
			Reason:  RenewalReasonNotExhausted,
			Subscription: &UserSubscription{
				ID:            99,
				QuotaLimitUSD: 10,
				QuotaUsedUSD:  1,
				ExpiresAt:     time.Now().Add(time.Hour),
			},
		},
	}
	svc := &PaymentService{
		configService: &PaymentConfigService{entClient: client},
		subscriptionSvc: NewSubscriptionService(
			nil,
			repo,
			nil,
			nil,
			nil,
		),
	}
	svc.subscriptionMultiplePurchasesEnabled = func(context.Context) bool { return true }

	got, err := svc.validateSubOrder(ctx, CreateOrderRequest{
		UserID:    7,
		OrderType: "subscription",
		PlanID:    plan.ID,
	})

	require.NoError(t, err)
	require.Equal(t, plan.ID, got.ID)
	require.False(t, repo.called, "multiple purchase mode should skip renewal eligibility blocking")
}
