package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const openAIIPGroupBindKeyPrefix = "openai:ip_group_bind:"

const openAIIPGroupCooldownKeyPrefix = "openai:ip_group_cooldown:"

func openAIIPGroupBindKey(accountID int64, sessionHash string) string {
	return openAIIPGroupBindKeyPrefix + strconv.FormatInt(accountID, 10) + ":" + sessionHash
}

func openAIIPGroupCooldownKey(accountID, proxyID int64) string {
	return openAIIPGroupCooldownKeyPrefix + strconv.FormatInt(accountID, 10) + ":" + strconv.FormatInt(proxyID, 10)
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

func (s *openAIIPGroupBindStore) DeleteBoundProxyID(ctx context.Context, accountID int64, sessionHash string) error {
	if s == nil || s.rdb == nil || sessionHash == "" {
		return nil
	}
	return s.rdb.Del(ctx, openAIIPGroupBindKey(accountID, sessionHash)).Err()
}

func (s *openAIIPGroupBindStore) MarkProxyCooldown(ctx context.Context, accountID, proxyID int64, ttl time.Duration) error {
	if s == nil || s.rdb == nil || accountID <= 0 || proxyID <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return s.rdb.Set(ctx, openAIIPGroupCooldownKey(accountID, proxyID), "1", ttl).Err()
}

func (s *openAIIPGroupBindStore) IsProxyCoolingDown(ctx context.Context, accountID, proxyID int64) (bool, error) {
	if s == nil || s.rdb == nil || accountID <= 0 || proxyID <= 0 {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, openAIIPGroupCooldownKey(accountID, proxyID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
