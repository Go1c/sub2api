---
name: account-error-history
description: 账号错误历史——多来源 best-effort 记录账号错误、管理后台懒加载查看；写入异步非阻塞、带去重/节流/裁剪
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 账号错误历史

后端记录每个账号来自不同来源的错误（网关真实请求 / 账号测试 / token 刷新 / 调度 / 管理员手动置错），供管理后台「更多 > 错误历史」弹窗按需排查。现有 `accounts.error_message` 是单值覆盖、无历史，本功能补上时间线。

## 背景 / 目标

- 这是**非关键、非强需求**的排查辅助：偶尔查问题用，丢几条记录完全可接受。
- 因此最高约束：写入**绝不能阻塞或拖慢主请求 / 网关热路径**；读取**懒加载**，只在管理员点开弹窗时查。

## 设计

- **实现面（写入 best-effort）**：`service.AccountErrorHistoryService.RecordAccountError(ctx, event)` 内部 `context.WithoutCancel(ctx)` + 5s 超时 + `go func`，错误只 `slog.Warn` 不上抛。调用方只管传上下文。
- **实现面（防刷爆 DB 三件套，repo 层原生 SQL）**：
  1. 60s 窗口内同 `(account_id, source, upstream_status_code, fingerprint)` 折叠为一行、`dup_count+1`、刷新 `created_at`（`upstream_status_code` 可空，用 `IS NOT DISTINCT FROM` 处理 NULL）；
  2. 每账号最小写入间隔节流（默认 2s，非折叠的新错误在间隔内丢弃）；
  3. 每账号裁剪保留最近 50 条（插入后删更早的）。
- **实现面（指纹）**：`computeAccountErrorFingerprint` 对 message 归一化（小写、压缩空白、抹掉数字 / 时间戳 / UUID / 十六进制等易变片段）后取 sha256 前 16 hex，让「同类错误、仅数字/时间不同」折叠到同一指纹。
- **实现面（埋点 / 字段可空性矩阵）**：只有 **gateway 源**（`gateway_service.go` 的 `handleErrorResponse` 收口点）字段齐全——`user_id`/`user_email`（从 gin context 的 `api_key` 读 `apiKey.User` 快照）/`model`（从 `ctxkey.Model`）/`upstream_status_code`。其它来源（test/refresh/schedule/admin）只填 `account_id`+`message`+`source`，其余 NULL。
- **设计面（避免重复记录）**：ratelimit 内部各 handler（handleAuthError/handleCustomErrorCode）由 `handleErrorResponse → HandleUpstreamError` 调用，已被 gateway 源覆盖，故不在 ratelimit 内部另记，避免一条错误记两遍。
- **交互面（接口）**：`GET /api/v1/admin/accounts/:id/error-history?limit=20`（默认 20、上限 50 钳制），挂在 admin 鉴权组、与 `/:id/stats` 同组。响应外层形状为 **`{ "items": [...] }`**（便于以后加分页）；每条 item（snake_case）：`id`(int64)、`created_at`(RFC3339 UTC)、`user_email`(string|null)、`model`(string|null)、`upstream_status_code`(int|null)、`message`(string)、`source`(string)、`dup_count`(int)。
- **交互面（前端）**：账号管理页每行「更多」菜单加「错误历史」项（红色 `exclamationTriangle` 图标），点开 `AccountErrorHistoryModal` 才请求数据（**懒加载**，不在列表渲染 / onMounted 预取），关闭清空。弹窗用 `DataTable` 列出：时间（相对时间 + title 全文）/ 用户邮箱 / 模型 / 状态码 / 来源（小徽章）/ 错误信息（truncate + title）/ 次数（`dup_count>1` 显示 ×N）。空字段显示「—」，无记录给空态文案。API 封装 `accountsAPI.getErrorHistory(id, limit=20)`，响应容错裸数组与 `{items}` 两种。

## 已决策

- **setter 注入而非改构造签名**：错误历史服务通过 `SetAccountErrorHistoryService` 注入进 gateway / openai gateway / account test / token refresh / admin 五个 service——避免改 `NewGatewayService` 等构造签名波及大量已有测试。`AccountHandler` 仍按 planner 要求走构造参数（只波及 `api_contract_test.go` 与几处 handler 测试的 stub）。
- **本组只接 gateway+test+refresh+schedule+admin 五类来源**：ratelimit / antigravity penalty 暂不单独接（gateway 路径已覆盖或价值低、控噪声）。表有去重+节流+裁剪兜底，后续全接也安全。
- **建表走编号 SQL 迁移 `152_add_account_error_histories.sql`**：Ent schema 仅做 ORM，列类型与 SQL 严格一致（本仓库不靠 Ent auto-migrate 建表）。

## 待解决

- 是否接入 ratelimit(1807 stream timeout) / antigravity penalty 来源——按需再加。
- 端到端实机验证（造错误 → 看折叠/裁剪/弹窗展示）尚未在真实 DB 上跑过，单测与集成测试逻辑已覆盖。

## 相关

- 迁移：`backend/migrations/152_add_account_error_histories.sql`
- schema：`backend/ent/schema/account_error_history.go`（+ `account.go` 反向 edge）
- repo：`backend/internal/repository/account_error_history_repo.go`
- service：`backend/internal/service/account_error_history.go`（类型/接口）、`account_error_history_service.go`（指纹/异步）
- 埋点：`gateway_service.go`（recordGatewayAccountError）、`account_test_service.go`、`token_refresh_service.go`、`openai_account_scheduler.go`、`admin_service.go`
- handler/route：`internal/handler/admin/account_handler.go`(ErrorHistory)、`internal/server/routes/admin.go`
- DI：`internal/repository/wire.go`、`internal/service/wire.go`、`cmd/server/wire_gen.go`（wire CLI 在本仓库无法重生成，wire_gen 为手维护）
- 前端：`frontend/src/components/admin/account/AccountErrorHistoryModal.vue`（新建弹窗）、`AccountActionMenu.vue`（菜单项）、`views/admin/AccountsView.vue`（接线）、`api/admin/accounts.ts`（`getErrorHistory`）、`i18n/locales/{zh,en,zh-Hant}.ts`（`admin.accounts.errorHistory.*`）
