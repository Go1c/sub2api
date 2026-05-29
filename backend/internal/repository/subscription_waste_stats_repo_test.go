package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWasteStatsWhere_ExcludesRevokedSubscriptions(t *testing.T) {
	where, args := wasteStatsWhere(service.WasteStatsQuery{
		StartTime: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})

	require.Contains(t, where, "us.status <> $3")
	require.Contains(t, where, "us.deleted_at IS NULL")
	require.Len(t, args, 3)
	require.Equal(t, service.SubscriptionStatusRevoked, args[2])
}
