---
status: completed
---

# T3 — 分组用量按日 rollup

## 做什么

把上游 `#5649` 的分组日桶、昨日费用与服务端时区带进 `dev`。迁移改用 fork 编号，不搬 CI / Go 版本钉。

## 涉及范围

- `backend/migrations/924_group_usage_daily_rollups.sql`（新增）
- `backend/migrations/925_group_usage_rollup_timezone.sql`（新增）
- `backend/internal/repository/custom_group_usage_rollup_repo.go`（新增）
- `backend/internal/service/custom_group_usage_rollup.go`（新增）
- `backend/internal/repository/usage_log_repo.go` / dashboard aggregation / usage cleanup
- `backend/internal/handler/admin/group_handler.go`
- `backend/internal/pkg/usagestats/usage_log_types.go`
- `backend/internal/config/config.go`（`TZ` 优先于 `TIMEZONE`）
- `frontend/src/api/admin/groups.ts`
- `frontend/src/views/admin/GroupsView.vue`
- `frontend/src/i18n/locales/{en,zh}/admin/overview.ts` 与 `zh-Hant.ts`

## 验收标准

- [x] 管理端 usage-summary 返回 `today_cost` / `yesterday_cost` / `total_cost`，按服务端时区，不再读 query `timezone`。
- [x] 分组列表展示昨日用量。
- [x] 迁移编号为 `924`/`925`，不与现有 `900+` / 双 `923_*` 冲突。
- [x] 用量清理 / 重算会失效并回填日桶。
- [x] 相关 backend 测试与 frontend typecheck / 聚焦单测通过。

## 依赖

无（可与 T1/T2 并行，但同一 topic 分支内后做，方便一起验收）。
