---
status: completed
title: 禁止购买订阅 — 知识沉淀
depends_on:
  - ban-subscription-purchase-schema
  - ban-subscription-purchase-admin-api
  - ban-subscription-purchase-enforce
  - ban-subscription-purchase-admin-ui
  - ban-subscription-purchase-user-ui
---

# 禁止购买订阅 — 知识沉淀

## 做什么

用 `spec-steward` 在 `knowledge/features/` 写短文档，并更新 `knowledge/README.md` 索引。

## 文档要点

- 字段：`users.subscription_purchase_disabled`
- 管理端：UserEditModal 开关
- 拦截：`CreateOrder` 订阅单 → `SUBSCRIPTION_PURCHASE_FORBIDDEN`
- 用户提示：无权限购买
- 不覆盖：管理员代开、兑换码、已有订阅使用

## 涉及范围

- `.spec/knowledge/features/user-subscription-purchase-ban.md`（或类似 slug）
- `.spec/knowledge/README.md`

## 验收标准

- [ ] feature 文档存在且 status 反映已实现
- [ ] README 导航有一行入口
- [ ] 与实现一致，无过时描述

## 依赖

- 前述实现卡全部完成后再写
