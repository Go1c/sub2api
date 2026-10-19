package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAcquireSlotsTolerateRedis8AndCountLiveLeases(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewConcurrencyCache(client, 15, 900)
	proxy, ok := cache.(service.AccountProxySlotCache)
	require.True(t, ok)
	ctx := context.Background()

	userAcquired, err := cache.AcquireUserSlot(ctx, 21, 2, "user-req")
	require.NoError(t, err)
	require.True(t, userAcquired)
	accountAcquired, err := cache.AcquireAccountSlot(ctx, 11, 2, "account-req")
	require.NoError(t, err)
	require.True(t, accountAcquired)
	proxyAcquired, err := proxy.AcquireAccountProxySlot(ctx, 11, 7, 1, "proxy-req")
	require.NoError(t, err)
	require.True(t, proxyAcquired)
	proxyCount, err := proxy.GetAccountProxyConcurrency(ctx, 11, 7)
	require.NoError(t, err)
	require.Equal(t, 1, proxyCount)

	require.NoError(t, client.ZAdd(ctx, liveUserSlotKey(22), redis.Z{Score: float64(int64(1) << 62), Member: "live-user"}).Err())
	require.NoError(t, client.ZAdd(ctx, liveAccountSlotKey(12), redis.Z{Score: float64(int64(1) << 62), Member: "live-account"}).Err())

	userAcquired, err = cache.AcquireUserSlot(ctx, 22, 1, "user-blocked")
	require.NoError(t, err)
	require.False(t, userAcquired)
	accountAcquired, err = cache.AcquireAccountSlot(ctx, 12, 1, "account-blocked")
	require.NoError(t, err)
	require.False(t, accountAcquired)
}
