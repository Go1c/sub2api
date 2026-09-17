package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type turnStateProbeStore struct {
	rdb *redis.Client
}

func NewTurnStateProbeStore(rdb *redis.Client) service.TurnStateTicketStore {
	return &turnStateProbeStore{rdb: rdb}
}

func turnStateTicketKey(accountID int64) string {
	return fmt.Sprintf("kin:tsp:{%d}:ticket", accountID)
}

func turnStateBindKey(accountID int64, turnKey string) string {
	return fmt.Sprintf("kin:tsp:{%d}:bind:%s", accountID, turnKey)
}

func turnStateLockKey(accountID int64) string {
	return fmt.Sprintf("kin:tsp:{%d}:lock", accountID)
}

func turnStateRPMKey(bucket string, minute time.Time) string {
	return fmt.Sprintf("kin:tsp:rpm:%s:%s", bucket, minute.UTC().Format("200601021504"))
}

func (s *turnStateProbeStore) Get(ctx context.Context, accountID int64) (*service.TurnStateTicketRecord, error) {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return nil, nil
	}
	raw, err := s.rdb.Get(ctx, turnStateTicketKey(accountID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec service.TurnStateTicketRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *turnStateProbeStore) Put(ctx context.Context, rec service.TurnStateTicketRecord) error {
	if s == nil || s.rdb == nil || rec.AccountID <= 0 {
		return nil
	}
	if rec.StateHash == "" {
		rec.StateHash = service.TurnStateProbeStateHash(rec.State)
	}
	if rec.StateLength == 0 {
		rec.StateLength = len(strings.TrimSpace(rec.State))
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = time.Now()
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	ttl := 48 * time.Hour
	if !rec.RecheckAt.IsZero() {
		if wait := time.Until(rec.RecheckAt) + 24*time.Hour; wait > ttl {
			ttl = wait
		}
	}
	return s.rdb.Set(ctx, turnStateTicketKey(rec.AccountID), raw, ttl).Err()
}

func (s *turnStateProbeStore) Delete(ctx context.Context, accountID int64) error {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return nil
	}
	return s.rdb.Del(ctx, turnStateTicketKey(accountID)).Err()
}

func (s *turnStateProbeStore) BindTurn(ctx context.Context, accountID int64, turnKey, state string, ttl time.Duration) (string, error) {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return state, nil
	}
	turnKey = strings.TrimSpace(turnKey)
	state = strings.TrimSpace(state)
	if turnKey == "" || state == "" {
		return state, nil
	}
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	key := turnStateBindKey(accountID, turnKey)
	ok, err := s.rdb.SetNX(ctx, key, state, ttl).Result()
	if err != nil {
		return "", err
	}
	if ok {
		return state, nil
	}
	existing, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return state, nil
	}
	if err != nil {
		return "", err
	}
	if existing == "" {
		return state, nil
	}
	return existing, nil
}

func (s *turnStateProbeStore) GetTurnBind(ctx context.Context, accountID int64, turnKey string) (string, error) {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return "", nil
	}
	turnKey = strings.TrimSpace(turnKey)
	if turnKey == "" {
		return "", nil
	}
	val, err := s.rdb.Get(ctx, turnStateBindKey(accountID, turnKey)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (s *turnStateProbeStore) TryLock(ctx context.Context, accountID int64, ttl time.Duration) (bool, error) {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return false, nil
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return s.rdb.SetNX(ctx, turnStateLockKey(accountID), "1", ttl).Result()
}

func (s *turnStateProbeStore) Unlock(ctx context.Context, accountID int64) error {
	if s == nil || s.rdb == nil || accountID <= 0 {
		return nil
	}
	return s.rdb.Del(ctx, turnStateLockKey(accountID)).Err()
}

func (s *turnStateProbeStore) AllowRPM(ctx context.Context, bucket string, rpm int) (bool, error) {
	if s == nil || s.rdb == nil {
		return true, nil
	}
	if rpm < 1 {
		rpm = 1
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		bucket = "global"
	}
	key := turnStateRPMKey(bucket, time.Now())
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		_ = s.rdb.Expire(ctx, key, 2*time.Minute).Err()
	}
	return n <= int64(rpm), nil
}
