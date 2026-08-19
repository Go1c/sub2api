---
name: balance-debit-wallet
description: 多外部站共用的站内余额原子扣款账本、本人流水查询与消费方密钥管理；接钱包或排查幂等/缓存一致性时查
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 通用站内余额扣款账本

简介：LumioAPI 提供 server-to-server 钱包扣款能力。用户 JWT 决定余额归属，`bcs_` 消费方密钥决定谁可扣款；成功扣款写不可变账本，并通过 durable outbox 失效两类余额快照缓存。

## 背景 / 目标

- 让 CCHaven 等多个外部站复用同一 `users.balance`，不引入订单、订阅或套餐语义。
- 解决同用户并发扣款、响应丢失重放、余额不足后重试和缓存旧余额问题。
- 给 JWT/`uat_` 提供严格本人隔离的成功交易查询。

## 设计

### 身份与权限

- debit 只接受 Header JWT；Cookie、`uat_`、Admin API Key 均返回稳定 `INVALID_TOKEN`。
- 外部服务同时提供 `X-Balance-Client-Key`；数据库只存 SHA-256 与展示前缀。
- client 在事务内再次锁定并校验 status、当前 secret hash 和 allowed purpose；轮换提交后旧密钥对尚未进入事务的请求立即失效。
- 交易查询接受 JWT/`uat_`，repository 的所有查询都包含 token 所属 `user_id`。
- 管理端 `GET /api/v1/admin/users/:id/balance-history` 默认合并 `balance_debit_transactions`，类型为 `wallet_debit`，金额为负；`total_recharged` 不含扣款。
- 管理端 `GET /api/v1/admin/users/:id/wallet-debits` 按用户返回与本人流水对齐的账本字段，不返回 `secret` / `secret_hash`。

### 原子性与幂等

- `balance_wallet_repo.go` 使用 PostgreSQL Read Committed 和 `FOR UPDATE`，先锁 client、再锁 user。
- 金额从 JSON raw number 经 `shopspring/decimal` 校验，SQL 以文本参数执行 numeric 运算，不经过 `float64`。
- 唯一键是 `(balance_client_id, user_id, idempotency_key_hash)`；fingerprint 使用规范化 amount/currency/purpose/ref。
- 成功重放读取账本中的首次 txn/余额；余额不足回滚且不写流水，因此充值后可复用 key。
- 交易表没有指向 users 的级联 FK，保留财务审计 user_id；client FK 为 RESTRICT，停用不删除。

### 缓存一致性

- 成功事务同时 upsert `balance_cache_invalidation_outbox`，每用户合并一条任务。
- worker 依次失效 billing balance cache 与包含 User.Balance 的 API-key auth cache；任一失败则退避重试。
- claim token 防止 worker 删除领取后又被新扣款合并更新的事件。缓存失败发生在事务提交后，不会把已成功扣款改报 5xx。

### HTTP 与审计

- 钱包接口使用固定 `{code,message,reason,data}` 信封，成功也显式输出空 reason；金额由 `WalletNumber` 直接编码为 JSON number。
- 流水列表在 service 层校验非空 `client_id` UUID，非法值稳定返回 `400 INVALID_BALANCE_TRANSACTION_QUERY`，不会下沉成数据库 503。
- backend mode 使用 `BACKEND_MODE_ACTIVE`；busy 响应带 `Retry-After`。
- debit 审计省略请求体和凭证，只附加 client_id/user_id/txn_id/purpose/ref/金额摘要。
- `X-Balance-Client-Key` 未加入 CORS allow headers，定位为服务端调用。

## 已决策

- `CNY` 只是统一站内余额标签，不换汇。
- purpose 是普通账本标签，不创建 LumioAPI 订单、订阅、权益、到期日或续费状态。
- 第一版只支持扣款和查询；退款/冲正必须以后追加独立账本事件，不修改历史交易。
- 管理端 DELETE 只将 client 设为 inactive；创建和轮换才一次性返回明文 secret。

## 待解决

- 无。后续若增加冲正/退款，需另立设计与迁移。

## 相关

- Migration / schema：`backend/migrations/928_balance_debit_wallet.sql`、`backend/ent/schema/balance_debit_*.go`
- Service / repository：`backend/internal/service/balance_wallet*.go`、`backend/internal/repository/balance_wallet_repo.go`
- HTTP / auth：`backend/internal/handler/balance_wallet_handler.go`、`backend/internal/handler/admin/balance_client_handler.go`、`backend/internal/handler/admin/user_handler.go`、`backend/internal/server/middleware/balance_wallet.go`
- 对外 API：`docs/balance-wallet-api.md`
- 外部开发 Agent 提示词：`docs/balance-wallet-integration-agent-prompt.md`
- UAT 权限：[`user-access-token.md`](./user-access-token.md)
