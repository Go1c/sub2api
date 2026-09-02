---
status: completed
---

# 任务：国模第三刀（前端账号 / 用量）

## 做什么

从 origin/main HEAD 移植并适配 fork 的 Create/Edit 弹窗（fork 弹窗已有 grok/oauth/header-override，不要整文件覆盖）。

Fork 插入点（开工时再核对）：

- `CreateAccountModal.vue` 平台 segmented control 现有 anthropic/openai/gemini/antigravity/grok，在 grok 后加 kimi/zhipu/deepseek
- `credentialsBuilder.ts`：`HEADER_OVERRIDE_PLATFORMS` 现为 anthropic/openai/grok，加 CN（#5847）
- `AccountUsageCell.vue`：在 grok/openai 分支旁挂 `CNProviderQuotaCell` / `CNProviderBalanceCell`
- `frontend/src/types/index.ts`：`AccountPlatform` 扩 `'kimi' | 'zhipu' | 'deepseek'`
- origin `CN_BASE_URL_PRESETS` / `defaultCNBaseUrl` / `isCNCodingPlan` / `isCNPaygBalance` 在 credentialsBuilder
- origin 组件：`CnBaseUrlPresets.vue`、`CNProviderBalanceCell.vue`、`CNProviderQuotaCell.vue` 用 HEAD（含 #5837 布局、#6009 刷新）

- `CreateAccountModal.vue` / `EditAccountModal.vue`：account_mode + api_protocol + base_url 预设（含 #5842 adaptive）
- `credentialsBuilder.ts`：CN 预设、header override 含 CN（#5847）
- `CnBaseUrlPresets.vue`
- `CNProviderBalanceCell.vue` / `CNProviderQuotaCell.vue`（含 #5837 布局、#6009 刷新入口）及 spec
- `AccountUsageCell.vue` 挂 CN 单元格
- `PlatformIcon.vue` / `PlatformTypeBadge.vue` / `platformColors.ts` / i18n accounts
- 类型 `AccountPlatform` 扩 kimi/zhipu/deepseek

## 验收标准

- [x] 创建 kimi/zhipu/deepseek 账号可选 payg/coding 与 chat_completions/anthropic/responses/adaptive
- [x] 选平台后 base_url 填官方预设
- [x] 用量列：coding 显示 5h/weekly，payg 显示余额
- [x] `pnpm typecheck && pnpm build`；相关 vitest 绿
- [x] 不碰 docker-compose、支付、publish

## 依赖

v0179-cn-1-constants-schema-probe（API `cnProviders.ts`）
