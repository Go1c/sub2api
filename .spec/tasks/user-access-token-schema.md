---
status: completed
title: 用户 Access Token 数据模型与迁移
---

# 用户 Access Token 数据模型与迁移

## 做什么

新增可撤销长效用户 Access Token 的持久化结构（Ent schema + SQL migration），供后续 service/auth 使用。

## 涉及范围

- `backend/ent/schema/`（新实体，如 `user_access_token.go`）
- `backend/migrations/`（下一序号，当前最新约 `921_...`）
- 必要时 `backend/migrations/migrations.go` / 相关 migration 测试惯例
- 生成 ent 代码（按仓库既有流程）

## 建议字段（可微调命名，语义必须具备）

| 字段 | 说明 |
|------|------|
| `user_id` | 所有者，索引 |
| `name` | 用户可见名称，必填，限长 |
| `token_hash` | 明文 token 的单向哈希（唯一） |
| `token_prefix` | 列表展示用前缀（如前 8 字符），**不是**完整 secret |
| `expires_at` | 过期时间（强制有值；创建时按 1–30 天计算） |
| `last_used_at` | 可选，最近使用 |
| `revoked_at` | 可选，撤销时间；非空即失效 |
| `created_at` / `updated_at` | 时间 mixin |

约束：

- 不存明文 token
- soft-delete 可选；若用 soft-delete，撤销语义仍以 `revoked_at` 为准更清晰
- 查询活跃 token 需能按 hash O(1) 查找

## 验收标准

- [x] 新增 migration 可干净应用到当前 schema
- [x] Ent schema 与 migration 字段一致
- [x] `token_hash` 唯一；`user_id` 有索引
- [x] 无明文 token 列
- [x] 相关 `go test` / ent generate 按仓库惯例通过（至少编译）

## 依赖

无（本功能首卡）
