---
name: checkin-gate-paid-orders
description: 签到 min_spend 的充值侧改为已完成实付（余额+订阅），不再读 users.total_recharged
---

# 签到门槛按真金白银实付，含订阅

- 状态: 生效
- 取代: [`2026-09-13-checkin-gate-recharge-fallback.md`](2026-09-13-checkin-gate-recharge-fallback.md)

## 背景

`min_spend` 曾回退到 `users.total_recharged`。该字段只在余额正向到账时增加，**买订阅不经过余额**，支付宝/微信订阅实付进不了门槛。赠送、返利、后台加款却会进去。用量明细再被清理后，真实付过订阅的用户会被误拦。

## 决策

门槛金额改为 `GREATEST(SUM(usage_logs.actual_cost), 已完成实付净额)`。

实付净额：`payment_orders` 中 `status=COMPLETED` 且 `amount>0`，`order_type=balance`，或 `order_type=subscription` 且支付方式不是余额；净额 `amount - refund_amount`。

不算：用余额买订阅、签到奖励、后台加款、邀请返利、兑换码、未完成/已退完的订单。

## 替代方案（未采用）

- **继续用 `users.total_recharged` 再手工加订阅**：字段语义已混有赠送，订阅仍要另查订单，两套口径。
- **只改用量、把已用订阅额度加回去**：订阅额度用不完或日志已删时仍对不上实付；运营口径是「用钱充的都算」，不是「额度用完才算」。
