//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type userResetWeeklyLimitRepoStub struct {
	userSubRepoNoop

	sub *UserSubscription

	casCalled    bool
	casUserID    int64
	casSubID     int64
	casWindow    time.Time
	casResetAt   time.Time
	casAffected  int
	casErr       error
	getByIDCalls int
}

func (r *userResetWeeklyLimitRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	r.getByIDCalls++
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	if r.sub.WeeklyLimitUSD != nil {
		v := *r.sub.WeeklyLimitUSD
		cp.WeeklyLimitUSD = &v
	}
	if r.sub.WeeklyLimitUserResetAt != nil {
		t := *r.sub.WeeklyLimitUserResetAt
		cp.WeeklyLimitUserResetAt = &t
	}
	if r.sub.WeeklyWindowStart != nil {
		t := *r.sub.WeeklyWindowStart
		cp.WeeklyWindowStart = &t
	}
	if r.sub.DailyWindowStart != nil {
		t := *r.sub.DailyWindowStart
		cp.DailyWindowStart = &t
	}
	if r.sub.GroupID != nil {
		g := *r.sub.GroupID
		cp.GroupID = &g
	}
	return &cp, nil
}

func (r *userResetWeeklyLimitRepoStub) UserResetWeeklyLimit(
	_ context.Context,
	subscriptionID, userID int64,
	windowStart, resetAt time.Time,
) (int, error) {
	r.casCalled = true
	r.casSubID = subscriptionID
	r.casUserID = userID
	r.casWindow = windowStart
	r.casResetAt = resetAt
	if r.casErr != nil {
		return 0, r.casErr
	}
	if r.casAffected == 0 {
		return 0, nil
	}
	if r.sub != nil {
		r.sub.WeeklyUsageUSD = 0
		ws := windowStart
		r.sub.WeeklyWindowStart = &ws
		ra := resetAt
		r.sub.WeeklyLimitUserResetAt = &ra
	}
	return r.casAffected, nil
}

func newUserResetWeeklyLimitSvc(stub *userResetWeeklyLimitRepoStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func usableWeeklySub(id, userID int64) *UserSubscription {
	weekly := 50.0
	gid := int64(20)
	oldWeeklyStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	oldDailyStart := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	return &UserSubscription{
		ID:                id,
		UserID:            userID,
		GroupID:           &gid,
		Status:            SubscriptionStatusActive,
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		WeeklyLimitUSD:    &weekly,
		WeeklyUsageUSD:    12.5,
		QuotaLimitUSD:     100,
		QuotaUsedUSD:      30,
		DailyUsageUSD:     5,
		DailyWindowStart:  &oldDailyStart,
		WeeklyWindowStart: &oldWeeklyStart,
	}
}

func TestUserResetWeeklyLimit_Success(t *testing.T) {
	sub := usableWeeklySub(1, 10)
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	beforeQuotaUsed := sub.QuotaUsedUSD
	beforeDailyUsage := sub.DailyUsageUSD
	beforeDailyWindow := *sub.DailyWindowStart

	result, err := svc.UserResetWeeklyLimit(context.Background(), 10, 1)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.casCalled)
	require.Equal(t, int64(1), stub.casSubID)
	require.Equal(t, int64(10), stub.casUserID)
	require.Equal(t, float64(0), result.WeeklyUsageUSD)
	require.NotNil(t, result.WeeklyWindowStart)
	require.Equal(t, startOfDay(time.Now()).UTC().Truncate(time.Second), result.WeeklyWindowStart.UTC().Truncate(time.Second))
	require.NotNil(t, result.WeeklyLimitUserResetAt)
	require.Equal(t, beforeQuotaUsed, result.QuotaUsedUSD, "quota_used_usd must not change")
	require.Equal(t, beforeDailyUsage, result.DailyUsageUSD, "daily usage must not change")
	require.NotNil(t, result.DailyWindowStart)
	require.Equal(t, beforeDailyWindow, *result.DailyWindowStart, "daily window must not change")
	require.Equal(t, 100.0, result.QuotaLimitUSD)
}

func TestUserResetWeeklyLimit_AlreadyUsed(t *testing.T) {
	sub := usableWeeklySub(2, 10)
	usedAt := time.Now().Add(-time.Hour)
	sub.WeeklyLimitUserResetAt = &usedAt
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 2)

	require.ErrorIs(t, err, ErrSubscriptionWeeklyLimitResetExhausted)
	require.False(t, stub.casCalled, "should short-circuit before CAS when already used")
}

func TestUserResetWeeklyLimit_NotOwner(t *testing.T) {
	sub := usableWeeklySub(3, 99)
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 3)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.casCalled)
}

func TestUserResetWeeklyLimit_NotFound(t *testing.T) {
	stub := &userResetWeeklyLimitRepoStub{sub: nil}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 999)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.casCalled)
}

func TestUserResetWeeklyLimit_NoWeeklyLimit(t *testing.T) {
	sub := usableWeeklySub(4, 10)
	sub.WeeklyLimitUSD = nil
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 4)

	require.ErrorIs(t, err, ErrSubscriptionNoWeeklyLimit)
	require.False(t, stub.casCalled)
}

func TestUserResetWeeklyLimit_NotUsable(t *testing.T) {
	sub := usableWeeklySub(5, 10)
	sub.Status = SubscriptionStatusSuspended
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 5)

	require.ErrorIs(t, err, ErrSubscriptionNotUsable)
	require.False(t, stub.casCalled)
}

func TestUserResetWeeklyLimit_CASZeroRows(t *testing.T) {
	sub := usableWeeklySub(6, 10)
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 0}
	svc := newUserResetWeeklyLimitSvc(stub)

	_, err := svc.UserResetWeeklyLimit(context.Background(), 10, 6)

	require.ErrorIs(t, err, ErrSubscriptionWeeklyLimitResetExhausted)
	require.True(t, stub.casCalled)
}

func TestUserResetWeeklyLimit_ConcurrentSecondFails(t *testing.T) {
	// Simulates: first call succeeds (CAS hits), second call sees reset_at set.
	sub := usableWeeklySub(7, 10)
	stub := &userResetWeeklyLimitRepoStub{sub: sub, casAffected: 1}
	svc := newUserResetWeeklyLimitSvc(stub)

	first, err := svc.UserResetWeeklyLimit(context.Background(), 10, 7)
	require.NoError(t, err)
	require.NotNil(t, first.WeeklyLimitUserResetAt)

	// After first success, in-memory sub has reset_at; second call must exhaust.
	_, err = svc.UserResetWeeklyLimit(context.Background(), 10, 7)
	require.ErrorIs(t, err, ErrSubscriptionWeeklyLimitResetExhausted)
}
