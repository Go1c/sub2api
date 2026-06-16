package service

import "math"

// SubscriptionQuotaExhaustionEpsilonUSD 是判定订阅总额度「是否耗尽」时使用的容差（USD）。
//
// 背景：订阅扣费额由 AllocateSubscriptionCredit 在 Go float64 里算出并被
// quota_remaining 封顶，写入 DECIMAL(20,10) 列时会被四舍五入到 10 位。于是
// 「quota_used + subCost」可能因浮点误差略小于 quota_limit（如 92.99999999999999432），
// 落库后却被舍入凑成 limit（remaining=0），而未舍入的跨越判定看到的是「差一丝没到」，
// 导致 exhausted_at 该盖未盖、续购被永久拦截。
//
// 取 1e-6 作为容差：远大于 float64 在 ~$100 量级的相对误差（~1e-11），
// 又远小于任何真实计费粒度（最小计费远高于 $1e-6），因此既能吸收浮点噪声、
// 又不会把「真的还差不少额度」的订阅误判为耗尽。repo 层的扣费跨越判定与
// 续购门槛兜底统一引用此常量，避免魔数散落。
const SubscriptionQuotaExhaustionEpsilonUSD = 1e-6

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
