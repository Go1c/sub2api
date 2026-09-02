---
name: openai-hidden-luna-autoreview
description: 对用户隐藏 GPT-5.6 Luna；默认把 Auto-review / luna 入站改写到 gpt-5.6-terra，仅显式 Luna 键或 Auto-review→Luna 值可打真 Luna
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 隐藏 Luna，默认改写到 Terra，账号可 opt-in 真 Luna

简介：用户目录不展示 `gpt-5.6-luna` / `codex-auto-review`。默认把这类入站改写成 `gpt-5.6-terra`，避免 Auto-review 404。真 Luna 只在账号 `model_mapping` **显式 opt-in** 时发送：Luna **键**，或 Auto-review **键且映射值是 Luna**。身份白名单里的 `codex-auto-review` 不算 opt-in。opt-in 号全不可用时回退 Terra。

## 背景 / 目标

- Codex Auto-review 独立打 `/v1/responses`，模型固定 `gpt-5.6-luna`（或 `codex-auto-review`）。项目下拉框和公共配置改不了审批器模型。
- 账号组若没把 Luna 写进 `model_mapping`，网关会在选账号阶段回 404 `not supported by any configured account in this group`。
- 产品要求：普通用户默认不能访问真正的 Luna；Auto-review 不能因此失败，可以落到 Terra。
- 运维需要「只给某一个上游账号开 Luna」时，该号必须能接到真 Luna，而不是被全局改写挡死。

## 设计

- 默认改写：`RewriteOpenAIHiddenIngressModel` 把 `gpt-5.6-luna*` 与 `codex-auto-review*` → `gpt-5.6-terra`。
- 显式 opt-in（映射非空且满足其一）：
  - **键**是 Luna 名（归一化后含 `gpt-5.6-luna`），含 `Luna→Terra`（`gpt-5.6-luna` → `gpt-5.6-terra`）。该号仍会接到 Luna 流量，上游按映射值发送。
  - **键**是 Auto-review（`codex-auto-review` / `codex-auto-review-*`）且**值**是 Luna（如 `codex-auto-review` → `gpt-5.6-luna`）。
- 不算 opt-in：身份白名单 `{"codex-auto-review": "codex-auto-review"}`（模型白名单把允许名存成 identity mapping）、`{"codex-auto-review": "gpt-5.6-terra"}`、仅 Terra/Sol/gpt-5.6、空映射、`*`、`gpt-5.6-*`。空映射等于允许全部，不能当信号。
- `#330` 之后的 bug：`accountExplicitlyServesOpenAIHiddenLuna` 把任意 Auto-review **键**当成 opt-in。历史账号为避免 Auto-review 404 而把 `codex-auto-review` 加进白名单后，会静默接到真 Luna（调度只看 opt-in、不看 `IsModelSupported`）。
- `codex-auto-review*` 在 opt-in 路径归一成 `gpt-5.6-luna`，管理员只配 Luna **键**即可接住 Auto-review。
- 调度：先只在显式 Luna 号里选；全挂 / 被排除 / 渠道 restrict 拦截 Luna 时回退 Terra。
- 上游归一化最后一道闸：当前账号未 opt-in 时，即使请求体仍是 Luna 也改成 Terra。
- 用户目录不展示 Luna / Auto-review：`openai.DefaultModels`、前端密钥白名单、OpenCode 配置片段。管理端账号白名单和映射预设加回 Luna（另保留 `Luna→Terra`）。
- 计费：真 Luna 按 Luna 价卡；回退 Terra 按 Terra。

## 已决策

- 不单独探测 Auto-review 客户端。默认改写到 Terra；账号显式 Luna opt-in 除外。
- 不新增管理员设置。opt-in 沿用现有 `model_mapping` / 白名单，但不把白名单 identity Auto-review 当成 opt-in。
- 不把 Luna 偷偷改成 Sol；Terra 是默认公开 5.6 档。
- 映射值 `luna → terra` 仍合法：该号会接到 Luna 流量，但上游按映射送 Terra。

## 待解决

- 若以后要按分组配置 Auto-review 落点且不想用账号映射，再加设置项。

## 相关

- 网关 404 分类：`backend/internal/handler/no_account_error.go`
- 改写 / opt-in：`backend/internal/service/openai_model_alias.go`
- 选账号：`backend/internal/service/openai_account_scheduler.go`、`openai_gateway_scheduling.go`
- 入站改写请求体：`backend/internal/handler/openai_gateway_handler.go`
- 管理端白名单：`frontend/src/composables/useModelWhitelist.ts`
