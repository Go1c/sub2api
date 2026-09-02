---
status: completed
---

# 禁止购买订阅 — 管理端 UI

## 做什么

在用户管理编辑弹窗增加「禁止购买订阅」开关，提交时写入 `subscription_purchase_disabled`。

## 设计定案

### UI

- 位置：`UserEditModal.vue`，紧挨 `invoice_enabled` 开关
- 控件：checkbox / toggle（与 invoice 同样式）
- `data-test="subscription-purchase-disabled-toggle"`
- 文案（i18n）：
  - 标题：禁止购买订阅
  - 提示：开启后，该用户无法购买订阅套餐
- 提交 payload 包含 `subscription_purchase_disabled: boolean`

### 类型

- `frontend/src/types/index.ts`：`User` / `AdminUser` 增加 `subscription_purchase_disabled?: boolean`

### 测试

- 扩展 `UserEditModal.spec.ts`：镜像 invoice 测试，校验 toggle 提交字段

## 涉及范围

- `frontend/src/components/admin/user/UserEditModal.vue`
- `frontend/src/components/admin/user/__tests__/UserEditModal.spec.ts`
- `frontend/src/types/index.ts`
- `frontend/src/i18n/locales/zh*.ts` / `en*.ts`（admin.users.form 下）

## 验收标准

- [ ] 编辑用户可见「禁止购买订阅」开关，默认反映当前用户字段（缺省 false）
- [ ] 勾选并保存后 API 调用带 `subscription_purchase_disabled: true`
- [ ] 取消勾选保存带 `false`
- [ ] 中英文案齐全
- [ ] 单测通过

## 依赖

- `ban-subscription-purchase-admin-api`
