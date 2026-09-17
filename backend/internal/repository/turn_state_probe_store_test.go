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

func TestTurnStateProbeStoreTicketAndBind(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewTurnStateProbeStore(rdb)
	ctx := context.Background()

	got, err := store.Get(ctx, 42)
	require.NoError(t, err)
	require.Nil(t, got)

	rec := service.TurnStateTicketRecord{
		AccountID:      42,
		State:          "harvested-blob",
		Model:          "gpt-6-astra",
		PolicyRevision: 3,
		Status:         "holding",
		RecheckAt:      time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, store.Put(ctx, rec))
	stored, err := store.Get(ctx, 42)
	require.NoError(t, err)
	require.Equal(t, "harvested-blob", stored.State)
	require.Equal(t, service.TurnStateProbeStateHash("harvested-blob"), stored.StateHash)
	require.Equal(t, len("harvested-blob"), stored.StateLength)

	first, err := store.BindTurn(ctx, 42, "s:one", "harvested-blob", time.Hour)
	require.NoError(t, err)
	require.Equal(t, "harvested-blob", first)
	second, err := store.BindTurn(ctx, 42, "s:one", "newer-blob", time.Hour)
	require.NoError(t, err)
	require.Equal(t, "harvested-blob", second)

	ok, err := store.TryLock(ctx, 42, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = store.TryLock(ctx, 42, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, store.Unlock(ctx, 42))

	allowed, err := store.AllowRPM(ctx, "global", 2)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = store.AllowRPM(ctx, "global", 2)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = store.AllowRPM(ctx, "global", 2)
	require.NoError(t, err)
	require.False(t, allowed)

	require.NoError(t, store.Delete(ctx, 42))
	got, err = store.Get(ctx, 42)
	require.NoError(t, err)
	require.Nil(t, got)
}
