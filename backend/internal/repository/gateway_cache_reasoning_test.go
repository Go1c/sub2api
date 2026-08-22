//go:build unit

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

func TestGatewayCacheReasoningContent(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewGatewayCache(client)
	ctx := context.Background()

	_, err := cache.GetReasoningContent(ctx, "item_missing")
	require.ErrorIs(t, err, service.ErrReasoningContentNotFound)

	require.NoError(t, cache.SetReasoningContent(ctx, "item_abc", "think hard", time.Minute))
	got, err := cache.GetReasoningContent(ctx, "item_abc")
	require.NoError(t, err)
	require.Equal(t, "think hard", got)

	require.NoError(t, cache.SetReasoningContent(ctx, "item_ttl", "x", 0))
	ttl := mr.TTL(reasoningContentPrefix + "item_ttl")
	require.Greater(t, ttl, 6*24*time.Hour)
	require.LessOrEqual(t, ttl, reasoningContentDefaultTTL)

	require.NoError(t, cache.SetReasoningContent(ctx, "", "x", time.Minute))
	require.NoError(t, cache.SetReasoningContent(ctx, "item_empty", "", time.Minute))
	_, err = cache.GetReasoningContent(ctx, "item_empty")
	require.ErrorIs(t, err, service.ErrReasoningContentNotFound)
}
