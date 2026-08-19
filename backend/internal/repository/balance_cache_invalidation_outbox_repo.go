package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type balanceCacheInvalidationOutboxRepository struct {
	db *sql.DB
}

func NewBalanceCacheInvalidationOutboxRepository(db *sql.DB) service.BalanceCacheInvalidationOutboxRepository {
	return &balanceCacheInvalidationOutboxRepository{db: db}
}

func (r *balanceCacheInvalidationOutboxRepository) Claim(ctx context.Context, limit int, lease time.Duration) ([]service.BalanceCacheInvalidationEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if lease <= 0 {
		lease = 30 * time.Second
	}
	claimToken := uuid.NewString()
	leaseInterval := fmt.Sprintf("%.3f seconds", lease.Seconds())
	rows, err := r.db.QueryContext(ctx, `
		WITH candidates AS (
			SELECT id FROM balance_cache_invalidation_outbox
			WHERE next_attempt_at <= NOW()
			  AND (claimed_at IS NULL OR claimed_at < NOW() - $1::interval)
			ORDER BY next_attempt_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE balance_cache_invalidation_outbox o
		SET claimed_at = NOW(), claim_token = $3, updated_at = NOW()
		FROM candidates c
		WHERE o.id = c.id
		RETURNING o.id, o.user_id, o.attempts, o.claim_token`, leaseInterval, limit, claimToken)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	defer func() { _ = rows.Close() }()
	events := make([]service.BalanceCacheInvalidationEvent, 0, limit)
	for rows.Next() {
		var event service.BalanceCacheInvalidationEvent
		if err := rows.Scan(&event.ID, &event.UserID, &event.Attempts, &event.ClaimToken); err != nil {
			return nil, mapBalanceStoreError(err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return events, nil
}

func (r *balanceCacheInvalidationOutboxRepository) Complete(ctx context.Context, id int64, claimToken string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM balance_cache_invalidation_outbox
		WHERE id = $1 AND claim_token = $2`, id, claimToken)
	if err != nil {
		return mapBalanceStoreError(err)
	}
	return nil
}

func (r *balanceCacheInvalidationOutboxRepository) Fail(ctx context.Context, id int64, claimToken, lastError string, retryAt time.Time) error {
	lastError = strings.TrimSpace(lastError)
	if len(lastError) > 512 {
		lastError = lastError[:512]
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE balance_cache_invalidation_outbox
		SET attempts = attempts + 1, next_attempt_at = $3, claimed_at = NULL,
		    claim_token = NULL, last_error = $4, updated_at = NOW()
		WHERE id = $1 AND claim_token = $2`, id, claimToken, retryAt.UTC(), lastError)
	if err != nil {
		return mapBalanceStoreError(err)
	}
	return nil
}

var _ service.BalanceCacheInvalidationOutboxRepository = (*balanceCacheInvalidationOutboxRepository)(nil)
