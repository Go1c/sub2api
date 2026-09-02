---
status: completed
---

# 禁止购买订阅 — Schema / Migration / Domain

## 做什么

为用户表新增「禁止购买订阅」布尔字段，默认不禁止，保证存量用户行为不变。

## 设计定案

### 字段

- 列名 / JSON：`subscription_purchase_disabled`
- 类型：`BOOLEAN NOT NULL DEFAULT FALSE`
- 语义：`true` = 管理员禁止该用户购买订阅；`false` = 允许（默认）
- 不拦截：管理员代开、兑换码兑换、已有订阅使用

### 改动清单

1. **Migration**：`backend/migrations/921_add_user_subscription_purchase_disabled.sql`
   ```sql
   ALTER TABLE users
     ADD COLUMN IF NOT EXISTS subscription_purchase_disabled BOOLEAN NOT NULL DEFAULT FALSE;
   ```
2. **Ent schema**：`backend/ent/schema/user.go` 在 `invoice_enabled` 附近加：
   ```go
   field.Bool("subscription_purchase_disabled").Default(false),
   ```
3. 按项目惯例生成/同步 ent 代码（若仓库要求 `go generate` 则执行；否则最小手改 + 迁移护栏测试覆盖）
4. **Domain**：`service.User` 增加 `SubscriptionPurchaseDisabled bool`
5. **Repo**：镜像 `invoice_enabled` 的 hydrate/set 路径（`user_repo.go` 的 `hydrateInvoiceEnabled` / `setUserInvoiceEnabled`），确保 Get/Update 读写一致
6. 若有 schema 列护栏测试（如 `user_schema_columns_migration_test.go`），补上新列

## 涉及范围

- `backend/migrations/921_add_user_subscription_purchase_disabled.sql`（新建）
- `backend/ent/schema/user.go`
- `backend/ent/*`（生成物，按惯例）
- `backend/internal/service/`（User 结构体定义处）
- `backend/internal/repository/user_repo.go`
- 相关 migration / schema 测试

## 验收标准

- [ ] 存在 migration `921_...`，为 `users.subscription_purchase_disabled` 加列，默认 `FALSE`
- [ ] Ent schema 含同名字段，默认 `false`
- [ ] `service.User` 可读写该字段；`userRepo` Get/Update 持久化正确
- [ ] 新用户 / 存量用户未设置时该字段为 `false`（允许购买）
- [ ] 不修改无关用户字段语义

## 依赖

无
