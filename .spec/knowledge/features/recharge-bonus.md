---
name: recharge-bonus
description: 充值满赠：管理员配置满赠档位（如充 1000 送 100）、订单快照、履约幂等发放赠送余额并流水溯源原订单、全额退款回收赠送——配置或排查赠送发放/流水时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 充值满赠（Recharge Bonus）

简介：管理员在支付设置配置满赠档位，用户余额充值支付成功履约时自动发放一笔赠送余额；赠送以独立兑换码流水记账并可溯源到原充值订单。

## 背景 / 目标

- 运营需要「充值 1000 送 100」类活动能力；上游 sub2api 只有余额倍率（multiplier），无档位满赠。
- 赠送必须：前端可见、支付成功后自动发放、流水可追溯到源充值订单、重试不重复发放、退款不可套利。

## 设计

### 配置

- 设置键：`RECHARGE_BONUS_ENABLED`（"true"/"false"）+ `RECHARGE_BONUS_TIERS`（JSON 数组 `[{"threshold":1000,"bonus":100},...]`，settings KV）。
- 管理端走 `PUT /api/v1/admin/settings` 的 `payment_recharge_bonus_enabled` / `payment_recharge_bonus_tiers`（也支持 `PUT /admin/payment/config`）；校验：threshold/bonus > 0、≤2 位小数、threshold 唯一、≤20 档。
- 读取统一经 `GetPaymentConfig` → `parseRechargeBonusTiers`（空/坏 JSON → 空 slice）→ `normalizeRechargeBonusTiers`（剔非法、同 threshold 留首个、**按 threshold 降序**）。不经过 `parseSettings`（payment 段直接从 PaymentConfigService 取）。

### 判定规则（已决策）

- **单档满赠、取满足条件的最高档**（不做「每满 X 送 Y」叠加）：`computeRechargeBonusAmount`。
- 判定基准 = **到账余额**（乘倍率后的 `order.Amount`；倍率 1 时即用户输入金额）。
- 仅 `order_type = balance` 的外部支付订单参与；余额购订阅不参与。

### 订单快照

- `payment_orders.bonus_amount`（decimal(20,2) DEFAULT 0，迁移 **948**）：`createOrderInTx` 创建时计算快照；履约与展示均用快照，改配置不影响已创建订单（与订阅快照同模式）。

### 履约幂等

- `doBalance`：主码 settle/redeem 后、两分支汇合处调 `fulfillRechargeBonus`（主码已用但赠送上次失败的重试可补发）→ affiliate 返利 → markCompleted（audit 加 `bonusAmount`）。
- 赠送锚定码 = `{recharge_code}-B`，type=`recharge_bonus`、notes=`payment_bonus:{recharge_code}`；与主码同构走 `resolveRedeemAction`（Create/Redeem/SkipCompleted），失败进 markFailed → 既有退避重试链路，天然不重复加钱。

### 入账与流水

- `redeem()` 新 case `recharge_bonus` → `userRepo.UpdateBalanceNoStat`：**只加 balance，不计 total_recharged**（不污染累计充值统计/签到门槛）。
- 流水即 `redeem_codes` 一条 status=used 记录：用户侧 `GET /redeem/history` 与管理端用户余额历史天然可见（notes 暴露条件、`SumPositiveBalanceByUser` 白名单均已加 `recharge_bonus`）。

### 退款回收（防套利）

- 用户 `RequestRefund` 余额校验改为 `amount + bonus_amount`；`prepDeduct` **全额退款**（refund >= amount）时 `BalanceToDeduct = min(refund + bonus, balance)`，部分退款不回收。否则「充值→退款」循环可白嫖赠送额。

### 前端

- 用户侧 `GET /payment/checkout-info` 新增 `recharge_bonus_enabled` / `recharge_bonus_tiers`（降序）；充值页金额卡顶部满赠 badge 行 + 到账预览「+ 赠送」badge 与「到账合计」行；结果页/订单表（OrderTable 组件）`bonus_amount > 0` 显示赠送标记。
- 管理端 SettingsView 支付设置「充值满赠」块：Toggle + 档位行编辑器（≤20 档，空行保存前剔除）；i18n 中/英/繁已补。

## 已决策

- 最高档命中而非叠加：最常见运营口径，实现与文案都最简单。
- 按到账余额判定：与订单记录、前端「到账」展示一致（倍率站点语义清晰）。
- 赠送走兑换码锚定（而非直接改余额）：复用履约幂等三件套（lease、锚定码、CAS）与流水可见性，零新表。
- 快照到订单：配置变更不影响进行中订单，可预测、可审计。
- 全额退款回收赠送：堵套利；部分退款不回收（口径简单、对用户友好）。

## 待解决

- 部分退款按比例回收赠送（当前不做，如运营需要再议）。

## 相关

- [[payment]]（履约链路）、[[daily-checkin]]（兑换码流水记账同模式）、[[affiliate-signup-bonus]]（余额历史框架）
- 后端：`payment_config_service.go`、`payment_order.go`（快照）、`payment_fulfillment.go`（`fulfillRechargeBonus`）、`redeem_service.go`、`payment_refund.go`、迁移 `948_recharge_bonus.sql`、测试 `recharge_bonus_test.go`
- 前端：`PaymentView.vue`、`PaymentResultView.vue`、`OrderTable.vue`、`SettingsView.vue`、`RedeemView.vue`、`UserBalanceHistoryModal.vue`
