---
name: admin-settings-idempotency
description: 管理员设置保存（PUT /api/v1/admin/settings）的隐式幂等、部分更新，以及 fork 字段必须走 parseSettings 才能在管理端读回 / 排查保存后输入框变空或重复写入时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 管理员设置保存幂等性

简介：`PUT /api/v1/admin/settings` 带有服务端隐式幂等保护，覆盖网络重试、浏览器重复提交或客户端状态更新延迟导致的重复写入。

## 背景 / 目标

管理员保存设置时，网络重试 / 浏览器重复提交 / 客户端状态更新延迟可能导致同一设置被重复写入。需要保证同一管理员短时间内重复提交同一 body 时只执行一次写入。

## 设计

服务端隐式幂等：

- 服务端按 `管理员 ID + 规范化请求 + 更新模式` 生成幂等键。
- 同一管理员在 **60 秒**内重复提交同一设置 body 时，只执行第一次写入。
- 后续重复请求直接复用第一次成功响应，并返回响应头 `X-Idempotency-Replayed: true`。
- 如果第一次请求仍在处理中，重复请求会收到现有幂等协调器的「处理中冲突」响应。

前端配合：设置页在保存请求进行中阻止再次提交，并保持保存按钮禁用到响应结束。

### 渠道状态 Banner 单字段更新

`{"channel_monitor_status_banner":"..."}` 是管理员设置接口支持的独立部分更新：

- 只规范化并写入 `channel_monitor_status_banner`，不会用请求结构体的零值覆盖其他系统设置。
- 响应 `data.channel_monitor_status_banner` 返回规范化后的文案；空字符串表示清空。
- 幂等 payload 包含 `status_banner_only` 模式位，避免单字段请求与字段值相同的完整设置请求互相重放。

### PostgreSQL 异常恢复

生产库曾出现字面 `UPDATE settings SET value/updated_at ...` 返回 `23505 settings_key_key` 的异常。批量设置写入保留 update-first 路径，但整批放在 savepoint 内：

1. `UPDATE` 返回 `23505` 后先 `ROLLBACK TO SAVEPOINT`，清除 PostgreSQL 的 aborted transaction 状态。
2. 再对整批执行 raw `INSERT ... ON CONFLICT DO UPDATE`。
3. fallback 成功后释放 savepoint 并提交；其他错误仍向上传播。

不能在收到 PostgreSQL 语句错误后直接在同一事务中继续 upsert，否则只会得到 `current transaction is aborted`，并让幂等记录进入 retry backoff。

### fork 字段必须在 parseSettings 读回

管理员 `GET` / `PUT` 成功响应都走 `GetAllSettings` → `parseSettings`。只把 key 写进 defaults 和 `setting_update.go`、不在 `parseSettings` 赋给 `SystemSettings`，管理端响应会永远是空串。前端保存后用响应体回填表单，空串会盖掉刚填的值，看起来像「点保存没反应」。

公开设置若直接读 raw map，用户侧仍可能拿到已写入的值；管理端读回缺口会在下一次整页保存时把空串写回库。

2026-08-16 已补 CCSwitch 五个默认模型的 parse（`ccswitch_default_model_*`）。OpenAI 空值回退 `gpt-5.4`，其余允许空串。新增 fork 设置字段时，defaults / write / parse / 管理员 DTO / 公开 DTO 必须同一条链补齐。

## 已决策

- **幂等键 = 管理员 ID + 规范化请求 + 更新模式**——同管理员、同内容、同更新语义才视为重复。
- **60 秒窗口**——短时间内的重试/重复提交去重。
- **服务端幂等是最终防线**——前端禁用按钮只是第一层；服务端覆盖网络重试、浏览器重复提交、客户端状态延迟等前端拦不住的情况。
- **复用现有幂等协调器**——「处理中冲突」语义沿用既有协调器，不另起一套。

## 待解决

（源文未列出未决问题。）

## 相关

- 接口：`PUT /api/v1/admin/settings`、`GET /api/v1/admin/settings`
- 响应头：`X-Idempotency-Replayed: true`（重放命中时）
- 设置键：`channel_monitor_status_banner`、`ccswitch_default_model_*`
- 读回：`backend/internal/service/setting_parse.go` 的 `parseSettings`
