---
status: in_progress
---

# Claude Fable 5.1 广场展示覆盖：按钮 + 接口两条清法

## Task

公开卡 `claude-fable-5-1` 被 `selected_models.pricing` 冻住（可出现输入/输出 `-`、缓存仍 $0.3/$3.75）。后台表单没有缓存两格。

两条链路：
1. 候选行「恢复自动价格」→ 保存模型广场
2. 运维 `GET/PUT /api/v1/admin/settings/model-market`，从 `selected_models` 删除 `anthropic:claude-fable-5-1`

## Scope

- `frontend/src/utils/modelMarketDisplayOverride.ts` 及测试
- `frontend/src/views/admin/SettingsView.vue` 候选行按钮
- i18n zh / en / zh-Hant
- `.spec/knowledge/features/model-market.md`

## Acceptance

- [x] 有覆盖的候选行显示「恢复自动价格」；自动同步下点按钮后保存不再带上该 key
- [x] 手动模式下保留勾选/排序，只去掉 billing_mode 与 pricing
- [x] 只清空输入输出仍不够；必须整条覆盖
- [x] 不改 BillingService
