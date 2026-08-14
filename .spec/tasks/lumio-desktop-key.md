---
status: completed
title: Lumio Desktop Key 账号级唯一与并发幂等
---

# Lumio Desktop Key 账号级唯一与并发幂等

## 做什么

复用现有 `POST /api/v1/keys`，让固定名称 `Lumio Codex Desktop` 在同一账号下始终复用一个未删除 Key，并由数据库保证多设备并发初始化不会重复创建。

## 涉及范围

- `backend/internal/service/api_key_service.go`
- `backend/internal/repository/api_key_repo.go`
- `backend/ent/schema/api_key.go` 与生成代码
- `backend/migrations/923_lumio_desktop_api_key_unique.sql`
- 对应 unit / integration / migration tests

## 验收标准

- [x] 已有固定名称 Key 直接复用并返回完整现有 DTO
- [x] 首次请求创建正常 Key
- [x] 不同幂等键/不同设备并发时数据库中只有一个未删除保留 Key
- [x] 软删除保留 Key 后可创建替代 Key
- [x] 普通名称 Key 的创建冲突语义不变
- [x] 历史重复凭据全部保留，仅非 canonical 行改显示名
- [x] Ent schema、迁移与 repository 查询一致
- [x] unit / integration / migration tests 通过

## 依赖

无；与 `lumio-desktop-config` 串行执行以便逐卡审查
