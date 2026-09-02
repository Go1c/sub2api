---
name: user-request-monitoring
description: 管理员在运维/风控中心针对指定用户监控其未来请求，限时抓取客户端原始请求体，非阻塞网关流量。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 用户请求监控
简介：风控中心功能，允许管理员针对单个用户监控其未来请求，限时抓取客户端的原始请求体；带每分钟限速、采样、保留期，且抓取走异步非阻塞路径，绝不拖慢或失败用户的网关请求。

## 背景 / 目标
让管理员监控某个用户未来的请求并限时抓取客户端原始请求体。首个目标用户为 `qokhi246487+luwnvj68kmze5@hotmail.com`，但功能必须对任意按邮箱选中的用户可用。

现有系统行为：
- 成功请求的元数据存于 `usage_logs`（用户、API key、账号、模型、端点、token 数、成本、延迟、IP、UA），但不含请求体。
- 失败请求在开启运维监控时可存入 `ops_error_logs`（可能含 `request_body`、`request_headers`、错误体、上游错误上下文），但面向错误，不提供对成功请求的定向监控。
- 网关有环境变量级别的调试体日志（`SUB2API_DEBUG_GATEWAY_BODY`），写快照到文件，是全局的，不适合定向监控。

## 设计

### 数据模型

`ops_user_request_monitors`（监控任务）：
- `id BIGSERIAL PK`、`user_id BIGINT NOT NULL`、`target_email VARCHAR(255)`
- `status VARCHAR(20)`：`active` / `expired` / `stopped`
- `duration_seconds INT`、`max_captures_per_minute INT`、`sample_rate_percent INT`
- `retention_days INT NOT NULL DEFAULT 7`
- `created_by BIGINT`、`created_at`、`starts_at`、`ends_at TIMESTAMPTZ NOT NULL`、`stopped_at NULL`、`last_capture_at NULL`、`capture_count BIGINT DEFAULT 0`
- 索引：`(user_id, status, starts_at, ends_at)`（热路径活动监控查询）、`(created_at DESC)`（管理列表）、`(ends_at)`（过期清理）

`ops_user_request_captures`（抓取的请求体）：
- `id BIGSERIAL PK`、`monitor_id BIGINT REFERENCES ops_user_request_monitors(id) ON DELETE CASCADE`
- `user_id`、`api_key_id NULL`、`account_id NULL`、`group_id NULL`、`request_id VARCHAR(64) NULL`、`model VARCHAR(100) NULL`、`inbound_endpoint VARCHAR(256) NULL`、`method VARCHAR(16) NULL`、`content_type VARCHAR(128) NULL`
- `body TEXT NOT NULL`、`body_bytes INT NOT NULL`、`body_truncated BOOLEAN DEFAULT false`、`sample_rate_percent INT`、`capture_minute TIMESTAMPTZ`、`created_at`、`expires_at TIMESTAMPTZ NOT NULL`
- 索引：`(monitor_id, created_at DESC)`、`(user_id, created_at DESC)`、`(expires_at)`、`(request_id)`（与 `usage_logs`/`ops_error_logs` 关联）

迁移文件：`backend/migrations/136_ops_user_request_monitoring.sql`（幂等建表+索引）。

### 后端架构
新增 `OpsUserRequestMonitorService`（归属运维子系统），职责：解析用户的活动监控、每分钟限速、采样、按 256 KiB 截断请求体、异步/短超时持久化（监控绝不阻塞网关响应）、过期监控与清理旧抓取。

热路径行为：网关处理器照常读取请求体，在认证解析出用户、转发上游之前调用 `CaptureClientRequestIfEnabled`，传入 user ID / API key ID / request ID / inbound endpoint / model（已知时）/ 原始 body / content type。服务用短 TTL 内存缓存按 `user_id` 查活动监控；存在则在 Redis 可用时检查每分钟计数器；Redis 不可用则跳过抓取而不拖慢/失败用户请求；然后用 1–100 随机整数应用采样率；写入 `ops_user_request_captures`。写失败仅记日志，不影响 API 响应。

限速语义：每分钟上限按 monitor 维度，key 格式 `ops:user-request-monitor:{monitor_id}:{yyyyMMddHHmm}`，计数器 TTL 至少 2 分钟，**限速门在采样之前**。

采样语义：`100` 在过门后抓取每个候选；`50` 约一半；`1` 约 1%；活动监控不允许 `0`。

过期与清理：列表端点即使后台 worker 未更新行也会从 `ends_at` 计算过期状态；轻量 worker 周期性把 `ends_at <= now()` 的活动监控标为 expired，并删除 `expires_at <= now()` 的抓取。

### API 设计
全部在 `/api/v1/admin/ops/user-request-monitors` 下，需管理员权限：
- `POST ""` 创建监控。请求体 `{user_id, duration_seconds, max_captures_per_minute, sample_rate_percent, retention_days}`。
- `GET ""?page=&page_size=&status=&user_query=` 列表。
- `POST "/:id/stop"` 停止（`status=stopped`, `stopped_at=now()`）。
- `GET "/:id/captures?page=&page_size="` 抓取摘要（默认不含 body）。
- `GET "/:id/captures/:capture_id"` 抓取详情（含原始 body 与截断元数据）。
- `DELETE "/:id/captures/:capture_id"` 删除单条抓取。

创建校验规则：`user_id` 必须存在；`duration_seconds` 1 秒到 24 小时；`max_captures_per_minute` 1 到 120；`sample_rate_percent` 1 到 100；`retention_days` 默认 7，1 到 30；每用户仅允许一个活动监控（重复创建返回冲突）。

### 前端设计
在现有运维仪表盘加"用户请求监控"卡片（`OpsUserRequestMonitorCard.vue`），挂在 `OpsDashboard.vue` 视觉指标行之后：
1. 监控列表：目标邮箱、状态、抓取数、每分钟上限、采样率、起止时间、保留期、操作（查看抓取 / 停止 / 创建）。
2. 创建对话框：按邮箱搜用户；时长预设 5 分钟 / 30 分钟 / 1 小时 / 24 小时；每分钟上限；采样率；保留期默认 7；**警告 body 以原文未脱敏保存**。
3. 抓取列表：时间、request ID、API key、模型、端点、body 字节数、是否截断；点击进详情。
4. 抓取详情：代码查看器展示原始 body、元数据、复制、删除。

## 已决策（Confirmed Requirements）
- 范围：仅抓取监控创建之后的未来请求。
- 抓取内容：仅客户端原始请求体；排除响应体、上游响应/转发体、请求头。
- body 以原文未脱敏保存，每请求最多 256 KiB，超出标记 `body_truncated=true` 且 `body_bytes` 记原始长度。
- 限速：每 monitor 每分钟最大抓取数；采样在过限速门之后应用。
- 时长到期自动过期；抓取在监控过期后仍保留，保留期后删除，默认 7 天。
- 权限：项目当前仅 `admin` 和 `user` 角色，故初始为 admin-only；若将来加 site-owner 角色，仅需收窄路由中间件，数据模型不变。
- 工作流：从 `dev` 切特性分支，开 PR 合回 `dev`。
- 非目标：不回填历史成功请求体（未存储）；不通过全局调试日志抓所有用户；不抓请求头/授权 token；本轮不抓响应体；本轮不加 owner 角色。
- 安全：本功能有意保存原始请求体，UI 创建前必须警告；后端不保存请求头；详情端点保持 admin-only。

## 相关
- 迁移：`backend/migrations/136_ops_user_request_monitoring.sql`
- 后端服务：`backend/internal/service/ops_user_request_monitor_service.go`、`backend/internal/service/ops_port.go`
- 仓储：`backend/internal/repository/ops_user_request_monitor_repo.go`
- 处理器：`backend/internal/handler/admin/ops_user_request_monitor_handler.go`、路由 `backend/internal/server/routes/admin.go`
- 网关钩子：`backend/internal/handler/gateway_handler.go`、`gateway_handler_chat_completions.go`、`gateway_handler_responses.go`、`gemini_v1beta_handler.go`、`openai_gateway_handler.go`、`openai_chat_completions.go`、`openai_images.go`
- 前端：`frontend/src/api/admin/ops.ts`、`frontend/src/views/admin/ops/components/OpsUserRequestMonitorCard.vue`、`frontend/src/views/admin/ops/OpsDashboard.vue`
- 技术栈：Go 1.23 + Gin、PostgreSQL、Redis、Wire DI、Vue 3 + TS、Vitest
