---
name: checkin-redeem-dark-nudge
description: 签到发奖走兑换码流水、暗色适配、未签到侧栏跳动
status: completed
---

# 每日签到优化：兑换码入账 + 暗色 + 未签到动效

基于 `dev` 上刚合入的独立签到模块（PR #342 / `feat(checkin): add independent daily rewards module`）做三处优化，不改签到规则本身。

## 做什么

1. **暗色模式**：用户签到页 `/checkin` 在 dark 下表头、今日结果条、统计卡和文字对比度要和现有 DataTable 一致。
2. **签到余额走兑换码流程**：实际发奖（`actual_reward > 0`）必须在同一签到 SQL 事务里写入一条已使用的 `redeem_codes` 流水，管理员「用户充值和并发变动记录」能看到「余额充值（签到）」。
3. **未签到吸引点击**：侧栏左下角「每日签到」在功能开启且今日未签到时加温和跳动，签到后停止。

## 设计约束（已锁定，不要另起方案）

- 不要调用 `RedeemService.Redeem`（它另开 Ent 事务，无法与签到预算/流水原子回滚）。
- 对齐发票税点 / 管理员调余额模式：同一 `database/sql` 事务里 `UPDATE users.balance` + `INSERT redeem_codes`。
- 新类型：`checkin_balance`（16 字符，适配 `redeem_codes.type VARCHAR(20)`）。
  - 常量放 `backend/internal/service/domain_constants.go`（fork 常量，仿 `RedeemTypeAffiliateBalance`）。
  - `backend/internal/checkin` **不要 import service**，包内自备同名字符串常量和本地随机码生成。
- 仅当 `actual.IsPositive()` 时写兑换码；`budget_exhausted` / 0 元 / 同日重放都不写。
- `notes` 用稳定机器值 `daily_checkin`（可带 `:<recordID>`）。
- `code` 用 32 位 hex（`GenerateRedeemCode` 同款），满足 `code VARCHAR(32)`。
- `status=used`，`used_by=userID`，`used_at=NOW()`，`value=actual`。
- `SumPositiveBalanceByUser` 把 `checkin_balance` 算进总充值。
- 未过滤的 `ListByUserPaginated` 已会带出该行，不必再合成 affiliate/promo 那种 merge 源。
- 不做历史回填。
- 不改 Ent schema。

## 涉及范围

### 后端
- `backend/internal/checkin/repository_transaction.go` 及 unit/integration 测试
- `backend/internal/checkin/repository_integration_test.go`：集成库补最小 `redeem_codes` 表；回滚用例断言无残留码
- `backend/internal/service/domain_constants.go`
- `backend/internal/repository/redeem_code_repo.go`（`SumPositiveBalanceByUser` 的 type 列表）

### 前端
- `frontend/src/features/checkin/CheckinView.vue`（暗色）
- `frontend/src/features/checkin/SidebarCheckinCard.vue`（未签到跳动）
- `frontend/src/features/checkin/__tests__/CheckinView.spec.ts`、`SidebarCheckinCard.spec.ts`
- `frontend/src/components/admin/user/UserBalanceHistoryModal.vue`：标题、`isBalanceType`、类型筛选
- `frontend/src/views/user/RedeemView.vue`：用户兑换历史标题与余额类型
- i18n（三份单体 + modular overlay，避免 locale 完整性测试失败）：
  - `frontend/src/i18n/locales/{en,zh,zh-Hant}.ts`
  - `frontend/src/i18n/locales/{en,zh}/dashboard.ts`
  - `frontend/src/i18n/locales/{en,zh}/admin/overview.ts`
  - 文案：`redeem.balanceAddedCheckin` = 余额充值（签到） / Balance Added (Check-in)
  - 筛选：`admin.users.typeCheckinBalance` = 余额（签到） / Balance (Check-in)

### 知识
- 更新 `.spec/knowledge/features/daily-checkin.md`：发奖现在同时写入 `redeem_codes`；管理端用户流水标题为签到。原「不写入通用余额历史」作废。

## 验收标准

- [ ] 暗色下签到页表头不是白底；`thead` 有 `dark:bg-dark-800`（或等价 dark token），与 `DataTable` 一致。
- [ ] 今日结果条、里程碑图标、统计数字/标题、表体分割线都有 dark: 变体；emerald/amber 条用 `dark:bg-*-950/40 dark:text-*-300`。
- [ ] 发奖成功时同一事务插入 `type=checkin_balance` 且已使用的兑换码；失败回滚后用户余额、签到流水、兑换码均为空。
- [ ] 奖池耗尽或 0 元发奖不插入兑换码。
- [ ] 管理员用户流水未筛选时出现该条，标题为「余额充值（签到）」；筛选「余额（签到）」只出这类。
- [ ] 侧栏在 `enabled && !checkedInToday` 时有跳动（尊重 `prefers-reduced-motion`）；已签到或奖池耗尽不加跳动。
- [ ] 后端 `go test -tags=unit ./internal/checkin ./internal/repository -count=1` 中与本次改动相关的测试通过。
- [ ] 前端签到相关 Vitest + `UserBalanceHistoryModal` 若有测试通过；`pnpm typecheck` 通过。
- [ ] 不 commit、不 push（主 loop 收口）。

## 建议测试（先红后绿）

后端 unit：
- 发奖 sqlmock 期望 `INSERT INTO redeem_codes`。
- 奖池耗尽不期望该 INSERT。
- 记录插入失败仍 rollback（现有用例仍成立）。

后端 integration（有 Docker 才跑）：
- 发奖后 `redeem_codes` 有 1 条 `checkin_balance` / `used` / 金额一致。
- 记录触发器失败时 `redeem_codes` 为 0。

前端：
- CheckinView 源码或 class 含 dark 表头 token。
- SidebarCheckinCard 未签到有 `data-test` 或 class 标记动画；已签到没有。

## 依赖

无。在当前 `dev` 工作区直接改。
