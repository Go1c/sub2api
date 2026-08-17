---
name: openai-hidden-luna-autoreview
description: 对用户隐藏 GPT-5.6 Luna，并把 Codex Auto-review 的 luna/codex-auto-review 入站改写到 gpt-5.6-terra
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 隐藏 Luna，Auto-review 改写到 Terra

简介：用户不能选、也不能真正跑到 `gpt-5.6-luna`。Codex Auto-review 写死该模型且没有客户端覆盖字段，网关在选账号前把它和 `codex-auto-review` 改写成 `gpt-5.6-terra`，避免 404。

## 背景 / 目标

- Codex Auto-review 独立打 `/v1/responses`，模型固定 `gpt-5.6-luna`（或 `codex-auto-review`）。项目下拉框和公共配置改不了审批器模型。
- 账号组若没把 Luna 写进 `model_mapping`，网关会在选账号阶段回 404 `not supported by any configured account in this group`。
- 产品要求：普通用户不能访问真正的 Luna；Auto-review 不能因此失败，可以落到 Terra。

## 设计

- 入站改写函数 `RewriteOpenAIHiddenIngressModel`：`gpt-5.6-luna*` 与 `codex-auto-review*` → `gpt-5.6-terra`。
- 选账号、可用性诊断、OAuth/API Key 上游归一化、`/v1/responses` / chat / WS 请求体都走同一套改写，避免渠道透传仍把 Luna 送上去。
- 用户目录不展示 Luna / Auto-review：`openai.DefaultModels`、前端密钥白名单、OpenCode 配置片段。
- 计费按改写后的 Terra 走；Luna 价卡仍留在定价表里，但不作为可调度模型。

## 已决策

- 不单独探测 Auto-review 客户端：Codex 公共配置没有审批器模型字段，审批器与用户请求身份难以可靠区分。全部 Luna 入站改写到 Terra，用户也拿不到真 Luna。
- 不新增管理员设置。目标固定 `gpt-5.6-terra`。若该组账号也不支持 Terra，仍会 404。
- 不把 Luna 偷偷改成 Sol；Terra 是当前账户组实际在用的公开 5.6 档。

## 待解决

- 若以后要按分组配置 Auto-review 落点，再加设置项；当前不需要。

## 相关

- 网关 404 分类：`backend/internal/handler/no_account_error.go`
- 改写：`backend/internal/service/openai_model_alias.go`
- 选账号：`backend/internal/service/openai_account_scheduler.go`、`openai_gateway_scheduling.go`
- 入站改写请求体：`backend/internal/handler/openai_gateway_handler.go`
