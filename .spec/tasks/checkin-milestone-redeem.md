---
status: completed
---

# 签到里程碑独立发放（兑换码流水）

## 诊断（已确认，不要另起方案）

用户截图（`admin@lumio.games`，连续 4 天，下一里程碑第 7 天 +$7）：

- 签到页「最近签到」里 **2026/08/21 第 3 天已有里程碑奖金 $3.0000**，实际奖励 $3.9325 = 基础 $0.9325 + 里程碑 $3，累计奖励 $7.82 已含这笔。
- **钱已经入账**，不是没发放。
- 当前实现把 `actual_reward`（基础 + 里程碑）写成 **一条** 已使用 `redeem_codes`：`type=checkin_balance`，`notes=daily_checkin:<recordID>`。用户兑换记录 / 管理端充值流水标题只有「余额充值（签到）」，**没有「签到里程碑x天」这一类型**，所以用户觉得「看不到奖励」。

要修的是：**发放形态 + 展示类型**，不是补发余额（会重复入账）。

## 做什么

命中里程碑且本次实际发奖 `actual > 0` 时，在同一签到 SQL 事务里再写一条已使用兑换码（对齐现有签到发放，**不要**调用 `RedeemService.Redeem`）。用户兑换历史和管理端用户充值/并发变动记录显示「签到里程碑{day}天」。基础奖励仍走原来的 `checkin_balance`。

**金额必须拆开，禁止双计：**

- `users.balance` 仍一次 `+ actual`（`actual` = 基础 + 里程碑，预算规则不变：不足则整笔 0，不做部分发放）。
- `daily_checkin_daily_counters.awarded_total` 仍累加 `actual`。
- `daily_checkin_records` 的 `base_reward` / `milestone_bonus` / `actual_reward` 列语义不变。
- `checkin_balance.value` = `actual - milestone_bonus`（当前全额发放时即基础奖励）。仅当该部分 `> 0` 时插入。
- `checkin_milestone.value` = `milestone_bonus`。仅当里程碑奖金 `> 0` 且本次 `actual > 0` 时插入。
- 无里程碑的日子：行为与现在完全一致（只有一条 `checkin_balance`，value=`actual`）。
- 奖池耗尽 / 0 元 / 同日重放：两条都不写。

## 设计约束（已锁定）

- 新类型：`checkin_milestone`（17 字符，适配 `redeem_codes.type VARCHAR(20)`）。
  - `backend/internal/service/domain_constants.go` 加 `RedeemTypeCheckinMilestone = "checkin_milestone"`（fork 常量，仿 `RedeemTypeCheckinBalance`）。
  - `backend/internal/checkin` **不要 import service**，包内自备同名字符串常量。
- `notes` 用稳定机器值：`daily_checkin_milestone:<recordID>:<day>`（day = `milestone_day`，如 3、7）。不要把中文写进 notes。
- `code` 仍 32 位 hex，`status=used`，`used_by=userID`，`used_at=NOW()`。
- `SumPositiveBalanceByUser` 的 `TypeIn` 必须加上 `checkin_milestone`（否则拆开后总充值少计里程碑）。
- 用户兑换历史 DTO（`dto.redeemCodeFromServiceBase`）目前 **只对 `admin_balance` / `admin_concurrency` 返回 notes**。要对 `checkin_milestone` 同样放行 notes，否则用户页拼不出「x天」。notes 是机器值，不是内部备注。不要对 `checkin_balance` 放行 notes。
- 前端标题：
  - 从 `notes` 解析 day（正则 `^daily_checkin_milestone:\d+:(\d+)$`）。
  - i18n：`redeem.balanceAddedCheckinMilestone` = `签到里程碑{day}天` / `Check-in Milestone Day {day}` / `簽到里程碑{day}天`。
  - 管理端筛选：`admin.users.typeCheckinMilestone` = `余额（签到里程碑）` / `Balance (Check-in Milestone)` / `餘額（簽到里程碑）`。
- 用户页 `isBalanceType` 与管理端 `isBalanceType`、类型筛选都要包含 `checkin_milestone`。
- **不做历史回填。** 8/21 那笔 $3 已经含在当日 `checkin_balance` 里；回填会改历史金额或双计。只影响此后命中的里程碑。
- 不改 Ent schema、不改签到预算/连续天数/循环规则、不改签到页表格（那里已经有「里程碑奖金」列）。
- 不 commit、不 push（主 loop 收口）。

## 涉及范围

### 后端
- `backend/internal/checkin/types.go`（常量）
- `backend/internal/checkin/repository_transaction.go`（拆开发放）
- `backend/internal/checkin/repository_test.go`、`repository_integration_test.go`
- `backend/internal/service/domain_constants.go`
- `backend/internal/repository/redeem_code_repo.go`（`SumPositiveBalanceByUser`）
- `backend/internal/handler/dto/mappers.go`（用户历史对 `checkin_milestone` 放行 notes）
- 如有 dto mapper 单测，一并补

### 前端
- `frontend/src/views/user/RedeemView.vue`
- `frontend/src/components/admin/user/UserBalanceHistoryModal.vue`
- i18n 三份单体 + modular overlay，避免 locale 完整性测试失败：
  - `frontend/src/i18n/locales/{en,zh,zh-Hant}.ts`
  - `frontend/src/i18n/locales/{en,zh}/dashboard.ts`
  - `frontend/src/i18n/locales/{en,zh}/admin/overview.ts`
- `frontend/src/i18n/__tests__/localeIntegrity.spec.ts`

### 知识
- 更新 `.spec/knowledge/features/daily-checkin.md`：命中里程碑时第二条 `checkin_milestone` 兑换码、金额拆分、展示标题、用户 DTO 放行 notes、不做历史回填。

## 验收标准

- [x] 无里程碑的发奖仍只插入 1 条 `checkin_balance`，value=`actual`，notes=`daily_checkin:<id>`。
- [x] 命中里程碑且 `actual > 0` 时同一事务插入 2 条已使用兑换码：`checkin_balance`（基础部分）+ `checkin_milestone`（`notes=daily_checkin_milestone:<id>:<day>`，value=里程碑奖金）。用户余额只增加一次 `actual`。
- [x] 奖池耗尽 / 0 元 / 重放不插入任何兑换码。
- [x] 第二条兑换码插入失败时整笔签到回滚（余额、流水、两条兑换码都为空）。
- [x] `SumPositiveBalanceByUser` 计入 `checkin_milestone`。
- [x] 用户 `GET /redeem/history` 对 `checkin_milestone` 返回 `notes`，标题为「签到里程碑{day}天」；管理端未筛选时出现该条，筛选「余额（签到里程碑）」只出这类。
- [x] localeIntegrity 覆盖新文案；中英繁三份 + modular overlay 同步。
- [x] 后端 `go test -tags=unit ./internal/checkin ./internal/repository ./internal/handler/dto -count=1` 与本次相关测试通过。
- [x] 前端相关 Vitest + `pnpm typecheck` 通过。
- [x] `daily-checkin.md` 已更新。
- [x] 不 commit、不 push。

## 建议测试（先红后绿）

后端 unit（sqlmock）：
- 现有无里程碑用例继续只期望 1 条 `checkin_balance`。
- 新增：连续天数打到配置里程碑（settings 已有 day=7 bonus=1.0000；前一天 streak=6）时期望两条 INSERT：`checkin_balance` value=基础（fixedRandom 下 min=max=0.1 → 0.1000）+ `checkin_milestone` value=1.0000 notes=`daily_checkin_milestone:8:7`。余额 UPDATE 仍是 1.1000。
- 里程碑 INSERT 失败 → rollback。

后端 integration（有 Docker 才跑）：
- 配置 milestones `[{"day":1,"bonus":"2.0000"}]`（或连续打到 day 7），发奖后 `redeem_codes` 两条类型/金额/notes 正确，`users.balance` 等于基础+里程碑，不双计。

前端 / i18n：
- localeIntegrity 增加 milestone 文案断言。
- 若 RedeemView / UserBalanceHistoryModal 有现成标题测试就补一条；没有则至少保证 `checkin_milestone` 进入 `isBalanceType` 且标题走带 `{day}` 的 key。

## 依赖

无。

## 实现记录

### 命令与结果

```
cd backend && go test -tags=unit ./internal/checkin ./internal/repository ./internal/handler/dto -count=1
# ok  github.com/Wei-Shaw/sub2api/internal/checkin     2.466s
# ok  github.com/Wei-Shaw/sub2api/internal/repository  5.076s
# ok  github.com/Wei-Shaw/sub2api/internal/handler/dto 2.265s

cd backend && go vet -tags integration ./internal/checkin ./internal/repository ./internal/handler/dto
# exit 0

cd frontend && pnpm exec vitest run src/i18n/__tests__/localeIntegrity.spec.ts src/views/user/__tests__/RedeemView.spec.ts
# Test Files  2 passed (2)
# Tests  11 passed (11)

cd frontend && pnpm typecheck
# vue-tsc --noEmit  exit 0
```

本机无 Docker，`./internal/checkin` 集成测试未跑（`TestRepositoryCheckInWritesSplitMilestoneRedeemCodes` 已写好，有 Docker 时会跑）。未 commit / push。
