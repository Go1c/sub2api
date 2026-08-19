//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type balanceOutboxStub struct {
	events       []BalanceCacheInvalidationEvent
	completed    []int64
	failed       []int64
	failureRetry time.Time
}

func (s *balanceOutboxStub) Claim(context.Context, int, time.Duration) ([]BalanceCacheInvalidationEvent, error) {
	return s.events, nil
}
func (s *balanceOutboxStub) Complete(_ context.Context, id int64, _ string) error {
	s.completed = append(s.completed, id)
	return nil
}
func (s *balanceOutboxStub) Fail(_ context.Context, id int64, _ string, _ string, retryAt time.Time) error {
	s.failed = append(s.failed, id)
	s.failureRetry = retryAt
	return nil
}

type balanceBillingInvalidatorStub struct{ err error }

func (s *balanceBillingInvalidatorStub) InvalidateUserBalance(context.Context, int64) error {
	return s.err
}

type balanceAPIKeyInvalidatorStub struct{ err error }

func (s *balanceAPIKeyInvalidatorStub) InvalidateAuthCacheByUserIDStrict(context.Context, int64) error {
	return s.err
}

func TestBalanceCacheInvalidationWorkerRetriesUntilBothCachesSucceed(t *testing.T) {
	repo := &balanceOutboxStub{events: []BalanceCacheInvalidationEvent{{ID: 7, UserID: 42, Attempts: 1, ClaimToken: "claim"}}}
	worker := NewBalanceCacheInvalidationWorker(
		repo,
		&balanceBillingInvalidatorStub{err: errors.New("redis down")},
		&balanceAPIKeyInvalidatorStub{},
	)
	worker.now = func() time.Time { return time.Unix(1000, 0).UTC() }

	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Empty(t, repo.completed)
	require.Equal(t, []int64{7}, repo.failed)
	require.True(t, repo.failureRetry.After(worker.now()))

	repo.failed = nil
	worker.billing = &balanceBillingInvalidatorStub{}
	require.NoError(t, worker.ProcessOnce(context.Background()))
	require.Equal(t, []int64{7}, repo.completed)
	require.Empty(t, repo.failed)
}
