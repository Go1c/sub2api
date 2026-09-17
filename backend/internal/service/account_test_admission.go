package service

import (
	"context"
	"errors"
	"time"
)

type TestAdmissionWaitError struct {
	Until  time.Time
	Reason string
}

func (e *TestAdmissionWaitError) Error() string { return e.Reason }

// Account/model cooldowns apply to probes too. This never changes policy or
// clears limits; it merely delays an attempt until the latest known boundary.
func accountTestCooldown(ctx context.Context, a *Account, model string, now time.Time) error {
	if a == nil {
		return errors.New("账号不存在")
	}
	until, reason := now, ""
	for _, item := range []struct {
		deadline *time.Time
		reason   string
	}{
		{a.RateLimitResetAt, "账号限流冷却尚未结束"}, {a.OverloadUntil, "账号过载冷却尚未结束"}, {a.TempUnschedulableUntil, "账号临时暂停尚未结束"},
	} {
		if item.deadline != nil && item.deadline.After(until) {
			until, reason = *item.deadline, item.reason
		}
	}
	if wait := a.GetModelRateLimitRemainingTimeWithContext(ctx, model); wait > 0 && now.Add(wait).After(until) {
		until, reason = now.Add(wait), "所选模型限流冷却尚未结束"
	}
	if reason != "" {
		return &TestAdmissionWaitError{Until: until, Reason: reason}
	}
	return nil
}
