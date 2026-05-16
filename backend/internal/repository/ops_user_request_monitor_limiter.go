package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type opsUserRequestRedisLimiter struct {
	client *redis.Client
}

func NewOpsUserRequestCaptureLimiter(client *redis.Client) service.OpsUserRequestCaptureLimiter {
	if client == nil {
		return nil
	}
	return &opsUserRequestRedisLimiter{client: client}
}

func (l *opsUserRequestRedisLimiter) Allow(ctx context.Context, monitorID int64, captureMinute time.Time, maxPerMinute int) (bool, error) {
	if l == nil || l.client == nil {
		return false, fmt.Errorf("redis client unavailable")
	}
	key := fmt.Sprintf("ops:user-request-monitor:%d:%s", monitorID, captureMinute.UTC().Format("200601021504"))
	n, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := l.client.Expire(ctx, key, 2*time.Minute).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(maxPerMinute), nil
}
