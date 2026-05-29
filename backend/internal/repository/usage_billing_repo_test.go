package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionQuotaWindowStarts_DefaultMatchesUTC(t *testing.T) {
	now := time.Date(2026, 5, 29, 15, 30, 0, 0, time.UTC)
	cfg := service.SubscriptionQuotaResetConfig{}

	daily, weekly := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC), daily)
	require.Equal(t, time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), weekly)
}

func TestSubscriptionQuotaWindowStarts_UsesUTCOffsetAndResetHour(t *testing.T) {
	now := time.Date(2026, 5, 28, 19, 30, 0, 0, time.UTC) // UTC+08 local 03:30, before 04:00 reset.
	cfg := service.SubscriptionQuotaResetConfig{UTCOffsetMinutes: 8 * 60, ResetHour: 4}

	daily, _ := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 27, 20, 0, 0, 0, time.UTC), daily)
}

func TestSubscriptionQuotaWindowStarts_WeeklyUsesMondayInConfiguredOffset(t *testing.T) {
	now := time.Date(2026, 5, 31, 23, 30, 0, 0, time.UTC) // UTC+08 Monday 07:30.
	cfg := service.SubscriptionQuotaResetConfig{UTCOffsetMinutes: 8 * 60, ResetHour: 4}

	_, weekly := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 31, 20, 0, 0, 0, time.UTC), weekly)
}
