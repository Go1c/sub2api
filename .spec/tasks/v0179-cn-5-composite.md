---
name: v0179-cn-5-composite
description: 国模第五刀——Composite 分组路由
status: completed
---

# 任务：国模第五刀（Composite）

origin `ebc102877` Composite 分组路由 + 窗口 PR #5816/#5817/#6048/#5654。

fork 现在 `sanitizeGroupMessagesDispatchFields` 仍把非 openai 的 `AllowMessagesDispatch` 打成 false。

## 做什么

- 常量 `PlatformComposite`
- `composite_platform.go`（handler + service）+ 测试
- 分组多平台调度、网关按组成员平台分流
- `sanitizeGroupMessagesDispatchFields` 允许 composite 的 messages dispatch（#6048）
- Composite Codex（#5816）、CN 作为 composite 目标（#5817）、视频端点（#5654）
- SQL：fork **没有** `composite_model_routes`。origin `172` 建表 remap 为 **932**；把 origin `227` 的 kimi/zhipu/deepseek 直接写进 932 的 CHECK（不必再单开 933）。不要用 origin 编号 172/227。
- `ctxkey.ResolvedTargetPlatform`：fork `internal/pkg/ctxkey` 没有该键，按 origin 补最小键，不要整包搬 origin ctxkey。
- `sanitizeGroupMessagesDispatchFields`：openai **或 composite** 才保留 AllowMessagesDispatch（#6048）；CN 分组豁免已在 ②。
- #5875 catalog 收 composite + kimi/zhipu/deepseek

## 验收标准

- [x] composite 分组可含 openai/anthropic/gemini/grok/kimi/zhipu/deepseek 成员
- [x] 网关按成员平台分流；messages dispatch 不再被 sanitize 成 false
- [x] 迁移号 932+，不覆盖 930/931
- [x] unit + vet；前端 typecheck/build
- [x] 不碰 docker-compose、支付、publish

## 依赖

v0179-cn-1（CN 平台常量）；网关分流建议在 2 刀之后
