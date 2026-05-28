package service

import "math"

// SubscriptionCreditWindowState 窗口节流状态：
//   - LimitUSD: 窗口上限（nil 或 <= 0 表示该窗口无限制）
//   - UsedUSD : 窗口当前已用量（调用前需保证已对齐"是否重置"——重置后应传 0）
type SubscriptionCreditWindowState struct {
	LimitUSD *float64
	UsedUSD  float64
}

// SubscriptionCreditAllocationInput 拆分计算的输入。
//
// 调用者职责：
//   - QuotaLimitUSD / QuotaUsedUSD 来自 SELECT FOR UPDATE 后的订阅快照
//   - Daily / Weekly 的 UsedUSD 已经处理过"窗口是否重置"
//     （重置 → 传 0；未重置 → 传当前 used）
//   - ActualCostUSD 是上游计费完成后的真实费用
type SubscriptionCreditAllocationInput struct {
	ActualCostUSD float64
	QuotaLimitUSD float64
	QuotaUsedUSD  float64
	Daily         SubscriptionCreditWindowState
	Weekly        SubscriptionCreditWindowState
}

// SubscriptionCreditAllocation 拆分计算的结果：
//
//	SubscriptionCostUSD + BalanceCostUSD == max(ActualCostUSD, 0)
//
// 注意：不返回"是否触顶"——触顶判定由扣费 SQL 的 RETURNING 跨越 bool 单源决定，
// 避免 Go 内存推算与 DB 状态的双源不一致。
type SubscriptionCreditAllocation struct {
	SubscriptionCostUSD float64
	BalanceCostUSD      float64
}

// AllocateSubscriptionCredit 计算单次请求的订阅扣费 / 余额扣费拆分。
//
// 拆分规则：
//  1. 实际费用为负数时按 0 处理（防御）
//  2. 订阅可用额度 = min(总剩余, 日窗口剩余, 周窗口剩余)，任何维度 <= 0 都会让可用为 0
//  3. SubscriptionCostUSD = min(实际费用, 可用额度)
//  4. BalanceCostUSD = 实际费用 - SubscriptionCostUSD
//
// LimitUSD 为 nil 或 <= 0 视为该窗口无限制（不参与最小化）。
func AllocateSubscriptionCredit(in SubscriptionCreditAllocationInput) SubscriptionCreditAllocation {
	cost := math.Max(in.ActualCostUSD, 0)
	available := math.Max(in.QuotaLimitUSD-in.QuotaUsedUSD, 0)
	available = minByWindow(available, in.Daily)
	available = minByWindow(available, in.Weekly)

	subCost := math.Min(cost, available)
	return SubscriptionCreditAllocation{
		SubscriptionCostUSD: subCost,
		BalanceCostUSD:      math.Max(cost-subCost, 0),
	}
}

// minByWindow 把窗口剩余作为可用额度的上限。
// 窗口无限制（LimitUSD nil 或 <= 0）时直接返回 current 不收紧。
func minByWindow(current float64, w SubscriptionCreditWindowState) float64 {
	if w.LimitUSD == nil || *w.LimitUSD <= 0 {
		return current
	}
	remaining := *w.LimitUSD - w.UsedUSD
	if remaining < 0 {
		remaining = 0
	}
	if remaining < current {
		return remaining
	}
	return current
}
