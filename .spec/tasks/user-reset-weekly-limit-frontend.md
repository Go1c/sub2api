---
status: completed
title: 用户重置周限 — SubscriptionsView UI / API / i18n / 测试
---

# 用户重置周限 — Frontend

## 做什么

在用户订阅页可用卡片标题区（与绿色「续费」并排）增加红色「重置周限」：确认弹窗说明每订阅周期一次并展示 remaining，确认后调用户 API 并刷新列表。

## UI 定案

- 位置：`frontend/src/views/user/SubscriptionsView.vue` 标题右侧 action 区；「重置周限」在左/与「续费」并排（参考产品截图：红按钮 + 绿续费）
- 显示条件：`subscription.is_usable === true && subscription.weekly_limit_usd != null`
- `weekly_limit_reset_remaining === 0`：按钮 **disabled**（不弹窗）
- remaining > 0：可点 → `ConfirmDialog`（`danger`）
  - 文案：每个订阅周期仅一次重置机会；当前剩余 **{remaining}** 次
  - 说明仅清零周限用量，不影响总额度/累计已用/日限/状态/到期
- 成功：toast + 关窗 + `loadSubscriptions()`
- 失败：用 `extractApiErrorCode` + 本地 error map 展示；fallback `extractApiErrorMessage`

## API

- `frontend/src/api/subscriptions.ts` 新增 `resetWeeklyLimit(id: number)` →  
  `POST /subscriptions/:id/reset-weekly-limit`
- `UserSubscription` 类型增加 `weekly_limit_reset_remaining?: number`（及如有 `weekly_limit_user_reset_at`）

## i18n

- `userSubscriptions` 命名空间（`zh` / `en` / `zh-Hant` 结构对齐）：
  - 按钮、标题、确认正文（含 remaining 插值）、成功/失败、错误码文案
- 错误码映射可挂 `subscriptionCreditErrorMessages` 或页面内 map：
  - `SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED`
  - `SUBSCRIPTION_NO_WEEKLY_LIMIT`
  - `SUBSCRIPTION_NOT_USABLE`
  - `SUBSCRIPTION_NOT_FOUND`（通用失败文案即可）

## 测试

- 扩展 `frontend/src/views/user/__tests__/SubscriptionsView.spec.ts`：
  - 有周限 usable 卡渲染重置按钮
  - 无周限不渲染
  - remaining=0 时 disabled
  - 确认后调用 API 且 payload/路径正确（可 mock API）
- 可选：`frontend/src/api/__tests__/subscriptions.spec.ts` 断言路径
- `localeIntegrity` 结构对齐保持绿

## 涉及范围

- `frontend/src/views/user/SubscriptionsView.vue`
- `frontend/src/api/subscriptions.ts`
- `frontend/src/types/index.ts`
- `frontend/src/i18n/locales/{zh,en,zh-Hant}.ts` 及 zh 拆分文件（若 `userSubscriptions` 在 `zh/misc.ts` 等，按现有结构改，勿双写漂移）
- `frontend/src/views/user/__tests__/SubscriptionsView.spec.ts`
- `frontend/src/components/common/ConfirmDialog.vue`（复用，不改组件本身除非缺 prop）

## 验收标准

- [ ] usable + 有周限：可见红色「重置周限」与绿色「续费」并排
- [ ] 无周限或非 usable 列表卡：无该按钮
- [ ] remaining=0：按钮 disabled
- [ ] 弹窗文案含「每个订阅周期仅一次」+ remaining
- [ ] 确认调用 `POST /subscriptions/:id/reset-weekly-limit`（非 admin 路径）
- [ ] 成功刷新列表；周用量展示归零（以后端返回为准）
- [ ] zh / en / zh-Hant 键齐全；locale integrity 通过
- [ ] 相关 vitest 通过；`pnpm typecheck` 无新增类型错误

## 依赖

- `user-reset-weekly-limit-api`（联调需要；可先 mock API 并行开发 UI，但合并前须接真接口）
