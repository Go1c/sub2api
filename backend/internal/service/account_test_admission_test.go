package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountTestCooldownBlocksUntilLatestBoundary(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	rateLimitUntil := now.Add(10 * time.Minute)
	overloadUntil := now.Add(2 * time.Minute)
	account := &Account{
		ID:                7,
		RateLimitResetAt:  &rateLimitUntil,
		OverloadUntil:     &overloadUntil,
		TempUnschedulableUntil: nil,
	}

	err := accountTestCooldown(context.Background(), account, "", now)
	var wait *TestAdmissionWaitError
	require.ErrorAs(t, err, &wait)
	require.Equal(t, "账号限流冷却尚未结束", wait.Reason)
	require.Equal(t, rateLimitUntil, wait.Until)
	require.NoError(t, accountTestCooldown(context.Background(), &Account{ID: 8}, "", now))
}

func TestAccountTestCooldownRequiresAccount(t *testing.T) {
	err := accountTestCooldown(context.Background(), nil, "", time.Now())
	require.EqualError(t, err, "账号不存在")
	require.False(t, errors.As(err, new(*TestAdmissionWaitError)))
}
