//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type creditCacheRepoStub struct {
	userSubRepoNoop
	calls int
	sub   *UserSubscription
	err   error
}

func (r *creditCacheRepoStub) GetUsableCreditSubscription(_ context.Context, userID int64) (*UserSubscription, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	if r.sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	cp.UserID = userID
	return &cp, nil
}

func TestGetUsableCreditSubscriptionCachesMissAndInvalidatesOnPurchase(t *testing.T) {
	repo := &creditCacheRepoStub{err: ErrSubscriptionNotFound}
	svc := NewSubscriptionService(nil, repo, nil, nil, &config.Config{
		SubscriptionCache: config.SubscriptionCacheConfig{
			L1Size:       32,
			L1TTLSeconds: 60,
		},
	})

	_, err := svc.GetUsableCreditSubscription(context.Background(), 7)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	svc.subCacheL1.Wait()
	_, err = svc.GetUsableCreditSubscription(context.Background(), 7)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.Equal(t, 1, repo.calls, "negative result should be cached on the auth hot path")

	repo.err = nil
	repo.sub = &UserSubscription{
		ID:            91,
		UserID:        7,
		QuotaLimitUSD: 25,
		StartsAt:      time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(24 * time.Hour),
		Status:        SubscriptionStatusActive,
	}

	svc.InvalidateSubCache(7, 0)
	sub, err := svc.GetUsableCreditSubscription(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(91), sub.ID)
	require.Equal(t, 2, repo.calls, "purchase fulfillment must clear the cached miss so auth can see the new subscription")
}
