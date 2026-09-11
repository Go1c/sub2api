package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const openAIIPGroupBindKeyPrefix = "openai:ip_group_bind:"

func openAIIPGroupBindKey(accountID int64, sessionHash string) string {
	return openAIIPGroupBindKeyPrefix + strconv.FormatInt(accountID, 10) + ":" + sessionHash
}

type openAIIPGroupBindStore struct {
	rdb *redis.Client
}

// NewOpenAIIPGroupBindStore persists conversation→proxy bindings in Redis.
func NewOpenAIIPGroupBindStore(rdb *redis.Client) service.OpenAIIPGroupBindStore {
	if rdb == nil {
		return nil
	}
	return &openAIIPGroupBindStore{rdb: rdb}
}

func (s *openAIIPGroupBindStore) GetBoundProxyID(ctx context.Context, accountID int64, sessionHash string) (int64, bool, error) {
	if s == nil || s.rdb == nil || sessionHash == "" {
		return 0, false, nil
	}
	val, err := s.rdb.Get(ctx, openAIIPGroupBindKey(accountID, sessionHash)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil || id <= 0 {
		return 0, false, nil
	}
	return id, true, nil
}

func (s *openAIIPGroupBindStore) SetBoundProxyID(ctx context.Context, accountID int64, sessionHash string, proxyID int64, ttl time.Duration) error {
	if s == nil || s.rdb == nil || sessionHash == "" || proxyID <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return s.rdb.Set(ctx, openAIIPGroupBindKey(accountID, sessionHash), strconv.FormatInt(proxyID, 10), ttl).Err()
}
