//go:build unit

package service

import (
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
