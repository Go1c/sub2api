package service

import (
	"math"
	"testing"
)

// 注：float64Ptr 已在 internal/service/ops_metrics_collector.go 定义，本测试直接复用。

func TestAllocateSubscriptionCredit(t *testing.T) {
	const eps = 1e-9

	tests := []struct {
		name      string
		input     SubscriptionCreditAllocationInput
		wantSub   float64
		wantBal   float64
	}{
		{
			name: "full subscription when quota and windows sufficient",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 2.8,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(10), UsedUSD: 1},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 5},
			},
			wantSub: 2.8,
			wantBal: 0,
		},
		{
			name: "split when subscription only partially covers (quota tight)",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 2.8,
				QuotaLimitUSD: 11,
				QuotaUsedUSD:  10,
				// 总剩余 1，所以 subCost = 1，余额扣 1.8
				Daily:  SubscriptionCreditWindowState{LimitUSD: float64Ptr(10), UsedUSD: 1},
				Weekly: SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 5},
			},
			wantSub: 1.0,
			wantBal: 1.8,
		},
		{
			name: "split when daily limit tightens",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 3,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  0,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(2), UsedUSD: 0.5},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 0},
			},
			// 日窗口剩余 1.5 收紧
			wantSub: 1.5,
			wantBal: 1.5,
		},
		{
			name: "split when weekly limit tightens",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 3,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  0,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(10), UsedUSD: 0},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(2), UsedUSD: 1.2},
			},
			// 周窗口剩余 0.8 收紧
			wantSub: 0.8,
			wantBal: 2.2,
		},
		{
			name: "total exhausted: all to balance",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 2,
				QuotaLimitUSD: 10,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
			},
			wantSub: 0,
			wantBal: 2,
		},
		{
			name: "daily exhausted: all to balance even with quota left",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 1.5,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(5), UsedUSD: 5},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 5},
			},
			wantSub: 0,
			wantBal: 1.5,
		},
		{
			name: "weekly exhausted: all to balance",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 1.5,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(5), UsedUSD: 1},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(20), UsedUSD: 20},
			},
			wantSub: 0,
			wantBal: 1.5,
		},
		{
			name: "zero cost request: both zero",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 0,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(10), UsedUSD: 1},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 5},
			},
			wantSub: 0,
			wantBal: 0,
		},
		{
			name: "negative cost defensive: treated as 0",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: -1.5,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  10,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(10), UsedUSD: 1},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 5},
			},
			wantSub: 0,
			wantBal: 0,
		},
		{
			name: "quota_used > quota_limit defensive: available clamped to 0",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 2,
				QuotaLimitUSD: 10,
				QuotaUsedUSD:  15, // overdrawn
				Daily:         SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
			},
			wantSub: 0,
			wantBal: 2,
		},
		{
			name: "nil daily limit means no daily restriction",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 5,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  0,
				Daily:         SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(50), UsedUSD: 0},
			},
			wantSub: 5,
			wantBal: 0,
		},
		{
			name: "zero or negative window limit treated as no restriction",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 3,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  0,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(0), UsedUSD: 999},
				Weekly:        SubscriptionCreditWindowState{LimitUSD: float64Ptr(-1), UsedUSD: 999},
			},
			wantSub: 3,
			wantBal: 0,
		},
		{
			name: "window over-used defensive: remaining clamped to 0",
			input: SubscriptionCreditAllocationInput{
				ActualCostUSD: 2,
				QuotaLimitUSD: 100,
				QuotaUsedUSD:  0,
				Daily:         SubscriptionCreditWindowState{LimitUSD: float64Ptr(5), UsedUSD: 10}, // 异常超用
				Weekly:        SubscriptionCreditWindowState{LimitUSD: nil, UsedUSD: 0},
			},
			wantSub: 0,
			wantBal: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AllocateSubscriptionCredit(tc.input)
			if math.Abs(got.SubscriptionCostUSD-tc.wantSub) > eps {
				t.Errorf("SubscriptionCostUSD = %v, want %v", got.SubscriptionCostUSD, tc.wantSub)
			}
			if math.Abs(got.BalanceCostUSD-tc.wantBal) > eps {
				t.Errorf("BalanceCostUSD = %v, want %v", got.BalanceCostUSD, tc.wantBal)
			}
			// 一致性约束：拆分总和不应超过实际费用（防止双扣）
			totalSplit := got.SubscriptionCostUSD + got.BalanceCostUSD
			expectedTotal := math.Max(tc.input.ActualCostUSD, 0)
			if math.Abs(totalSplit-expectedTotal) > eps {
				t.Errorf("split sum = %v, expected total = %v", totalSplit, expectedTotal)
			}
		})
	}
}
