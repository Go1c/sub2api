---
name: affiliate-tier-rebate
description: 阶梯式邀请返利：管理员配置 L1-L4 门槛与返利比例，用户页展示后端计算进度。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 阶梯式邀请返利

简介：邀请返利从单一全局比例扩展为 L1-L4 四档阶梯。管理员配置每档邀请人数、邀请充值总额和返利比例；后端按用户当前进度计算实际返利比例，前端只展示后端返回的数据。

## 背景 / 目标

- 原邀请返利缺失配置时会回退到 20%，容易让用户页显示与后台预期不一致。
- 新需求要求等级固定为 L1-L4，但每档门槛和返利比例由后台配置，不能把示例数值写成生产常量。
- 用户页需要展示当前等级、下一等级进度、邀请充值总额和当前实际返利比例。

## 设计

### 后端配置

- 新增设置键 `affiliate_rebate_tiers`，JSON 数组存储四档配置。
- 配置结构：`level`、`min_invitees`、`min_recharge`、`rebate_rate_percent`。
- 后端保存时规范化为固定 L1-L4；负数门槛归零，比例 clamp 到 0-100，比例为 `null` 表示该等级未配置/不生效。

### 等级判定

- 邀请人数和邀请充值总额必须同时达标。
- 多档同时达标时取最高档。
- 发放返利时，用“已完成邀请充值总额 + 本次订单金额”判定等级；一笔订单把用户推过门槛时，该笔订单按新等级计算。
- 只有用户实际付费订单产生返利：余额充值订单、外部支付购买订阅订单；手工余额兑换码和余额内扣购买订阅不产生返利。
- 用户专属返利比例仍是显式覆盖：用户 `aff_rebate_rate_percent` 非空时优先使用；清空后回到阶梯配置。

### API / 前端

- `/api/v1/user/aff` 返回：
  - `effective_rebate_rate_percent`：实际返利比例，未配置或未达标时为 `null`。
  - `invitee_recharge_total`：被邀请用户已完成且可返点的付费订单金额累计；包含余额充值订单和外部支付订阅订单，不包含手工余额兑换码或余额内扣订阅。
  - `affiliate_tiers`、`current_affiliate_tier`、`next_affiliate_tier`：后端计算后的阶梯数据。
  - `rules`：当前生效的全站运行规则（非用户进度），供门户/用户页展示文案，避免硬编码：
    - `rebate_freeze_hours`：返利冻结小时，`0` = 不冻结。
    - `rebate_duration_days`：邀请关系有效天数，`0` = 永久有效。
    - `rebate_per_invitee_cap`：单被邀人返利上限，`0` = 无上限。
    - `signup_bonus_enabled` / `signup_bonus_amount`：注册赠送开关与当前额度。
    取值走 `SettingService` 已有 getter（含默认值与 clamp），不另写解析。
- 管理后台设置页展示 L1-L4 四行，管理员只编辑门槛和返利比例。
- 用户页不再使用 demo 数值推算等级；只消费 API 返回的配置和进度。

## 已决策

- 没有 L0；未达 L1 或未配置任何等级时，实际返利比例返回 `null`，不再静默默认 20%。
- 邀请充值总额按 `payment_orders.status = 'COMPLETED'` 的 `amount` 汇总，仅包含 `order_type = 'balance'` 的余额充值订单和 `order_type = 'subscription'` 且 `payment_type <> 'balance'` 的外部支付订阅订单。
- 旧字段 `affiliate_rebate_rate` 保留作兼容字段，但阶梯返利的计算与展示以 `affiliate_rebate_tiers` 为准。
- 用户侧 `GET /user/aff` 用内嵌 `rules` 对象暴露冻结/有效期/单人上限与注册赠送当前值；`0` 语义与设置键默认值一致。不改计提/绑定逻辑，不改管理端，不改 schema。

## 待解决

- 无。

## 相关

- [[affiliate-signup-bonus]]
- [[payment]]
- 后端：`backend/internal/service/affiliate_service.go`、`backend/internal/service/affiliate_tiers.go`、`backend/internal/repository/affiliate_repo.go`
- 前端：`frontend/src/views/admin/SettingsView.vue`、`frontend/src/views/user/AffiliateView.vue`
