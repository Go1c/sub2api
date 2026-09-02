---
name: daily-checkin
description: 独立每日签到模块：原子发奖、连续周期、全站每日预算，以及用户和管理员界面
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 每日签到

每日签到是一套与邀请返利和公共 Settings DTO 解耦的奖励模块。它为活跃用户按运营时区每日发放一次随机奖励，可叠加循环里程碑奖金，并在全站每日预算不足时保留签到连续性但不发放余额。实际发奖通过 `redeem_codes` 记入通用余额流水。

## 目标与边界

- 签到业务集中在 `backend/internal/checkin/` 和 `frontend/src/features/checkin/`，共享模块只负责依赖注入、路由、导航与页面挂载。
- 实际发奖（`actual_reward > 0`）在同一签到 SQL 事务内插入已使用的 `redeem_codes`。不调用 `RedeemService.Redeem`，也不写入 `AffiliateService` 或 `user_affiliate_ledger`。奖池耗尽、0 元发奖或同日重放不写兑换码。金额拆开、禁止双计：
  - `users.balance` 与 `daily_checkin_daily_counters.awarded_total` 仍一次加上整笔 `actual`（`actual` = 基础 + 里程碑；预算不足则整笔 0，不做部分发放）。`daily_checkin_records` 的 `base_reward` / `milestone_bonus` / `actual_reward` 列语义不变。
  - 基础部分（`actual - milestone_bonus`，仅 >0 时）：`type=checkin_balance`，`notes=daily_checkin:<recordID>`。用户/管理端标题为「余额充值（签到）」。无里程碑的日子只写这一条，value=`actual`。
  - 里程碑奖金（仅 `milestone_bonus > 0` 且 `actual > 0` 时）：`type=checkin_milestone`，`notes=daily_checkin_milestone:<recordID>:<day>`（day = `milestone_day`）。用户兑换历史 DTO 对 `checkin_milestone` 放行 notes（`checkin_balance` 仍不放行），前端用 `^daily_checkin_milestone:\d+:(\d+)$` 解析 day，标题为「签到里程碑{day}天」；管理端筛选「余额（签到里程碑）」。
  - `SumPositiveBalanceByUser` 把 `checkin_balance` 和 `checkin_milestone` 都计入总充值。
  - 不做历史回填：拆分只影响此后命中的里程碑；旧流水里里程碑已含在当日 `checkin_balance`。
- 配置由独立管理 API 维护，不加入现有 Settings 请求或解析链。
- 数据表由独立 SQL migration 创建，Repository 使用 `database/sql`，不修改 Ent Schema。

## 数据与事务

迁移 `backend/migrations/929_daily_checkin.sql` 创建三张表：

- `daily_checkin_settings`：单行配置，默认关闭；保存奖励区间、IANA 时区、每日上限和最多 10 条里程碑。
- `daily_checkin_records`：不可变签到流水，记录用户快照、业务日期、连续/循环天数、奖励拆分、实际发放、余额快照和客户端信息。用户删除后 `user_id` 置空，审计快照保留。
- `daily_checkin_daily_counters`：按业务日期累计实际发放额，事务内锁行以串行化每日预算分配。

签到事务锁定配置和用户，检查同一业务日的已有流水，再锁定当日预算行。连续天数根据上一条业务日期计算；随机基础奖励使用密码学安全随机源并按万分之一美元取值，最高里程碑天数构成循环周期。余额增加、流水写入、预算累计，以及发奖时写入已使用兑换码（无里程碑一条 `checkin_balance`，命中里程碑时再加一条 `checkin_milestone`），都在同一事务中完成，任一步失败全部回滚。

每日上限为 `0` 时不限额。剩余预算不足以完整支付本次奖励时，不做部分发放：流水状态记为 `budget_exhausted`、实际奖励为 `0`，但该日仍计入连续签到。同日重复请求返回原流水，不重复发奖。事务提交后以 best-effort 方式失效余额和认证缓存，失效失败只记录日志。

## 接口与界面

用户接口：

- `GET /api/v1/user/checkin`：功能开关、今日状态、累计次数与奖励、连续/循环天数、下一里程碑、余额和最近 20 条流水。
- `POST /api/v1/user/checkin`：执行或重放当日签到，返回奖励拆分、状态、余额快照和 `already_checked_in`。

管理员接口：

- `GET /api/v1/admin/affiliates/checkins`：按用户、邮箱/用户名、业务日期和状态筛选，并支持排序与分页。
- `GET /api/v1/admin/affiliates/checkins/stats`：按运营时区汇总今日 / 本周（周一起） / 本月 / 累计的参与用户数、签到次数，以及已发放 `actual_reward` 的总额、平均、P50、P90、最大值。可叠加 search / user_id / status，不使用列表的单日 `business_date`。
- `GET /api/v1/admin/affiliates/checkins/settings`：读取独立配置。
- `PUT /api/v1/admin/affiliates/checkins/settings`：校验、规范化并读回独立配置。

用户页位于 `/checkin`，包含签到操作、汇总数据、下一里程碑和最近流水；用户导航与侧栏快捷卡仅在功能开启时显示。管理员流水位于 `/admin/affiliates/checkins`，即使功能关闭也保留入口。顶栏可切换今日 / 本周 / 本月 / 累计统计：参与用户与签到次数计入全部流水；发放总额与平均 / P50 / P90 / 最大仅统计 `awarded` 的 `actual_reward`。功能设置页挂载独立设置卡，可配置奖励、时区、每日上限和里程碑，并提示理论最高单次奖励高于每日预算的风险。

## 配置规则

- 金额必须非负且最多 4 位小数；开启功能时最大奖励必须大于 `0`，最小值不得大于最大值。
- 时区必须是有效 IANA 时区。
- 里程碑天数必须为唯一正整数，奖金非负，最多 10 条；后端按天数排序后保存。
- 每日上限允许为 `0` 或正数。低于理论最高单次奖励时仍可保存，由管理界面显示风险提示。
- 配置和时区变更只影响后续签到，历史流水不重算。

## 验证

- 后端单元测试覆盖金额精度、随机边界、连续签到、循环里程碑、预算耗尽、重复请求、回滚、缓存失效容错，以及命中里程碑时 `checkin_balance` / `checkin_milestone` 兑换码拆分与第二条插入失败回滚。
- PostgreSQL 集成测试覆盖并发重复签到、并发预算限制、余额原子增加和事务回滚；本地无 Docker 时测试按既有 TestMain 约定跳过。
- 前端 Vitest 覆盖 API、Store、用户页、管理员筛选、设置保存、侧栏状态和共享接线。
- `go vet -tags integration ./...`、签到包定向 lint、前端 `pnpm typecheck`、`pnpm build`、`pnpm lint:check` 已通过；桌面 1440x900 与移动 375x811 已完成浏览器验收。

## 相关

- 后端：`backend/internal/checkin/`、`backend/internal/server/checkin.go`
- 前端：`frontend/src/features/checkin/`
- 数据库：`backend/migrations/929_daily_checkin.sql`
- 任务卡：`.spec/tasks/daily-checkin.md`、`.spec/tasks/checkin-redeem-dark-nudge.md`、`.spec/tasks/checkin-milestone-redeem.md`
