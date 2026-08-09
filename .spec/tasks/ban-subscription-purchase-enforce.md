---
status: completed
title: 禁止购买订阅 — CreateOrder 硬拦截
depends_on:
  - ban-subscription-purchase-schema
---

# 禁止购买订阅 — 购买硬拦截

## 做什么

用户侧创建订阅订单时，若该用户 `subscription_purchase_disabled == true`，返回 Forbidden，错误码稳定，前端可映射为「无权限购买」。

## 设计定案

### 拦截点

- 主路径：`PaymentService.CreateOrder` → `validateSubOrder`（`payment_order.go`）
- 覆盖：`order_type=subscription` 的**所有**支付方式（外部支付 + 余额支付 `createSubscriptionBalanceOrder`）
  - 因余额支付在 `CreateOrder` 内先 `validateOrderInput` → `validateSubOrder`，在 `validateSubOrder` 开头按 user 拦截即可一处覆盖
- **必须**在校验套餐可用性之前或之中加载 user 并检查；若 `validateSubOrder` 当前无 user，可：
  1. 在 `CreateOrder` 取 user 后、调用 `validateSubOrder` 前检查；或
  2. 扩展 `validateSubOrder` 接收 user / userID 再查库

推荐：在 `CreateOrder` 已 `GetByID` 用户之后、分支创建订单之前做检查（user 已加载）；同时在 `validateSubOrder` 内也可查一次以保证其他入口不漏。优先**最少改动且无漏网**。

### 错误

```go
infraerrors.Forbidden("SUBSCRIPTION_PURCHASE_FORBIDDEN", "subscription purchase is not allowed for this user")
```

- HTTP 403
- reason 固定：`SUBSCRIPTION_PURCHASE_FORBIDDEN`

### 不拦截

- 余额充值（`OrderTypeBalance`）
- 管理员后台分配/撤销订阅
- 兑换码兑换订阅
- 已有订阅消费 / 网关扣费

## 涉及范围

- `backend/internal/service/payment_order.go`
- 相关 payment 单测（新建或扩展）

## 验收标准

- [ ] `subscription_purchase_disabled=true` 的用户创建订阅订单 → 403 + `SUBSCRIPTION_PURCHASE_FORBIDDEN`
- [ ] 同一用户余额充值订单仍可创建
- [ ] `subscription_purchase_disabled=false` 时订阅购买行为与改前一致
- [ ] 余额支付订阅与外部支付订阅均被拦截
- [ ] 有 unit 测试覆盖拦截与放行

## 依赖

- `ban-subscription-purchase-schema`（字段可读）
- 可与 `ban-subscription-purchase-admin-api` 并行（拦截只依赖字段与 user 读取）
