package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const desktopPaymentHandoffKeyPrefix = "desktop_payment_handoff:"

type desktopPaymentHandoffStore struct {
	rdb *redis.Client
}

func NewDesktopPaymentHandoffStore(rdb *redis.Client) service.DesktopPaymentHandoffStore {
	return &desktopPaymentHandoffStore{rdb: rdb}
}

func (s *desktopPaymentHandoffStore) Store(
	ctx context.Context,
	tokenHash string,
	data service.DesktopPaymentHandoffData,
	ttl time.Duration,
) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal desktop payment handoff: %w", err)
	}
	return s.rdb.Set(ctx, desktopPaymentHandoffKeyPrefix+tokenHash, raw, ttl).Err()
}

func (s *desktopPaymentHandoffStore) Consume(
	ctx context.Context,
	tokenHash string,
) (*service.DesktopPaymentHandoffData, error) {
	raw, err := s.rdb.GetDel(ctx, desktopPaymentHandoffKeyPrefix+tokenHash).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrDesktopPaymentHandoffInvalid
	}
	if err != nil {
		return nil, err
	}

	var data service.DesktopPaymentHandoffData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("unmarshal desktop payment handoff: %w", err)
	}
	return &data, nil
}
