---
status: completed
---

# 用户重置周限 — Schema

## 做什么

为「每个订阅周期用户仅可手动重置周限一次」增加持久化字段，并贯通 Ent schema、migration、domain 映射，供后续 API 读写。

## 设计定案

- 表 `user_subscriptions` 新增：
  - `weekly_limit_user_reset_at timestamptz NULL`
  - 语义：NULL = 本订阅周期内用户尚未手动重置；非 NULL = 已使用过一次机会（存执行时刻即可）
- **不是**自然月计数；不依赖 starts_at/expires_at 做二次比对（机会绑定本行生命周期）
- 管理员 `AdminResetQuota` **不**读写该字段

## 涉及范围

- `backend/migrations/187_user_subscription_weekly_limit_user_reset_at.sql`（若 187 已被占用则取下一个空号；勿用 900+）
- `backend/migrations/README.md` 约定（forward-only、`ADD COLUMN IF NOT EXISTS`）
- `backend/ent/schema/user_subscription.go`
- `go generate ./ent`（或 `make generate` 中 ent 部分）
- `backend/internal/service/user_subscription.go`（domain 字段）
- `backend/internal/repository/user_subscription_repo.go`（entity ↔ domain 映射读写）
- 相关 stub/mapper 若因 struct 增字段编译失败则最小修补

## 验收标准

- [x] migration 文件存在且可幂等：`ADD COLUMN IF NOT EXISTS weekly_limit_user_reset_at TIMESTAMPTZ NULL`
- [x] Ent schema 含对应 Optional/Nillable Time 字段
- [x] `go generate ./ent` 后工程可编译
- [x] domain `UserSubscription` 含 `WeeklyLimitUserResetAt *time.Time`
- [x] repo 读/写映射该字段（Create/Update/Get 路径不丢字段）
- [x] **不**实现用户重置业务 API（留给下一卡）

## 依赖

无
