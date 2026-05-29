package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserSubscriptionFromServiceIncludesCreditPoolFields(t *testing.T) {
	now := time.Now().UTC()
	planID := int64(77)
	dailyLimit := 10.0
	weeklyLimit := 50.0
	dailyWindow := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	weeklyWindow := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)

	out := UserSubscriptionFromService(&service.UserSubscription{
		ID:                 123,
		UserID:             456,
		GroupID:            nil,
		PlanID:             &planID,
		ScopeType:          service.SubscriptionScopeSelectedGroups,
		ScopeConfig:        map[string]any{"group_ids": []any{float64(1), float64(2)}},
		QuotaLimitUSD:      100,
		QuotaUsedUSD:       25,
		DailyLimitUSD:      &dailyLimit,
		WeeklyLimitUSD:     &weeklyLimit,
		StartsAt:           now.Add(-time.Hour),
		ExpiresAt:          now.Add(24 * time.Hour),
		Status:             service.SubscriptionStatusActive,
		DailyWindowStart:   &dailyWindow,
		WeeklyWindowStart:  &weeklyWindow,
		DailyUsageUSD:      3,
		WeeklyUsageUSD:     12,
		Recent30dWastedUSD: 3.25,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	require.NotNil(t, out)
	require.Nil(t, out.GroupID)
	require.Equal(t, planID, *out.PlanID)
	require.True(t, out.IsUsable)
	require.Nil(t, out.ExhaustedAt)
	require.Equal(t, 100.0, out.QuotaLimitUSD)
	require.Equal(t, 25.0, out.QuotaUsedUSD)
	require.Equal(t, 75.0, out.QuotaRemainingUSD)
	require.Equal(t, 3.25, out.Recent30dWastedUSD)
	require.Equal(t, dailyLimit, *out.DailyLimitUSD)
	require.Equal(t, 3.0, out.DailyUsageUSD)
	require.NotNil(t, out.DailyResetAt)
	require.Equal(t, dailyWindow.Add(24*time.Hour), *out.DailyResetAt)
	require.Equal(t, weeklyLimit, *out.WeeklyLimitUSD)
	require.Equal(t, 12.0, out.WeeklyUsageUSD)
	require.NotNil(t, out.WeeklyResetAt)
	require.Equal(t, weeklyWindow.Add(7*24*time.Hour), *out.WeeklyResetAt)
	require.Equal(t, service.SubscriptionScopeSelectedGroups, out.ScopeType)
	require.NotNil(t, out.ScopeConfig)
}
