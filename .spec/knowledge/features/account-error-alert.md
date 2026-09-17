---
name: account-error-alert
description: 账号异常 Telegram 告警：后台定时聚合 ops_error_logs，通知最近窗口内异常账号
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 账号异常 Telegram 告警

后台定时从线上运维错误日志中聚合最近时间窗口内的异常账号，达到阈值后通过 Telegram 发送简短摘要。它是独立后台功能，不参与网关请求热路径。

## 背景 / 目标

- 管理员需要知道最近一段时间内哪些账号不稳定，自己再判断处理。
- 告警内容只保留排障必需信息：账号、错误码、次数、最近时间、错误信息、影响用户邮箱 Top N。
- 不做 AI 分析、不做 P0/P1/P2 优先级、不推建议、不在通知中展示平台 / 分组。
- 账号页可单独开关监控、限制关键字，并用规则覆盖窗口 / 次数 / 推送上限。这与自定义错误码无关：后者只影响调度，不控制 Telegram。

## 设计

- **数据口径**：以线上 `ops_error_logs` 为准，不使用 `account_error_histories`；后者会折叠和节流，不适合判断 10 分钟内真实异常次数。
- **聚合方式**：后台定时任务按配置窗口查询 `ops_error_logs`，优先展开 `upstream_errors` 中的账号失败事件；没有事件数组时回落到行级 `account_id`。影响用户邮箱 Top N 只统计这些已触发异常账号上的 `user_id` 错误次数，通知中展示对应用户邮箱，不展示用户 ID，避免混入无关客户端错误。Top N 只统计本次实际通知的账号，沿用各候选命中的窗口与关键字；冷却或限额排除的账号不计入，也不再按全局阈值重新筛选用户。
- **账号配置**：存在 `accounts.extra.error_alert`。缺省或未写该键视为加入监控；`enabled: false` 才排除。`keywords` 为空表示统计该账号全部错误，否则按关键字（消息包含，大小写不敏感）或状态码字符串匹配。`rules` 可覆盖窗口、最少次数，以及同一账号+关键字在窗口内的 `max_sends`。
- **性能边界**：SQL 内完成窗口过滤、JSONB 展开、账号聚合、排序、账号开关 / 关键字过滤和 `LIMIT`；Go 层只处理聚合后的候选账号。带规则的账号会再查一次（可能低于全局最少次数）。查询使用现有 `created_at` / `account_id, created_at` 索引覆盖最近窗口扫描。
- **通知方式**：Telegram `sendMessage` 纯文本发送，不启用 Markdown/HTML parse mode，避免转义导致发送失败。发送错误在返回前脱敏 Bot Token，HTTP 错误中的请求 URL 不得原样写入日志或 heartbeat。
- **降噪**：无规则时按账号 + 状态码 + 归一化错误信息生成冷却 key。命中规则时改用 Redis（或进程内）计数，限制该账号+关键字在规则窗口内的发送次数。Redis 可用时跨实例冷却 / 计数，不可用时退化为进程内。有 Redis 时还用 leader lock 避免多副本重复发送。

## 已决策

- 全局告警默认关闭；管理员在运维设置里配置 Bot Token、Chat ID、扫描间隔、统计窗口、触发次数、冷却时间和最多账号数后开启。
- 未写 `extra.error_alert` 的存量账号默认加入监控；右上角 JSON 导入和 Codex 向导导入一律写入 `enabled: false`（覆盖源文件），默认关闭，可在账号编辑页开启。关键字和规则也只在账号页配置，不与自定义错误码共用。
- 默认扫描间隔 10 分钟、统计窗口 10 分钟、单账号 5 次触发、同账号同错误 60 分钟冷却、单次最多 10 个账号、展示影响用户邮箱 Top 3（可设为 5 或 0 关闭）。
- 通知表格不包含平台 / 分组，因为这次目标是快速知道“哪些账号有问题”。

## 待解决

- 暂无。

## 相关

- 后端配置：`backend/internal/service/ops_settings.go`
- 账号 extra 解析：`backend/internal/service/account_error_alert.go`
- 后台任务 / Telegram：`backend/internal/service/ops_account_error_alert_service.go`
- 聚合查询：`backend/internal/repository/ops_repo_account_error_alert.go`
- 管理端接口：`backend/internal/handler/admin/ops_settings_handler.go`
- 前端设置：`frontend/src/views/admin/ops/components/OpsSettingsDialog.vue`
- 账号编辑：`frontend/src/components/account/EditAccountModal.vue`
