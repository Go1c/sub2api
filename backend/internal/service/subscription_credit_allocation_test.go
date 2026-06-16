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

// quotaCrossedExhaustion 复刻扣费 SQL 中 exhausted_at CASE 的跨越判定：
//
//	quota_used + subCost >= quota_limit - epsilon
//
// epsilon=0 即修复前的严格判定；epsilon=SubscriptionQuotaExhaustionEpsilonUSD 即修复后。
// 用纯 Go 函数镜像 SQL 谓词，验证容差逻辑（真实精度丢失发生在 Postgres DECIMAL(20,10)
// 的舍入层，无法在纯 Go float64 减法里复现，故这里直接喂入线上观测到的临界值）。
func quotaCrossedExhaustion(quotaUsed, subCost, quotaLimit, epsilon float64) bool {
	return quotaLimit > 0 &&
		quotaUsed < quotaLimit &&
		quotaUsed+subCost >= quotaLimit-epsilon
}

// TestSubscriptionQuotaExhaustionEpsilon 锁死「用满到 limit 的精度边界」回归：
// 当 quota_used + subCost 因浮点/DECIMAL 舍入误差略小于 limit 时，
// 修复前（epsilon=0）漏判耗尽，修复后（epsilon=1e-6）正确判定耗尽。
func TestSubscriptionQuotaExhaustionEpsilon(t *testing.T) {
	const limit = 93.0

	// 线上 user_id=306 / sub_id=28 的场景：quota_used 落库被舍入成 93.0000000000
	// （remaining=0），但未舍入的跨越判定看到的是「差一丝没到 limit」。这里用
	// 紧贴 limit 之下的最大 float64 表示这个「差一丝」（gap ~1e-14，落在 1e-6 容差内），
	// 复刻精度边界。
	nearMiss := math.Nextafter(limit, 0)

	// 防御性自检：这个值确实「严格小于 limit、但落在 1e-6 容差内」，
	// 否则用例就没在测真正的边界。
	if !(nearMiss < limit) {
		t.Fatalf("nearMiss %.20g should be strictly below limit %.20g", nearMiss, limit)
	}
	if limit-nearMiss > SubscriptionQuotaExhaustionEpsilonUSD {
		t.Fatalf("gap %.3g must be within epsilon %.3g", limit-nearMiss, SubscriptionQuotaExhaustionEpsilonUSD)
	}

	t.Run("strict predicate misses exhaustion (reproduces the bug)", func(t *testing.T) {
		if quotaCrossedExhaustion(nearMiss, 0, limit, 0) {
			t.Fatalf("strict predicate unexpectedly crossed; bug no longer reproducible")
		}
	})

	t.Run("epsilon predicate catches exhaustion (the fix)", func(t *testing.T) {
		if !quotaCrossedExhaustion(nearMiss, 0, limit, SubscriptionQuotaExhaustionEpsilonUSD) {
			t.Fatalf("epsilon predicate must mark subscription exhausted at limit boundary")
		}
	})

	// 端到端：用 AllocateSubscriptionCredit 算出「正好用满到 limit」的 subCost，
	// 再让 pre+subCost 落在临界值上，证明修复在真实分配链路下生效。
	t.Run("allocation fills to limit then epsilon predicate crosses", func(t *testing.T) {
		pre := nearMiss // 已累积到临界值的 quota_used
		alloc := AllocateSubscriptionCredit(SubscriptionCreditAllocationInput{
			ActualCostUSD: 5, // 远超剩余，必被封顶到剩余额度
			QuotaLimitUSD: limit,
			QuotaUsedUSD:  pre,
		})
		// 剩余额度极小（<1e-6），subCost 被封顶到约 limit-pre，pre+subCost 仍可能因
		// DECIMAL 舍入略小于 limit；严格判定漏盖、容差判定正确盖戳。
		if quotaCrossedExhaustion(pre, alloc.SubscriptionCostUSD, limit, 0) {
			// 若 Go 侧加法恰好达标，严格判定也会过；此时不构成 bug 场景，跳过断言。
			t.Skipf("go-side sum reached limit exactly (subCost=%.17g); DB-rounding gap not reproduced in pure Go", alloc.SubscriptionCostUSD)
		}
		if !quotaCrossedExhaustion(pre, alloc.SubscriptionCostUSD, limit, SubscriptionQuotaExhaustionEpsilonUSD) {
			t.Fatalf("epsilon predicate must cross after allocation fills quota (subCost=%.17g)", alloc.SubscriptionCostUSD)
		}
	})
}
