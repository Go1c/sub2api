---
status: completed
---

# 禁止购买订阅 — 管理端 Update API

## 做什么

管理员更新用户时，可读写 `subscription_purchase_disabled`，镜像 `invoice_enabled` 全链路。

## 设计定案

### API

- `PATCH/PUT` 管理员更新用户（现有 `UpdateUser`）请求体增加可选：
  - `subscription_purchase_disabled?: boolean`（指针：未提供则不修改）
- 用户详情 / 列表 DTO 响应包含该字段（与 `invoice_enabled` 一致）

### 改动清单

1. `UpdateUserRequest`（`handler/admin/user_handler.go`）加 `SubscriptionPurchaseDisabled *bool`
2. `UpdateUserInput`（`admin_service.go`）加同名字段
3. `admin_user.go` `UpdateUser`：若指针非 nil 则写入 user 并持久化
4. DTO / mapper：`handler/dto/types.go` + `UserFromService*` 映射
5. 现有 admin user API 测试 / contract 如有 invoice 样例，补一行字段

## 涉及范围

- `backend/internal/handler/admin/user_handler.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/service/admin_user.go`
- `backend/internal/handler/dto/types.go` 及 mappers
- 相关 unit / contract 测试

## 验收标准

- [ ] 管理员 `UpdateUser` 传入 `subscription_purchase_disabled: true/false` 可持久化
- [ ] 未传该字段时不覆盖原值
- [ ] 用户列表 / 详情响应含该布尔字段
- [ ] 行为与 `invoice_enabled` 一致（指针语义）

## 依赖

- `ban-subscription-purchase-schema`
