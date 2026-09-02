---
name: recharge-invoice-balance-gate
description: 充值页新增开票须知提示，并优化首次使用余额门槛的最低充值提示文案。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 充值开票须知与余额使用门槛文案
简介：在充值页的账户卡片上新增"开票须知"提示，并把仅获得赠送余额用户首次使用时的"最低历史充值"门槛错误信息改写得更清楚（说明防滥用原因、支持无理由退款）。

## 背景 / 目标
- 让充值页清晰展示开票须知（价格不含税、最低税点、税点可从余额扣除）。
- 改进首次使用余额门槛的提示：只获得赠送余额的用户尝试使用时，门槛提示应解释反滥用原因和无理由退款支持。
- 现有支付与计费行为保持不变。

## 设计
前端 `frontend/src/views/user/PaymentView.vue`：账户卡片保留左侧账户摘要、右侧绿色规则面板。右侧改为竖向堆叠的两条紧凑提示：
- 充值规则：沿用现有文案（`payment.rechargeExchangeRule`）。
- 开票须知（新增 `payment.invoiceNotice`）：`开票须知：此价格不含税，最低 1% 税点，税点可以选择从账户余额扣除。`

堆叠沿用当前白卡布局，使用响应式换行（`w-full space-y-2 sm:w-auto sm:max-w-xl`），小屏下不挤压账户余额显示。新增 i18n key 加在 `frontend/src/i18n/locales/zh.ts`。

后端 `backend/internal/service/billing_cache_service.go`：仅改 `minRecharge` 分支。错误码仍为 `BALANCE_USAGE_GATE_NOT_MET`，错误信息改为：

```go
fmt.Sprintf("为防止批量注册，本站最低充值 %.2f 元才可以正常使用赠送余额。您可以放心，充值余额支持无理由退款。", minRecharge)
```

## 已决策
- `minBalance` 分支保持不变，因为本次要求的文案专门针对"首次使用历史充值"门槛。
- 错误码不变（`BALANCE_USAGE_GATE_NOT_MET`），仅改人类可读消息，保持 API 兼容。
- UI 改动局限在 `PaymentView.vue` 及其 i18n key，后端行为除上述消息外不变，以减小后续 upstream 同步负担。

## 实现
Task 1（前端开票须知）：在 `PaymentView.spec.ts` 先写失败测试（断言渲染 `payment.invoiceNotice`），把单一充值规则面板改为右侧竖向堆叠两条提示，加 `zh.ts` 文案，跑测试通过。

Task 2（后端余额门槛文案）：更新 `TestBillingCacheServiceCheckBillingEligibility_BalanceUsageGateRequiresRecharge` 断言，应包含"为防止批量注册""最低充值 5.00 元""正常使用赠送余额""无理由退款"，且不包含"当前历史充值"和"0.00"。改 `billing_cache_service.go` 的消息后测试通过。

Task 3（验证）：
- 前端：`cd frontend && npm run test:run -- src/views/user/__tests__/PaymentView.spec.ts --runInBand`
- 后端：`cd backend && GOCACHE=/tmp/sub2api-go-build-cache GOMODCACHE=/tmp/sub2api-go-mod-cache go test ./internal/service -run BalanceUsageGate -count=1`

## 相关
- [[payment]]
- 前端：`frontend/src/views/user/PaymentView.vue`、`frontend/src/views/user/__tests__/PaymentView.spec.ts`、`frontend/src/i18n/locales/zh.ts`
- 后端：`backend/internal/service/billing_cache_service.go`、`backend/internal/service/billing_cache_service_test.go`
- 技术栈：Vue 3、Vue i18n、Vitest、Go、testify
