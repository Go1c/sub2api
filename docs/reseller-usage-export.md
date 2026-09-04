# 下游分销用量对账接入说明

给下游 sub2API（分销）用：把「你们的一笔消费」对上「我们的一笔 `usage_log`」。

- 产品：LumioAPI
- Base：`https://api.lumio.games`
- 对账键：请求头 `X-Sub2-Request-ID` → 我们落库 `usage_logs.correlation_id`
- 拉取：`GET /api/v1/usage/export`（**不要**用 `GET /api/v1/usage`）

历史日志没有该头，对不上。只保证本功能上线后的增量。

生产可用性：先按本文改你们的出站头和对账任务。接口合入 `publish` 并部署后才在 `api.lumio.games` 生效；部署前该路径会 404。我们推送 / 开 PR 后会把链接给你。

---

## 1. 出站打我们时带关联头

你们进自己网关时已有 `ClientRequestID`（UUID）。出站打我们（`sk-` 调模型）时带：

```http
X-Sub2-Request-ID: <你们的 ClientRequestID>
```

规则：

| 项 | 要求 |
|----|------|
| 值 | 你们的 `ClientRequestID` 原文。一般是裸 UUID，不要加 `client:` 前缀 |
| 长度 | 去空白后 ≤ 64 字节，合法 UTF-8 |
| 非法 / 过长 / 未传 | 我们**忽略**（当没传），请求照常计费，该行 `correlation_id` 为 `null` |
| 不要用 | `X-Client-Request-ID` / `x-client-request-id`（Claude CLI 模仿会改写或丢掉） |
| 不要用 | 我们响应里的 `X-Request-ID`（那不是用量对账键） |

请把该头加入你们出站白名单，并在指向上游池的 apikey 请求上发送。

---

## 2. 对账规则（在你们本地 join）

```text
我们.correlation_id  ==  你们的 ClientRequestID
（即你们 usage_logs.request_id 去掉前缀 "client:" 的那一段）
```

不要用两边的 `request_id` 互相对：

- 我们的 `usage_logs.request_id` 是计费去重键 `(request_id, api_key_id)`，冲突则丢弃写入。不能写成你们的 ID。
- OpenAI WS / 部分 Codex 路径的计费 `request_id` 可能走上游 ID，不是 `client:` 前缀。对账以 `correlation_id` 为准。

`correlation_id` **只出现在 export 接口**，不会出现在控制台或 `GET /api/v1/usage`。

---

## 3. 拉增量：`GET /api/v1/usage/export`

鉴权：本人 JWT，或个人资料里创建的 `uat_`（只读）。`sk-` 不能调这个接口。

```http
GET /api/v1/usage/export?after_id=88100231&limit=500
Authorization: Bearer uat_...
```

### Query

| 参数 | 规则 |
|------|------|
| `after_id` | 上一页最后一条 `id`（> 0）。`WHERE user_id = 调用者 AND id > after_id` |
| `since` | RFC3339。仅在还没有游标时作首次对齐；`created_at >= since`，且不得早于 **现在 − 24h** |
| `limit` | 默认 100，最大 **500**（传更大也会被封顶） |

必须提供 `after_id` 或 `since` 之一，否则 HTTP 400。两者都有时以 `after_id` 为准。禁止 `after_id=0` 当全表扫描。

不是「最近一小时打成一个 JSON 包」。我们不推送、不打包文件。每次最多 `limit` 条。

### 成功响应

信封是 `{code,message,data}`。`code === 0` 为成功。`data`：

```json
{
  "items": [
    {
      "id": 88100231,
      "correlation_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "request_id": "client:3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "created_at": "2026-09-04T07:12:01Z",
      "api_key_id": 12,
      "model": "gpt-5",
      "requested_model": "gpt-5",
      "input_tokens": 120,
      "output_tokens": 40,
      "cache_creation_tokens": 0,
      "cache_read_tokens": 0,
      "total_cost": 0.012,
      "actual_cost": 0.01,
      "billing_type": 0
    }
  ],
  "next_after_id": 88100231,
  "has_more": true
}
```

- `ORDER BY id ASC`，无 `total` / 无 `COUNT(*)` / 无 OFFSET。
- 不 hydrate 用户 / API Key / 分组。
- 空页：`items=[]`，`has_more=false`，`next_after_id` 可能省略，或等于请求的 `after_id`。
- `correlation_id` 可能为 `null`（该请求没带合法头）。

`billing_type`：`0` 钱包余额，`1` 订阅套餐，`2` 混合扣费。金额单位 USD。

### 分页循环

1. **第一次对齐**（还没有游标）：`since=<最多 24h 内的起点>`，例如现在 − 1h。
2. **同一轮还没拉完**：带着上一页的 `next_after_id` 再请求，直到 `has_more=false`。本地 `concat(items)`。
3. **以后**：只带上次存下的 `after_id`。拿到的是「上次拉完之后新产生的行」。

举例：一小时 3000 条、`limit=500` → 一轮 6 次请求，远低于下面的 10 次/分钟。

```bash
UAT=uat_xxx
BASE=https://api.lumio.games

# 首次对齐（把 since 换成最多 24h 内的 RFC3339）
curl -sS -H "Authorization: Bearer $UAT" \
  "$BASE/api/v1/usage/export?since=2026-09-04T06:00:00Z&limit=500"

# 之后只带游标
curl -sS -H "Authorization: Bearer $UAT" \
  "$BASE/api/v1/usage/export?after_id=88100231&limit=500"
```

伪代码：

```text
cursor = load_saved_after_id()          # 没有则 nil
if cursor is nil:
    page = GET export?since=<now-1h>&limit=500
else:
    page = GET export?after_id=<cursor>&limit=500

loop:
    save items locally
    if page.has_more:
        cursor = page.next_after_id
        page = GET export?after_id=<cursor>&limit=500
        continue
    persist cursor = page.next_after_id or cursor
    stop
```

空闲时（没有新消费）第二步起就是空页。

---

## 4. 限流（必须当硬限制）

独立配额，不占用控制台 `GET /usage` 的桶。

| 项 | 值 |
|----|-----|
| 频率 | **10 次/分钟** / 用户 |
| Redis 故障 | fail-closed：也返回 429，不会放行打库 |
| 同用户并发 | **1**（进行中的 export 未结束时再打会 429） |
| `limit` | ≤ 500 |

429 时退避，优先尊重 `Retry-After`（并发锁约为 30 秒）。

两种 429 形态都要当限流处理：

```json
{"error":"rate limit exceeded","message":"Too many requests, please try again later"}
```

```json
{"code":429,"message":"usage export already in progress, retry later","reason":"USAGE_EXPORT_IN_FLIGHT"}
```

不要并行开多个 export 任务，不要用 `GET /api/v1/usage` 做对账拉取。

---

## 5. 错误

成功：HTTP 200，`{"code":0,"message":"success","data":{...}}`。

| HTTP | 何时 |
|------|------|
| 400 | 缺 `after_id`/`since`、`after_id=0`、非法 `since`、`since` 早于 24h、非法 `limit` |
| 401 | 未登录 / token 无效 |
| 403 | `uat_` 调了白名单外的路径 |
| 429 | 超 10 RPM、Redis 故障、或同用户已有一次 export 在飞 |

没有按 `correlation_id` 反查，也没有 webhook 每笔推送。

---

## 6. 你们需要改什么

1. 出站网关：对打我们的请求加 `X-Sub2-Request-ID: <ClientRequestID>`，并加入出站头白名单。
2. 对账任务：用 `uat_` 调 `GET /api/v1/usage/export`，按上面循环拉，本地用 `correlation_id` join。
3. 持久化 `after_id`，不要每小时用 `since` 重扫。

`uat_` 的创建与白名单见 [`user-access-token-api.md`](./user-access-token-api.md)。
