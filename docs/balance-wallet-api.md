# 站内余额钱包 API

Sub2API 可作为多个外部站点共用的钱包账本。`balance` 是站内货币，接口固定使用 `CNY` 标签，金额直接扣站内余额，不做汇率换算。

Base URL：`https://api.lumio.games`

## 安全模型

- 用户 Bearer JWT 决定扣谁的余额，不接受请求体 `user_id`。
- `X-Balance-Client-Key` 决定哪个已登记的外部服务获准消费余额。
- 扣款只接受 `Authorization` Header 中的 JWT；Cookie、`uat_`、Admin API Key 均不能扣款。
- `bcs_` 密钥只能配置在外部服务端，禁止下发浏览器、桌面端或移动端，也不要写日志。
- 若 LumioAPI 启用了 session binding，接入服务必须按受信代理配置透传用户原始 IP 和 User-Agent。

## 用户扣款

`POST /api/v1/user/balance/debit`

```http
Authorization: Bearer <user-jwt>
X-Balance-Client-Key: bcs_<64hex>
Idempotency-Key: CC20260819-100001
Content-Type: application/json
```

```json
{
  "amount": 19.90,
  "currency": "CNY",
  "purpose": "cchaven_monthly",
  "ref": "CC20260819-100001"
}
```

约束：

- `amount` 必须是 JSON number、大于 0、最多两位小数，最大 `999999999999999999.99`。
- `currency` 只接受精确值 `CNY`。
- `purpose` 是消费方允许列表中的精确值，格式为 1–64 位小写 slug。
- `ref` 去首尾空格后必填，最长 128 个 Unicode 字符，不能包含控制字符。
- `Idempotency-Key` 去首尾空格后为 1–128 位可见 ASCII。
- 请求体拒绝未知字段，因此不能加入 `user_id` 或其它代扣参数。

成功：

```json
{
  "code": 0,
  "message": "ok",
  "reason": "",
  "data": {
    "txn_id": "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
    "amount": 19.90,
    "balance": 3.25000000,
    "currency": "CNY"
  }
}
```

### 幂等

命名空间是“消费方 + 用户 + Idempotency-Key”。同一订单号可被不同消费方或不同用户使用。`19.9` 与 `19.90` 具有相同 fingerprint。

- 同 key、同规范化请求：返回首次 `txn_id`、首次 `balance`。
- 同 key、不同 amount/currency/purpose/ref：`409 IDEMPOTENCY_KEY_CONFLICT`。
- 余额不足不消耗 key；用户充值后可以用完全相同的 key 和请求重试。
- `409 BALANCE_DEBIT_BUSY` 或 `503 BALANCE_STORE_UNAVAILABLE` 应保持同 key、同请求重试；优先遵守 `Retry-After`。

### 稳定错误

| HTTP | reason | 处理 |
|---|---|---|
| 400 | `IDEMPOTENCY_KEY_REQUIRED` | 补订单号，不重试当前坏请求 |
| 400 | `IDEMPOTENCY_KEY_INVALID` | 修正 key |
| 400 | `INVALID_BALANCE_DEBIT_REQUEST` | 修正请求字段 |
| 401 | `INVALID_TOKEN` / `TOKEN_EXPIRED` | 刷新或重新获取用户 JWT |
| 401 | `INVALID_BALANCE_CLIENT` | 检查服务端密钥配置/轮换 |
| 403 | `USER_INACTIVE` | 停止订单流程 |
| 403 | `PURPOSE_NOT_ALLOWED` | 修正消费方 allowed purposes |
| 403 | `INSUFFICIENT_BALANCE` | 引导充值；之后可用同 key 重试 |
| 403 | `BACKEND_MODE_ACTIVE` | 等待用户面恢复 |
| 409 | `IDEMPOTENCY_KEY_CONFLICT` | 订单数据不一致，人工排查 |
| 409 | `BALANCE_DEBIT_BUSY` | 按 `Retry-After` 用同请求重试 |
| 503 | `BALANCE_STORE_UNAVAILABLE` | 退避后用同请求重试 |

余额不足的 `data.balance` 与 `data.required` 都是 JSON number。

## 本人交易流水

JWT 和 `uat_` 均可查询，仅返回 token 所属用户的成功扣款。

```http
GET /api/v1/user/balance/transactions?page=1&page_size=20&purpose=&ref=&client_id=
Authorization: Bearer <jwt-or-uat>
```

- 默认 `page=1`、`page_size=20`，最大 100。
- `purpose`、`ref`、稳定 `client_id` 均为精确过滤。
- 非空 `client_id` 不是合法 UUID 时返回 `400 INVALID_BALANCE_TRANSACTION_QUERY`。
- 排序为 `created_at DESC, id DESC`。

单笔：

```http
GET /api/v1/user/balance/transactions/:txn_id
Authorization: Bearer <jwt-or-uat>
```

访问他人的 `txn_id` 与不存在的交易均返回 404。

## 管理端流水

外部钱包扣款会进入管理员「用户充值和并发变动记录」的默认合并列表：

```http
GET /api/v1/admin/users/:id/balance-history
GET /api/v1/admin/users/:id/balance-history?type=wallet_debit
```

- 默认（无 `type`）与兑换码、管理员加款、分销、优惠码、站内订阅余额支付、外部订阅消费一起按时间倒序合并。
- `type=wallet_debit` 只返回外部钱包扣款。
- 每条 `type` 为 `wallet_debit`，`value` 为负数（例如 `-19.90`），`code` 为 `txn_id`，`notes` 含 `client_name`、`purpose`、`ref`、`txn_id`、扣后余额。
- `total_recharged` 不把扣款算进去。
- 不要把这类记录当成 `balance_payment`（那是站内订阅）。

按用户查看完整钱包账本字段（对齐用户侧 `ListTransactions`，不返回 `secret` / `secret_hash`）：

```http
GET /api/v1/admin/users/:id/wallet-debits
```

使用现有 admin auth；与 `balance-history` 一样不额外要求 step-up。写操作仍只在 `/admin/balance-clients` 上按既有政策门控。

```json
{
  "code": 0,
  "message": "ok",
  "reason": "",
  "data": {
    "txn_id": "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
    "client_id": "9668f69e-32c4-48e9-9992-280951dcb85c",
    "client_name": "CCHaven Control",
    "amount": 19.90,
    "balance_after": 3.25000000,
    "currency": "CNY",
    "purpose": "cchaven_monthly",
    "ref": "CC20260819-100001",
    "created_at": "2026-08-19T08:00:00Z"
  }
}
```

## 管理消费方

使用现有 admin auth。写操作在站点启用 step-up 时需要已完成 TOTP step-up 的管理员 JWT 会话。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/admin/balance-clients` | 创建；201 响应一次性返回 `secret` |
| GET | `/api/v1/admin/balance-clients` | 列表，不返回完整 secret/hash |
| GET | `/api/v1/admin/balance-clients/:id` | 详情 |
| PUT | `/api/v1/admin/balance-clients/:id` | 更新 name、allowed_purposes、status |
| POST | `/api/v1/admin/balance-clients/:id/rotate-secret` | 轮换；响应一次性返回新 secret，旧 secret 立即失效 |
| DELETE | `/api/v1/admin/balance-clients/:id` | 停用，不删除历史流水 |

创建：

```json
{
  "name": "CCHaven Control",
  "allowed_purposes": ["cchaven_monthly"]
}
```

首次部署顺序：运行 migration `928_balance_debit_wallet.sql`，再创建消费方，将一次性 `bcs_` 密钥写入 cchaven-control 服务端 secret manager，最后执行一笔小额扣款和本人流水查询联调。
