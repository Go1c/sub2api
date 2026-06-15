package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 回归：总额度池耗尽（已用 >= 上限，或 exhausted_at 已置位）的订阅，
// 即使日/周窗口仍未触顶，ValidateAndCheckLimits 也必须返回可回落错误
// （ErrSubscriptionInvalid），使中间件回落到余额闸门，避免绕过余额检查、
// 把余额扣成负数。
func TestValidateAndCheckLimitsExhaustedPoolReturnsInvalid(t *testing.T) {
	svc := &SubscriptionService{}
	daily := 35.0
	weekly := 35.0

	t.Run("quota used reaches limit", func(t *testing.T) {
		sub := &UserSubscription{
			Status:         SubscriptionStatusActive,
			ExpiresAt:      time.Now().Add(time.Hour),
			DailyLimitUSD:  &daily,
			DailyUsageUSD:  33, // 窗口冻结在上限以下
			WeeklyLimitUSD: &weekly,
			WeeklyUsageUSD: 33,
			QuotaLimitUSD:  93,
			QuotaUsedUSD:   93, // 总池耗尽
		}
		_, err := svc.ValidateAndCheckLimits(sub, nil)
		require.ErrorIs(t, err, ErrSubscriptionInvalid)
	})

	t.Run("exhausted_at set", func(t *testing.T) {
		now := time.Now()
		sub := &UserSubscription{
			Status:        SubscriptionStatusActive,
			ExpiresAt:     time.Now().Add(time.Hour),
			DailyLimitUSD: &daily,
			DailyUsageUSD: 10,
			QuotaLimitUSD: 93,
			QuotaUsedUSD:  50, // 剩余尚有，但 exhausted_at 已置位
			ExhaustedAt:   &now,
		}
		_, err := svc.ValidateAndCheckLimits(sub, nil)
		require.ErrorIs(t, err, ErrSubscriptionInvalid)
	})
}
