---
name: reseller-usage-correlation
description: 分销用量对账用独立 correlation 头/列 + 专用增量 export，不用 usage_logs.request_id，也不复用 GET /usage
---

# 下游分销用量对账的关联键与拉取通道

- 状态: 生效

## 背景

下游 sub2API 需要把他们的一笔消费对上我们的一笔 `usage_log`。两边各自生成 `request_id`，现有列表接口也不适合被当成对账拉取通道。

## 决策

1. **关联键**：请求头 `X-Sub2-Request-ID`（下游 `ClientRequestID`）写入 `usage_logs.correlation_id`。不写入、不替换计费用的 `usage_logs.request_id`。
2. **拉取**：`GET /api/v1/usage/export`，按 `id` 游标增量，无 COUNT/OFFSET；独立 10 RPM、fail-closed、同用户并发 1。
3. **第一版不加 `correlation_id` 索引**，也不提供按该字段反查。下游拉增量后在本地 join。
4. **不改** 其他用户的 `GET /usage` / dashboard / 计费去重。

## 替代方案（未采用）

- **让下游 ID 成为我们的 `request_id`**：`(request_id, api_key_id)` 冲突会 `DO NOTHING`，对方可控 ID 可能导致漏扣费。
- **让下游保存我们响应里的 `X-Client-Request-ID`，再拼 `client:` 去对 `request_id`**：依赖内部前缀；`X-Request-ID` 与 usage 的 `request_id` 不是同一个值，容易对空；WS/Codex 路径本来就不走 `client:` 前缀。
- **复用 `GET /usage` 做小时拉取**：有 `user_id` 时每次 `COUNT(*)` + OFFSET，60 RPM 且 Redis fail-open；频率「说不好」时会打大表，并与所有用户的面板配额混在一起。
- **复用 `X-Client-Request-ID` 当对账头**：与 Claude CLI 模仿头冲突，会被改写或从出站白名单丢掉。
- **Webhook 每笔推送**：增加写路径与重试，比有界增量轮询更伤稳定性；对方也不要求实时。
- **给 `correlation_id` 建索引并提供点查**：大表 `CREATE INDEX` 影响全表写入；对账不需要点查。
