---
status: completed
---

# 用户重置周限 — Knowledge

## 做什么

把用户自助重置周限的规则与实现锚点写入功能知识，避免与管理员重置混淆。

## 涉及范围

- `.spec/knowledge/features/subscription-credit-pool.md`（增补一小节）  
  和/或 `.spec/knowledge/features/subscription-admin.md`（若管理员文中需交叉链接）
- 若新增独立短文：更新 `.spec/knowledge/README.md` 索引（按 knowledge 规范）

## 应记录内容

- 用户：`POST /api/v1/subscriptions/:id/reset-weekly-limit`
- 字段：`weekly_limit_user_reset_at` / 响应 `weekly_limit_reset_remaining`（0|1）
- 周期 = 单条订阅记录生命周期，非自然月
- 效果：仅周窗用量；对齐 `AdminResetQuota` weekly-only
- 管理员重置不消费用户机会
- 权限：仅本人 + usable + 有周限 + 未用过
- 前端入口：用户 `SubscriptionsView` 可用卡

## 验收标准

- [ ] 知识文档可检索到上述规则与关键文件路径
- [ ] 明确区分 admin vs user 两条路径
- [ ] 索引（如有新文）已更新

## 依赖

- `user-reset-weekly-limit-api`
- `user-reset-weekly-limit-frontend`（可在前后端合并后写最终锚点）
