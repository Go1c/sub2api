//go:build unit

package handler

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCheckoutPlanFromSubscriptionPlanExposesCreditPoolFields(t *testing.T) {
	daily := 3.5
	weekly := 12.5
	groupDaily := 1.0
	groupWeekly := 2.0
	monthly := 30.0
	plan := &dbent.SubscriptionPlan{
		ID:             42,
		Name:           "Credit Pool",
		Price:          19.9,
		QuotaUsd:       25,
		DailyLimitUsd:  &daily,
		WeeklyLimitUsd: &weekly,
		ScopeType:      service.SubscriptionScopePlatforms,
		ScopeConfig:    map[string]any{"platforms": []any{service.PlatformAnthropic}},
		ValidityDays:   30,
		ValidityUnit:   "day",
	}

	got := checkoutPlanFromSubscriptionPlan(plan, 0, service.PlanGroupInfo{
		Platform:        service.PlatformAnthropic,
		Name:            "legacy group",
		RateMultiplier:  1.5,
		DailyLimitUSD:   &groupDaily,
		WeeklyLimitUSD:  &groupWeekly,
		MonthlyLimitUSD: &monthly,
		ModelScopes:     []string{"claude-*"},
	})

	require.Equal(t, int64(42), got.ID)
	require.Equal(t, int64(0), got.GroupID)
	require.InDelta(t, 25.0, got.QuotaUSD, 1e-9)
	require.Equal(t, &daily, got.DailyLimitUSD)
	require.Equal(t, &weekly, got.WeeklyLimitUSD)
	require.Equal(t, &monthly, got.MonthlyLimitUSD)
	require.Equal(t, service.SubscriptionScopePlatforms, got.ScopeType)
	require.Equal(t, plan.ScopeConfig, got.ScopeConfig)
	require.Equal(t, 30, got.ValidityDays)
}
