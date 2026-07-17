---
status: completed
title: 用户重置周限 — Service / Repo CAS / Handler / Route / DTO
---

# 用户重置周限 — Backend API

## 做什么

实现用户鉴权下的「重置周限」端点：校验所有权与每周期一次机会，效果对齐管理员 weekly-only 重置，并在列表/响应中暴露 remaining。

## 设计定案

### API

- `POST /api/v1/subscriptions/:id/reset-weekly-limit`（用户 JWT，挂在 `backend/internal/server/routes/user.go` 的 `/subscriptions` 组）
- 成功：返回更新后的 `dto.UserSubscription`（含 remaining 字段）
- 失败错误码（稳定 reason，供前端 i18n）：
  - `SUBSCRIPTION_NOT_FOUND`：不存在或非本人（统一不泄露）
  - `SUBSCRIPTION_NOT_USABLE`：非 active / 已过期 / 不可消费（`!IsUsable()`）
  - `SUBSCRIPTION_NO_WEEKLY_LIMIT`：`weekly_limit_usd == nil`（无周限）
  - `SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED`：本周期已重置过（CAS 未命中或 reset_at 非空）

### Service

新增 `UserResetWeeklyLimit(ctx, userID, subscriptionID int64) (*UserSubscription, error)`：

1. `GetByID`；若 `err` 或 `sub.UserID != userID` → `ErrSubscriptionNotFound`
2. `!sub.IsUsable()` → 新错误 `SUBSCRIPTION_NOT_USABLE`
3. `sub.WeeklyLimitUSD == nil` → `SUBSCRIPTION_NO_WEEKLY_LIMIT`
4. `sub.WeeklyLimitUserResetAt != nil` → `SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED`
5. **原子**更新（推荐单一 repo 方法，避免非事务双写竞态）：
   - 条件：`id = ? AND user_id = ? AND weekly_limit_user_reset_at IS NULL AND deleted_at IS NULL`（及必要时 status 条件）
   - 设置：`weekly_usage_usd = 0`，`weekly_window_start = startOfDay(now)`（与 `AdminResetQuota` 一致），`weekly_limit_user_reset_at = now`
   - 影响行数 0 → `SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED`（并发双点）
6. 缓存失效对齐 `AdminResetQuota`：
   - `InvalidateSubCache(userID, gid)` + L1 `Wait()`
   - `billingCacheService.InvalidateSubscription(ctx, userID, gid)`
7. 再 `GetByID` 返回最新行
8. **禁止**改动：`quota_used_usd` / `quota_limit_usd` / daily 窗口 / status / expires_at / exhausted_at

### DTO / 列表

- `dto.UserSubscription` 增加：
  - `weekly_limit_reset_remaining int`（json: `weekly_limit_reset_remaining`）
  - 可选：`weekly_limit_user_reset_at`（若暴露需 omitempty；remaining 为前端主字段）
- remaining 计算：有周限且 `WeeklyLimitUserResetAt == nil` → 1，否则 0
- `UserSubscriptionFromService` / list 映射必须带上 remaining（用户 `GET /subscriptions` 等）

### 范围外

- 不改管理员 reset 语义
- 不写前端

## 涉及范围

- `backend/internal/service/subscription_service.go`（及错误变量定义处）
- `backend/internal/repository/user_subscription_repo.go`（CAS/原子重置方法）
- `backend/internal/service` 中 `UserSubscriptionRepository` 接口定义处
- 所有实现该接口的 stub（unit test / api_contract stub）最小补方法
- `backend/internal/handler/subscription_handler.go`
- `backend/internal/server/routes/user.go`
- `backend/internal/handler/dto/types.go` + `mappers.go` + mapper tests
- 新单测：`subscription_user_reset_weekly_limit_test.go`（或并入现有 reset 测试文件）

## 验收标准

- [x] 路由注册：`POST /subscriptions/:id/reset-weekly-limit`
- [x] 本人 + usable + 有周限 + 未用过：weekly_usage=0，weekly_window_start 更新，reset_at 非空；quota_used 等不变
- [x] 再次调用：`SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED`
- [x] 非本人：`SUBSCRIPTION_NOT_FOUND`
- [x] 无周限：`SUBSCRIPTION_NO_WEEKLY_LIMIT`
- [x] 不可用订阅：`SUBSCRIPTION_NOT_USABLE`
- [x] 成功路径失效 L1 + billing 订阅缓存（与 AdminResetQuota 同序）
- [x] `GET /subscriptions` 响应含 `weekly_limit_reset_remaining`（0 或 1）
- [x] unit 测试覆盖：成功、已用尽、非本人、无周限、不可用、并发 CAS（第二次失败）
- [x] `cd backend && go test -tags=unit ./internal/service -run UserResetWeeklyLimit`（或等价命名）通过

## 依赖

- `user-reset-weekly-limit-schema`
