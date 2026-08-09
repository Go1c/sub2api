---
status: completed
title: 禁止购买订阅 — 用户购买页错误提示
depends_on:
  - ban-subscription-purchase-enforce
---

# 禁止购买订阅 — 用户侧提示

## 做什么

被禁止的用户点击购买订阅时，前端展示「无权限购买」（toast / 错误弹层，与现有 payment 错误一致）。

## 设计定案

### 错误映射

- 后端 reason：`SUBSCRIPTION_PURCHASE_FORBIDDEN`
- 中文：`无权限购买`
- 英文：`No permission to purchase subscriptions`（或等价清晰文案）

### 实现

1. 在 payment 错误 i18n 表增加该 reason（与现有 `SUBSCRIPTION_BALANCE_PAYMENT_DISABLED` 等并列）
2. `PaymentView` / `paymentStore.createOrder` 已有 `extractI18nErrorMessage` 路径则只补 key 即可
3. **可选增强**（非必须）：checkout 响应带 `subscription_purchase_disabled` 提前隐藏订阅 tab 或禁用按钮——若做，后端 checkout 也需字段；**本卡默认只做错误映射**，避免扩大范围。硬拦截以后端为准。

## 涉及范围

- `frontend/src/i18n/locales/zh.ts`（及拆分文件若 payment.errors 在子模块）
- `frontend/src/i18n/locales/en.ts`（同上）
- 必要时 `PaymentView.spec.ts` 增加错误码映射断言

## 验收标准

- [ ] 创建订阅订单返回 `SUBSCRIPTION_PURCHASE_FORBIDDEN` 时，用户看到「无权限购买」
- [ ] 英文 locale 有对应文案
- [ ] 不破坏其他支付错误映射

## 依赖

- `ban-subscription-purchase-enforce`
