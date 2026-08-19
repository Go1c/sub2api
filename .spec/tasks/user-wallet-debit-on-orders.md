---
status: completed
title: 用户「我的订单」展示外部钱包扣款（精简：时间 / 金额 / 来源）
---

# 用户「我的订单」展示外部钱包扣款

## 做什么

管理员余额历史已能看到「外部钱包扣款」（#328）。用户目前在「我的订单」看不到这笔扣款。要在用户侧同一页面展示本人外部钱包扣款，信息精简，**必须能看到什么时候扣的款**。

## 已确认的产品口径

- 用户原话：「信息可以少一点，但是他能看到什么时候扣的款，我觉得就够了。」
- **不要**把扣款写成 LumioAPI 支付订单 / 订阅。`purpose` 只是账本标签，见 `.spec/knowledge/features/balance-debit-wallet.md`。
- **不要**改 `GET /payment/orders/my`，不要把 wallet 行混进支付订单表格（分页、取消、退款语义都不兼容）。
- 后端已有本人流水：`GET /api/v1/user/balance/transactions`（JWT/`uat_`，严格本人）。本卡只接前台。

## UI 定案

在 `frontend/src/views/user/UserOrdersView.vue` 现有支付订单表格**下方**加独立卡片：

- 标题：`外部钱包扣款`（i18n）
- 列（仅这三列，不要 purpose / txn_id / 扣后余额）：
  1. **扣款时间** `created_at`（`toLocaleString()`，与 `OrderTable` 一致）
  2. **来源** `client_name`（如 `CCHaven Control`）
  3. **金额** 展示为负值 `-$19.90`（接口 `amount` 为正；扣款用红色/danger，数字用 `.ui-mono`）
- 独立分页，默认 page_size=20，与订单区分开。
- 顶部「刷新」同时刷新订单和扣款。
- 空列表：显示 empty 文案，不把扣款行插入订单表。
- 无取消 / 退款按钮。

## API

- 在用户 API（建议 `frontend/src/api/user.ts`，`userAPI` 导出）增加 `getMyWalletTransactions({ page, page_size })`
- 路径：`GET /user/balance/transactions`（走现有 `apiClient`，它会 unwrap `{code,data}`）
- 类型字段对齐后端 `walletTransactionResponse`：`txn_id, client_id, client_name, amount, balance_after, currency, purpose, ref, created_at`
- 页面只用 `created_at / client_name / amount`；其余字段可留在类型里但不展示

## i18n

命名空间建议 `payment.walletDebits`（与 `payment.orders` 并列），三语 + zh overlay 对齐：

- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`
- `frontend/src/i18n/locales/zh-Hant.ts`
- `frontend/src/i18n/locales/zh/misc.ts`（zh overlay 含 `payment`，新 key 必须写进去，避免被 overlay 漏掉）

键：

- `title`: 外部钱包扣款 / External wallet debit / 外部錢包扣款（可复用 `redeem.walletDebit` 语义，但独立 key 以免耦合兑换页）
- `empty`: 暂无外部钱包扣款
- `debitedAt`: 扣款时间
- `source`: 来源
- `amount`: 金额

## 测试

- `frontend/src/api/__tests__/user.spec.ts`（或新建 wallet 侧 spec）：断言 `GET /user/balance/transactions` 带 `page` / `page_size`
- `frontend/src/views/user/__tests__/UserOrdersView.spec.ts`（新建，参考 `SubscriptionsView.spec.ts`）：
  - mount 后同时请求 `getMyOrders` 与 `getMyWalletTransactions`
  - mock 一条 `{ client_name: 'CCHaven Control', amount: 19.9, created_at: '2026-08-19T07:49:08Z' }`
  - 页面出现来源文案、负金额、以及格式化后的时间（不要出现 purpose / txn_id / 扣后余额）
  - 扣款行不进入 `OrderTable` 的 `orders` prop
  - 刷新按钮会再次请求扣款接口
- 相关 vitest 绿；`pnpm typecheck` 无新增类型错误

## 知识沉淀

更新 `.spec/knowledge/features/balance-debit-wallet.md`：用户可在「我的订单」看到本人外部钱包扣款（时间 / 来源 / 金额），不创建支付订单。不改 `knowledge/README.md` 文件名（已有该文档）。

## 涉及范围

- `frontend/src/api/user.ts`（及 `frontend/src/api/index.ts` 若需 re-export 类型）
- `frontend/src/views/user/UserOrdersView.vue`
- i18n 上述 4 个 locale 文件
- 测试文件
- `.spec/knowledge/features/balance-debit-wallet.md`

## 不做

- 不改 backend / 不写 payment_orders
- 不把 wallet_debit 混进管理员之外的订单状态筛选
- 不在当前 `feat/openai-luna-account-opt-in` 分支上改；从 `origin/dev` 拉新分支 `feat/user-wallet-debit-orders`
- 不提交 git（实现 + 自测即可）

## 验收标准

- [ ] 「我的订单」下方可见「外部钱包扣款」独立列表
- [ ] 每行至少展示扣款时间、来源、负金额；不展示 purpose / txn_id / 扣后余额
- [ ] 数据来自 `GET /user/balance/transactions`，不是 `/payment/orders/my`
- [ ] 支付订单表行为不变（筛选、取消、退款）
- [ ] zh / en / zh-Hant + zh/misc overlay 键齐全
- [ ] 相关 vitest 通过；`cd frontend && pnpm typecheck` 通过
- [ ] knowledge 文档已补用户可见性说明

## 依赖

无。后端接口已在 `feat(wallet)` / #328 合入。
