---
status: completed
---

# 修复渠道状态 Banner 单字段保存失败

## 现象

- `dc0a820f2` 已部署后，管理页能显示「自定义 Banner」。
- `GET /api/v1/settings/public` 与管理员设置读取能暴露 `channel_monitor_status_banner`。
- 管理页仅提交 `{ "channel_monitor_status_banner": "..." }` 时，首次 `PUT /api/v1/admin/settings` 返回 `internal error`；相同 body 重试进入 `idempotent request is in retry backoff window`。
- 失败后公开设置仍为空，保存按钮保持 dirty。

## 根因

- 生产日志显示批量设置写入在 `smtp_use_tls` 的 `UPDATE` 上触发 PostgreSQL `23505 settings_key_key`。
- 现有 fallback 随后直接在同一事务中执行 upsert，但 PostgreSQL 已将事务标记为 aborted，因此返回 `current transaction is aborted` 并导致首次 PUT 500。
- 修复需要在可恢复的事务边界内回滚失败语句，再执行已验证的 raw upsert fallback。

## 做什么

定位并最小修复 Banner 单字段保存链路。重点覆盖管理员 settings 的隐式幂等包装、部分更新语义、响应捕获和设置持久化；不要改 `release/*` 或 `publish`，不要整包重构 settings。

## TDD 要求

1. 先新增能复现首次 PUT 失败的回归测试，并确认红灯原因与线上现象一致。
2. 最小实现使测试通过。
3. 覆盖成功保存、响应字段、清空隐藏语义和幂等成功重放；失败请求不得被误判为成功。

## 验收标准

- [x] 管理员单字段 `PUT /api/v1/admin/settings` 返回 HTTP 200。
- [x] 成功响应 `data.channel_monitor_status_banner` 等于规范化后的文案。
- [x] `GET /api/v1/admin/settings` 与 `GET /api/v1/settings/public` 随后均返回该文案。
- [x] 空字符串保存成功且公开值为空，状态页据此隐藏。
- [x] 相同成功请求在幂等窗口内重试可重放成功，不返回 backoff。
- [x] 不覆盖请求未包含的其他系统设置。
- [x] 后端相关单元 / 集成编译校验通过；本次未改前端。

## 验证记录

- TDD 红灯：仓储 fallback 未创建 savepoint；Banner 单字段 PUT 把 `registration_enabled` 从 `true` 覆盖为 `false`。
- 通过：`go test -tags=unit ./...`
- 通过：`go test -tags=integration ./...`
- 通过：`go vet -tags integration ./...`
- `golangci-lint v2.9` 只报告两个未改文件的既有 gofmt 问题：`admin/user_handler.go:76`、`service/admin_service.go:167`。

## 相关文件

- `backend/internal/handler/admin/setting_handler_update.go`
- `backend/internal/service/setting_update.go`
- `backend/internal/service/setting_public.go`
- `backend/internal/handler/admin/*settings*_test.go`
- `.spec/knowledge/features/admin-settings-idempotency.md`
