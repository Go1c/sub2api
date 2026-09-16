package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	requestHealthAccountKeyPrefix     = "reqhealth:acct:"
	requestHealthAccountProxyKeyInfix = ":proxy:"
)

type redisRequestHealthStore struct {
	rdb *redis.Client
}

type noopRequestHealthStore struct{}

func NewAccountRequestHealthStore(rdb *redis.Client) RequestHealthStore {
	if rdb == nil {
		return noopRequestHealthStore{}
	}
	return &redisRequestHealthStore{rdb: rdb}
}

func requestHealthAccountKey(accountID int64) string {
	return requestHealthAccountKeyPrefix + strconv.FormatInt(accountID, 10)
}

func requestHealthAccountProxyKey(accountID, proxyID int64) string {
	return requestHealthAccountKey(accountID) + requestHealthAccountProxyKeyInfix + strconv.FormatInt(proxyID, 10)
}

func (s *redisRequestHealthStore) Append(ctx context.Context, ev RequestHealthEvent) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	accountKey := requestHealthAccountKey(ev.AccountID)
	pipe.LPush(ctx, accountKey, raw)
	pipe.LTrim(ctx, accountKey, 0, RequestHealthMaxEvents-1)
	pipe.Expire(ctx, accountKey, requestHealthTTL)
	if ev.ProxyID > 0 {
		proxyKey := requestHealthAccountProxyKey(ev.AccountID, ev.ProxyID)
		pipe.LPush(ctx, proxyKey, raw)
		pipe.LTrim(ctx, proxyKey, 0, RequestHealthMaxEvents-1)
		pipe.Expire(ctx, proxyKey, requestHealthTTL)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisRequestHealthStore) List(ctx context.Context, accountID, proxyID int64, limit int) ([]RequestHealthEvent, error) {
	if limit <= 0 {
		limit = RequestHealthMaxEvents
	}
	key := requestHealthAccountKey(accountID)
	if proxyID > 0 {
		key = requestHealthAccountProxyKey(accountID, proxyID)
	}
	vals, err := s.rdb.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	events := make([]RequestHealthEvent, 0, len(vals))
	for _, val := range vals {
		var ev RequestHealthEvent
		if unmarshalErr := json.Unmarshal([]byte(val), &ev); unmarshalErr != nil {
			continue
		}
		events = append(events, ev)
	}
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}

func (s *redisRequestHealthStore) Runtime(ctx context.Context, accountID, proxyID int64) (int, *time.Time) {
	current := 0
	if proxyID > 0 {
		if n, err := s.rdb.Get(ctx, requestHealthConcurrencyKey(accountID, proxyID)).Int(); err == nil {
			current = n
		}
		ttl, err := s.rdb.PTTL(ctx, requestHealthCooldownKey(accountID, proxyID)).Result()
		if err == nil && ttl > 0 {
			until := time.Now().Add(ttl)
			return current, &until
		}
	}
	return current, nil
}

func requestHealthConcurrencyKey(accountID, proxyID int64) string {
	return "conc:acct-proxy:" + strconv.FormatInt(accountID, 10) + ":" + strconv.FormatInt(proxyID, 10)
}

func requestHealthCooldownKey(accountID, proxyID int64) string {
	return "openai:ip_group_cooldown:" + strconv.FormatInt(accountID, 10) + ":" + strconv.FormatInt(proxyID, 10)
}

func (noopRequestHealthStore) Append(context.Context, RequestHealthEvent) error {
	return nil
}

func (noopRequestHealthStore) List(context.Context, int64, int64, int) ([]RequestHealthEvent, error) {
	return nil, nil
}

func (noopRequestHealthStore) Runtime(context.Context, int64, int64) (int, *time.Time) {
	return 0, nil
}
