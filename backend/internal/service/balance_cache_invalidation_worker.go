package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type balanceBillingCacheInvalidator interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

type balanceAPIKeyCacheInvalidator interface {
	InvalidateAuthCacheByUserIDStrict(ctx context.Context, userID int64) error
}

type BalanceCacheInvalidationWorker struct {
	repo    BalanceCacheInvalidationOutboxRepository
	billing balanceBillingCacheInvalidator
	apiKeys balanceAPIKeyCacheInvalidator
	now     func() time.Time

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewBalanceCacheInvalidationWorker(
	repo BalanceCacheInvalidationOutboxRepository,
	billing balanceBillingCacheInvalidator,
	apiKeys balanceAPIKeyCacheInvalidator,
) *BalanceCacheInvalidationWorker {
	return &BalanceCacheInvalidationWorker{
		repo: repo, billing: billing, apiKeys: apiKeys,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (w *BalanceCacheInvalidationWorker) Start() {
	if w == nil || w.repo == nil || w.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if err := w.ProcessOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("balance cache invalidation worker failed", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *BalanceCacheInvalidationWorker) Stop() {
	if w == nil || w.cancel == nil {
		return
	}
	w.cancel()
	w.wg.Wait()
}

func (w *BalanceCacheInvalidationWorker) ProcessOnce(ctx context.Context) error {
	if w == nil || w.repo == nil {
		return nil
	}
	events, err := w.repo.Claim(ctx, 50, 30*time.Second)
	if err != nil {
		return err
	}
	for _, event := range events {
		eventErr := w.invalidate(ctx, event.UserID)
		if eventErr == nil {
			if err := w.repo.Complete(ctx, event.ID, event.ClaimToken); err != nil {
				return err
			}
			continue
		}
		retryAt := w.now().Add(balanceCacheRetryDelay(event.Attempts + 1))
		if err := w.repo.Fail(ctx, event.ID, event.ClaimToken, eventErr.Error(), retryAt); err != nil {
			return err
		}
	}
	return nil
}

func (w *BalanceCacheInvalidationWorker) invalidate(ctx context.Context, userID int64) error {
	if w.billing != nil {
		if err := w.billing.InvalidateUserBalance(ctx, userID); err != nil {
			return err
		}
	}
	if w.apiKeys != nil {
		if err := w.apiKeys.InvalidateAuthCacheByUserIDStrict(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

func balanceCacheRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 9 {
		attempt = 9
	}
	delay := time.Second * time.Duration(1<<uint(attempt-1))
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

func ProvideBalanceCacheInvalidationWorker(
	repo BalanceCacheInvalidationOutboxRepository,
	billing *BillingCacheService,
	apiKeys *APIKeyService,
) *BalanceCacheInvalidationWorker {
	worker := NewBalanceCacheInvalidationWorker(repo, billing, apiKeys)
	worker.Start()
	return worker
}
