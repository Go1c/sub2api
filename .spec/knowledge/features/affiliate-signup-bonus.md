---
name: affiliate-signup-bonus
description: 把邀请注册赠送记录纳入与返利余额转账相同的管理/用户记录框架，并提供独立的管理员侧记录页。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 邀请注册赠送记录（Affiliate Signup Bonus History）
简介：让邀请注册赠送（signup bonus）在与"邀请返利余额转账"相同的管理/用户记录框架中可见。把它当作余额历史中的一类条目展示，并把 `affiliate_invite_logs` 提升为一级管理员侧栏页面。

> 注：本特性只有 plan，没有独立 design spec。以下设计内容来自 plan 推导。

## 背景 / 目标
邀请激励统一记录在 `user_affiliate_ledger`，而不是发放真实兑换码（redeem code）。把 `action = 'signup_bonus'` 当作一种 affiliate 余额历史条目，在用户余额对话框中显示，并把邀请记录暴露为独立的管理员侧栏页面。

## 设计

### 后端余额历史
- 更新 `listAffiliateBalanceHistory` 与 `countAffiliateBalanceHistory`，纳入 `action IN ('transfer', 'signup_bonus')`。
- 两种 action 都映射到 `RedeemTypeAffiliateBalance`（用"合成兑换码"的形状表达，不发真实码）。
- 用 ledger 的 action 填充 `Notes`，让前端据此选择正确标题。
- 文件：`backend/internal/service/admin_service.go`、测试 `backend/internal/service/admin_balance_history_test.go`。

### 管理员注册赠送记录页
- 在 `frontend/src/api/admin/affiliates.ts` 加 `listSignupBonusRecords()`，落在 `/admin/affiliates/invite-logs` 附近。
- 新建路由 `/admin/affiliates/signup-bonuses` 与侧栏项（归在"邀请返利"下）。页面 `frontend/src/views/admin/affiliates/AdminAffiliateSignupBonusesView.vue`。
- 复用 `AdminAffiliateRecordsTable.vue`，扩展 `signup-bonuses` 类型，从 `AffiliateInviteLog` 渲染邀请人、被邀请人、结果、赠送金额、指纹、IP、创建时间。
- 从 Settings 移除内嵌的邀请记录表，避免从 Settings 加载邀请记录（邀请返利设置与自定义用户配置仍留在 Settings）。

### 用户余额历史文案
- `frontend/src/components/admin/user/UserBalanceHistoryModal.vue`：当 `item.type === 'affiliate_balance'` 时，`item.notes === 'signup_bonus'` 显示"余额充值（邀请注册赠送）"，否则保持"余额充值（返利转入）"。
- 过滤值仍保留 `affiliate_balance`，保持现有 API 契约稳定。

## 已决策
- affiliate 激励统一存 `user_affiliate_ledger`，不发真实兑换码（合成兑换码形状仅用于复用既有余额历史展示）。
- 通过 ledger `action`（`transfer` vs `signup_bonus`）区分两类条目，并经 `Notes` 透传给前端选标题。
- 过滤值不改（仍 `affiliate_balance`），保证 API 契约稳定。
- Settings 中去重：移除内嵌邀请记录表，邀请记录改由专属侧栏页承载。
- 用户侧 `GET /user/aff` 的 `rules.signup_bonus_enabled` / `rules.signup_bonus_amount` 透传当前配置，供门户展示「邀请注册得 X 元」；发放与记录逻辑不变。

## 实现
- Task 1 后端：先写失败测试（区分 `transfer`/`signup_bonus`），改 `listAffiliateBalanceHistory`/`countAffiliateBalanceHistory`，跑 `go test ./backend/internal/service -run 'TestMergeBalanceHistoryCodes|TestAffiliateBalanceHistoryItem'`。
- Task 2 前端：加 API、路由、侧栏项；扩展记录表 `signup-bonuses` 类型；从 Settings 移除重复记录表。
- Task 3 用户文案：按 ledger action 拆分标题；过滤值保持兼容。
- Task 4 验证与发布：后端 `go test ./backend/internal/service ./backend/internal/repository ./backend/internal/handler/admin ./backend/internal/server`；前端 `pnpm --dir frontend typecheck` + `build`；commit 到 `dev` → push → `publish` 合并 `dev` → push。

## 相关
- [[subscription-credit-pool]]
- [[payment]]
- 后端：`backend/internal/service/admin_service.go`、`backend/internal/service/admin_balance_history_test.go`、数据表 `user_affiliate_ledger`、`affiliate_invite_logs`
- 前端：`frontend/src/api/admin/affiliates.ts`、`frontend/src/views/admin/affiliates/AdminAffiliateSignupBonusesView.vue`、`frontend/src/views/admin/affiliates/AdminAffiliateRecordsTable.vue`、`frontend/src/components/admin/user/UserBalanceHistoryModal.vue`、`frontend/src/router/index.ts`、`frontend/src/components/layout/AppSidebar.vue`
- 技术栈：Go service/repository、Gin admin 路由、Vue 3 admin UI、vue-i18n
