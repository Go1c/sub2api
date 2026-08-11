---
status: completed
title: 沉淀用户 Access Token 功能知识
---

# 沉淀用户 Access Token 功能知识

## 做什么

功能落地后，用 `spec-steward` 写 feature 文档并更新 `knowledge/README.md` 索引。

## 文档要点

路径建议：`.spec/knowledge/features/user-access-token.md`

写清：

- 产品目的与非目标
- Token 形态（opaque、hash 存储、一次性明文）
- 有效期默认 7 / 最大 30
- 授权范围白名单路径
- 管理 API 与鉴权行为
- 本人隔离保证
- 与 sk- API Key（网关推理密钥）的区别
- 相关文件索引

## 验收标准

- [x] feature 文档存在且 frontmatter 合法
- [x] `knowledge/README.md` 有一行导航
- [x] 与实现一致，无过时描述

## 依赖

- 后端鉴权与管理 API 已实现（建议 3、4 卡完成后）
