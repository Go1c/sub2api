---
name: balance-debit-wallet
description: 通用外部消费方站内余额扣款、本人流水查询与消费方管理
status: completed
---

# 通用站内余额扣款与流水查询

## 任务 1：冻结钱包请求与错误契约

- 涉及范围：`backend/internal/service`、`backend/internal/handler`
- 验收标准：
  - [x] 金额使用 decimal 解析，仅接受大于 0 且最多两位小数。
  - [x] currency、purpose、ref、Idempotency-Key 按冻结规则校验。
  - [x] canonical fingerprint 将 `19.9` 与 `19.90` 视为同一请求。
  - [x] 钱包响应始终包含非省略的 `reason`。
- 依赖：无

## 任务 2：建立钱包账本数据模型

- 涉及范围：`backend/migrations/928_balance_debit_wallet.sql`、`backend/ent/schema`
- 验收标准：
  - [x] client、transaction、cache invalidation outbox 三张表及约束可重复迁移。
  - [x] 金额与余额快照使用 PostgreSQL numeric，幂等命名空间唯一。
  - [x] 用户删除不级联删除交易，查询与 outbox 索引完整。
  - [x] Ent 生成代码与 schema 同步。
- 依赖：任务 1

## 任务 3：实现原子钱包事务和查询

- 涉及范围：`backend/internal/repository`、`backend/internal/service`
- 验收标准：
  - [x] Read Committed 事务锁定 client 与 user，锁内重查状态和 purpose。
  - [x] 成功扣款、重放、冲突、余额不足及并发行为符合冻结契约。
  - [x] 成功事务同时写账本与按 user 合并的 outbox。
  - [x] JWT/UAT 查询仅返回本人成功流水，过滤、排序和 404 隔离正确。
  - [x] client 密钥仅创建/轮换返回一次，数据库只存 hash 与展示前缀。
- 依赖：任务 2

## 任务 4：接入鉴权、路由、缓存和审计

- 涉及范围：`backend/internal/server`、`backend/internal/handler`、Wire providers
- 验收标准：
  - [x] debit 只接受 Header JWT，并同时验证 `X-Balance-Client-Key`。
  - [x] UAT 白名单仅新增两个交易查询 GET 路径。
  - [x] 钱包接口遵守 backend mode 并返回稳定 reason。
  - [x] outbox worker 最终失效 billing balance 与 API-key auth cache。
  - [x] 用户及管理员路由、step-up、审计与 Wire 接线完整。
- 依赖：任务 3

## 任务 5：文档与回归验证

- 涉及范围：`docs`、`.spec/knowledge`、后端测试套件
- 验收标准：
  - [x] 接入文档说明 server-to-server 密钥边界、API 与首部署顺序。
  - [x] 新功能知识文档和导航同步，状态准确。
  - [x] unit、integration、`go vet -tags integration` 与 lint 校验通过或明确记录环境阻塞。
  - [x] 现有 `/auth/me`、gateway billing、支付和订阅代码未改变。
- 依赖：任务 4

## 验证记录（2026-08-19）

- 钱包相关 unit 包、migration contract、Wire cleanup test 均通过。
- `go test -tags=integration ./...` 与 `go vet -tags integration ./...` 通过；本机无 Docker，repository harness 明确跳过真实 PostgreSQL 集成用例。
- `/Users/cui/go/bin/golangci-lint run ./...` 通过（0 issues）。
- 全仓 `go test -tags=unit ./...` 中仅既有 `internal/server` settings API contract 仍因 `ccswitch_default_model_openai` 期望空值、实际 `gpt-5.4` 失败；本次未修改 settings 实现或该契约测试。
