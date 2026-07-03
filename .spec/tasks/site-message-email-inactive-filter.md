---
status: completed
---

# Site Message Email And Inactive User Filter

## Task

站内信管理新增批量发送邮箱副本开关，并支持全员模式下筛选最近 N 天没有使用服务的活跃用户。

## Acceptance

- 管理员站内信管理页可显式切换是否同时发送到用户邮箱。
- 全员收件范围支持“最近 N 天未使用服务”筛选。
- 后端按 `usage_logs.created_at` 判断用户是否最近使用过服务，包含从未使用过的活跃用户。
- 指定邮箱模式保持原有行为。
- 相关后端 / 前端测试覆盖新增行为。
