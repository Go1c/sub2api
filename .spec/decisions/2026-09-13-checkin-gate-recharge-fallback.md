---
name: checkin-gate-recharge-fallback
description: 签到 min_spend 取用量实际扣除与 users.total_recharged 的较高值
---

# 签到门槛在用量明细被清理时回退累计充值

- 状态: 生效

## 背景

`min_spend` 最初只 `SUM(usage_logs.actual_cost)`。`pg-log-retention` 会删掉旧用量明细，老付费用户的「累计消费」被截成近几周的实际扣除，门槛误拦真实充值用户。精确 `usage_logs` 不回填。

## 决策

门槛金额改为 `GREATEST(SUM(usage_logs.actual_cost), users.total_recharged)`。签到奖励不走 `UpdateBalance`，不进入 `total_recharged`，不能靠签到奖自我滚过门槛。

## 替代方案（未采用）

- **只改用累计充值**：纯订阅用量、零充值但消费高的用户会被误拦。
- **从备份回填 usage_logs**：运维成本高，且与「丢失明细不回填」的事故约定冲突。
