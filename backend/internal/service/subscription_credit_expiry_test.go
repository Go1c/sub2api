package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type subscriptionCreditExpiryRepoStub struct {
	userSubRepoNoop

	creditExpiryCalled bool
	legacyExpiryCalled bool
	updated            int64
}

func (r *subscriptionCreditExpiryRepoStub) ExpireCreditSubscriptions(context.Context) (int64, error) {
	r.creditExpiryCalled = true
	return r.updated, nil
}

func (r *subscriptionCreditExpiryRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	r.legacyExpiryCalled = true
	return 0, nil
}

func TestSubscriptionCreditExpiryRunOnceUsesCreditExpiryPath(t *testing.T) {
	repo := &subscriptionCreditExpiryRepoStub{updated: 2}
	svc := NewSubscriptionExpiryService(repo, time.Second)

	svc.runOnce()

	require.True(t, repo.creditExpiryCalled)
	require.False(t, repo.legacyExpiryCalled)
}
