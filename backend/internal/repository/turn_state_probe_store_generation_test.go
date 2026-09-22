package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTurnStateProbeStoreRejectsOlderGeneration(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewTurnStateProbeStore(rdb)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	current := service.TurnStateTicketRecord{
		AccountID: 9, State: "fresh", Status: "holding", Generation: 2, UpdatedAt: now,
		ExitDigest: "aa", HarvestedAt: now, ExpiresAt: now.Add(4 * time.Minute),
	}
	require.NoError(t, store.Put(ctx, current))

	older := current
	older.Generation = 1
	older.State = "stale"
	older.UpdatedAt = now.Add(time.Minute)
	require.NoError(t, store.Put(ctx, older))

	stored, err := store.Get(ctx, 9)
	require.NoError(t, err)
	require.Equal(t, "fresh", stored.State)
	require.Equal(t, int64(2), stored.Generation)
	require.Equal(t, "aa", stored.ExitDigest)
	require.NotContains(t, stored.State, "cookie")

	sameGenOlder := current
	sameGenOlder.UpdatedAt = now.Add(-time.Minute)
	sameGenOlder.State = "rewound"
	require.NoError(t, store.Put(ctx, sameGenOlder))
	stored, err = store.Get(ctx, 9)
	require.NoError(t, err)
	require.Equal(t, "fresh", stored.State)
}
