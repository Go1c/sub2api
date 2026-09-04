---
name: reseller-usage-correlation
description: 下游 sub2API 分销对账：请求头写入 correlation_id，专用增量 export 拉取；不影响其他用户面板与 GET /usage
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 下游分销用量对账

简介：让下游 sub2API（分销）把「他们的一笔消费」稳定对上「我们的一笔 usage_log」。对账键走独立请求头与列；拉取走专用增量接口，与其他用户的控制台用量查询隔离。

## 背景 / 目标

- 下游也跑 sub2API，终端用户先打他们，再由他们作为 apikey 账号打我们。两边各自写 `usage_logs`，各自生成 `request_id`（`client:` + 本进程 `X-Client-Request-ID`），所以按小时拉我们的日志也对不上。
- 现有 `usage_logs.request_id` 是计费去重键 `(request_id, api_key_id)`，冲突则 `ON CONFLICT DO NOTHING`。不能让下游写入这个字段。
- 现有 `GET /api/v1/usage` 在带 `user_id` 时每次 `COUNT(*)` + `OFFSET`，60 RPM 且 Redis 故障 fail-open。不能当作分销对账通道。
- 下游声称每小时增量拉一次，但频率不可信。接口必须在「拉得更勤」时仍然有界，且不拖累其他用户。

目标：上线后增量可 1:1 对上；热路径几乎不增成本；其他用户的面板与 `GET /usage` 行为不变。

## 设计

### 隔离（其他用户）

| 面 | 其他用户 |
|----|----------|
| 控制台 / `GET /usage` / dashboard | **不改路径、限流、DTO、SQL** |
| 网关计费 / `request_id` 去重 | **不改** |
| `usage_logs` 新列 | 可空 `correlation_id`，未带头时为 NULL；PG 无 default 的 `ADD COLUMN` 为元数据变更，不改写表 |
| 网关 | 每个请求多一次头读取；非法头丢弃，不 4xx |
| 对账查询 | 只扫调用者自己的 `user_id`；独立配额；不占用 `user-usage-list` 桶 |
| 索引 | 第一版**不建** `correlation_id` 索引，避免大表建索引拖慢全表写入 |

共享池里他们仍会占少量连接，但单次是 `user_id + id > after_id` 的有界顺扫，上限 10 次/分钟、同用户并发 1。比该用户自己刷面板（`COUNT(*)`、60 RPM）更轻。

### 关联协议

下游在进自己网关时已有 `ClientRequestID`（UUID）。出站打我们时带：

```http
X-Sub2-Request-ID: <下游 ClientRequestID>
```

- 最长 64 字节；去空白后须为合法 UTF-8。非法或过长 **忽略**（当没传），网关照常计费。
- **不**复用 `X-Client-Request-ID` / `x-client-request-id`（Claude CLI 模仿会改写或丢掉）。
- **不**写入 `usage_logs.request_id`。我们照旧用本进程生成的 ID 做计费去重。
- 落库字段：`usage_logs.correlation_id`（原文）。未传则为 NULL。

对账（下游本地 join）：

```text
我们.correlation_id  ==  下游 ClientRequestID
（即他们 usage_logs.request_id 去掉前缀 "client:" 的那一段）
```

历史日志没有该头，对不上。只保证本功能上线后的增量。

OpenAI WS / 部分 Codex 路径的计费 `request_id` 可能走上游 ID 而不是 `client:` 前缀。对账以 `correlation_id` 为准，不要用两边的 `request_id` 互相对。

### 增量导出

`GET /api/v1/usage/export`

鉴权：本人 JWT 或 `uat_`（只读）。**不**开放 Admin usage。路由必须注册在 `GET /usage/:id` **之前**，避免 `export` 被当成 id。

Query：

| 参数 | 规则 |
|------|------|
| `after_id` | 可选。上一页最后一条 `id`。`WHERE user_id = $caller AND id > $after_id` |
| `since` | 可选。RFC3339。仅在没有可用 `after_id` 时作首次对齐；`created_at >= since`，且 `since` 不得早于 **现在−24h** |
| `limit` | 默认 100，最大 **500** |

必须提供 `after_id` 或 `since` 之一，否则 400。两者都有时以 `after_id` 为准（严格增量）。禁止 `after_id=0` 当全量扫描：没有游标就用 `since`。

响应信封仍是 `{code,message,data}`；`data` 无 `total` / 无 COUNT：

```json
{
  "items": [ { "id", "correlation_id", "request_id", "created_at", "api_key_id", "model", "requested_model", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_cost", "actual_cost", "billing_type" } ],
  "next_after_id": 12345,
  "has_more": true
}
```

- `ORDER BY id ASC`，不用 OFFSET。
- 不 hydrate 用户 / API Key / 分组 / 订阅。
- 空页：`items=[]`，`has_more=false`，`next_after_id` 可省略或等于请求的 `after_id`。
- `correlation_id` **只出现在本接口**，不改现有 `GET /usage` DTO。

下游：存 `next_after_id`，之后只带 `after_id`。join 在他们库上完成。

不是「最近一小时打成一个 JSON 包」。我们不推送、不打包文件。每次 HTTP 最多 `limit` 条（默认 100，最大 500）。几千条/小时由他们的任务循环拉完：

1. **第一次对齐**（还没有游标）：`since=<最多 24h 内的起点>`，例如现在−1h。返回一页 `items` + `next_after_id` + `has_more`。
2. **同一轮还没拉完**：带着上一页的 `next_after_id` 再请求，直到 `has_more=false`。
3. **以后每小时**（或他们自己的间隔）：只带上次存下的 `after_id`。这时拿到的是「上次拉完之后新产生的行」，若他们真的一小时跑一次，大约就是这一小时的增量，而不是每次重扫最近一小时。

举例：一小时 3000 条、`limit=500` → 一轮 6 次请求（约几十秒，远低于 10 次/分钟）。他们若要把这一轮结果存成一个本地 JSON，是下游自己 `concat(items)`，不是我们返回一个大包。

```http
GET /api/v1/usage/export?since=2026-09-04T06:00:00Z&limit=500
Authorization: Bearer uat_...
```

```http
GET /api/v1/usage/export?after_id=88100231&limit=500
Authorization: Bearer uat_...
```

空闲时（没有新消费）第二步起就是空页，成本接近一次主键顺扫 0 行。

### 限流与稳定性

独立配额 `user-usage-export`，按 `user_id`（缺则 IP）：

| 项 | 值 |
|----|-----|
| 频率 | 10 次/分钟 |
| Redis 故障 | **fail-closed**（429），不得 fail-open 打库 |
| 同用户并发 | 1（Redis NX 锁，TTL 30s，结束时删除；冲突 429 + `Retry-After`） |
| `limit` | ≤ 500 |

不提供按 `correlation_id` 反查、不提供 webhook 每笔推送、不把 dashboard/stats 当对账通道。

### 实现面

- 迁移：`usage_logs.correlation_id VARCHAR(64) NULL`，无 default、无 index。分区表上同样只加列。
- **不改 Ent schema、不 `go generate`**：`usage_logs` 写入走 raw SQL；控制台 / `GET /usage` 的 SELECT 也不暴露该列。
- `RecordUsage` / OpenAI billing insert 写入 context 中的关联 ID。
- 网关入口读取 `X-Sub2-Request-ID` 放入独立 ctx key（不要改 `ClientRequestID` 中间件的生成逻辑）。
- `uat_` 白名单显式加入 `GET /api/v1/usage/export`（`/usage/` 下非嵌套 GET 会落到 `/:id`；路由必须注册在 `/:id` 之前）。
- 下游（他们的 sub2API）：对指向上游池的 apikey 出站增加该头，并放进出站白名单。本仓库若一并做出站自动带上头，他们升级即可对齐；不是我们上线的阻塞项。
- 接入说明：仓库 `docs/reseller-usage-export.md`。

## 已决策

- 对账键是独立头 + `correlation_id` 列，不是 `usage_logs.request_id`（避免计费去重被对方 ID 误伤）。
- 拉取走专用 export，不复用 `GET /usage`（避免 COUNT/OFFSET 与 60 RPM fail-open）。
- 第一版不加 `correlation_id` 索引、不做点查（大表索引会拖所有写入）。
- 头非法则忽略，不失败请求（保护网关，也保护未升级的下游）。
- 其他用户的用量 API 与计费路径保持原样。

## 待解决

- 无。下游出站改动由对方仓库负责；本仓库可随后加「出站自动带 `X-Sub2-Request-ID`」，不阻塞本功能交付。

## 相关

- [[user-access-token]]（`uat_` 只读白名单含 export）
- [[user-request-monitoring]]（`request_id` 关联的是运维抓取，不是分销对账）
- ADR：`.spec/decisions/2026-09-04-reseller-usage-correlation.md`
- 给下游的接入说明：`docs/reseller-usage-export.md`
- 计费去重：`backend/internal/service/gateway_usage_billing.go` 的 `resolveUsageBillingRequestID`
- 现有用量列表：`backend/internal/handler/usage_handler.go`、`backend/internal/repository/usage_log_repo.go`
