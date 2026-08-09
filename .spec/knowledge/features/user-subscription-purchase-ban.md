---
name: user-subscription-purchase-ban
description: 管理员可按用户禁止购买订阅——字段、管理端开关、CreateOrder 硬拦截与「无权限购买」提示
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 禁止用户购买订阅

简介：管理员在用户编辑中可开启「禁止购买订阅」。被禁用户创建订阅订单时后端返回 403，前端提示「无权限购买」。

## 背景 / 目标

- 需要按用户粒度限制购买订阅（风控 / 纠纷 / 特殊账号），而不影响余额充值与已有订阅使用。
- 与 `invoice_enabled` 同级的用户权限开关模式。

## 设计

### 数据

- 表字段：`users.subscription_purchase_disabled BOOLEAN NOT NULL DEFAULT FALSE`
- Migration：`backend/migrations/921_add_user_subscription_purchase_disabled.sql`
- Ent：`backend/ent/schema/user.go`
- 读写：`user_repo` 通过 SQL hydrate/set（与 `invoice_enabled` 相同，不依赖 ent 全量列映射路径）

### 管理端

- Update API：`subscription_purchase_disabled` 指针字段，未传不修改
- UI：`UserEditModal` 开关（`data-test="subscription-purchase-disabled-toggle"`）

### 拦截

- 点：`PaymentService.CreateOrder`，在用户状态校验后、创建订单前
- 条件：`order_type=subscription` 且 `user.SubscriptionPurchaseDisabled`
- 错误：`Forbidden` / reason `SUBSCRIPTION_PURCHASE_FORBIDDEN`
- 覆盖：外部支付 + 余额支付订阅（同一入口）
- **不拦截**：余额充值、管理员代开、兑换码、已有订阅消费

### 用户提示

- i18n：`payment.errors.SUBSCRIPTION_PURCHASE_FORBIDDEN`
  - zh：无权限购买
  - en：No permission to purchase subscriptions

## 已决策

- 字段默认 `false`，存量用户行为不变。
- 仅支付购买硬拦；兑换码不在本期范围。
- 前端以错误码映射为主，不强制提前隐藏订阅 tab。

## 相关

- 任务卡：`.spec/tasks/ban-subscription-purchase-*.md`
- 代码：`payment_order.go`、`UserEditModal.vue`、`user_repo.go`
