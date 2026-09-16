package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRequestHealthRuntimeUsesKinConcurrency(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	ctx := t.Context()
	cache := NewConcurrencyCache(rdb, 15, 0)
	proxyCache := cache.(service.AccountProxySlotCache)
	ok, err := proxyCache.AcquireAccountProxySlot(ctx, 11, 42, 3, "active")
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, rdb.ZAdd(ctx, "conc:acct-proxy:11:42", redis.Z{Score: float64(time.Now().Add(-time.Hour).Unix()), Member: "expired"}).Err())
	require.NoError(t, rdb.Set(ctx, "openai:ip_group_cooldown:11:42", "1", time.Minute).Err())
	store := service.NewAccountRequestHealthStore(rdb, cache)
	current, cooldown := store.Runtime(ctx, 11, 42)
	require.Equal(t, 1, current)
	require.NotNil(t, cooldown)
	ok, err = cache.AcquireAccountSlot(ctx, 11, 3, "single")
	require.NoError(t, err)
	require.True(t, ok)
	current, _ = store.Runtime(ctx, 11, 0)
	require.Equal(t, 1, current)
}
