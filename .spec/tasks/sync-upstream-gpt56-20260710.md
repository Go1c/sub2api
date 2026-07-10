---
status: completed
---

# Sync Upstream GPT-5.6 — 2026-07-10

## Task

将 Wei-Shaw/sub2api PR #3905、#3898、#3794 与 #3800 作为独立 topic 同步到 fork 的最新 `dev` 基线，保留本仓库今天已落地的 Codex headers 与 image tool hotfix。

## Scope

- 吸收 PR #3905 commit `657c4f97d904d796c806c7bd612ac5929307dcaf`，统一 Codex 生产常量与测试期望为 `0.144.1`。
- 按顺序吸收 PR #3898 commits：
  - `4a2b10c94e91c275a43b61ee68b6fe89855d1953`
  - `383f61d0e9aa737e33bf2201b47228338fe55fa7`
  - `062af81fb5964c78f64ee31195bed88ab3e858ad`
- 保留 GPT-5.6 缓存写入计费的协议转换、Responses / Chat Completions / WebSocket、usage 三桶拆分及 Sol / Terra / Luna 官方定价改动。
- 与本仓库 GPT-5.4 fallback 冲突时做最小语义合并。
- 吸收 PR #3794 commit `a02700c16f89e26d49a6c1cc4442432349b137fd`，识别 `namespace: image_gen` 与 `input.additional_tools` 生图意图并参与账号能力调度，不删除 GPT-5.6 `tool_search` schema 所需 namespace。
- 吸收 PR #3800 commit `13e773ef5e7908b0af0f2938295775b38a26eaaa`，支持 Codex manifest 透传与 `/v1/models?client_version=...` 动态模型清单；默认 client version 使用 `0.144.1`，普通 `/v1/models` 保持静态格式。

## Acceptance

- Codex 四个生产版本常量及相关测试期望一致为 `0.144.1`，不丢现有 headers 修复。
- GPT-5.6 cached input 与 cache write usage 在各协议链路正确拆分、计费。
- Sol / Terra / Luna 使用同步后的官方定价。
- image generation intent 能识别 namespace 与 additional tools，同时不回归 hosted/local image tool 冲突修复。
- Codex 模型清单可按 client version 从上游透传，静态模型列表及已有 GPT-5.6 三模型映射不回归。
- 聚焦测试与仓库后端校验通过，或准确记录环境阻断。
- `gofmt` 与 `git diff --check` 通过。
- topic 分支已推送，不创建或合并 PR。

## Knowledge

纯 upstream topic 同步，不新增 knowledge 文档。

## Verification Results

- `gofmt`：所有改动 Go 文件已格式化。
- `git diff --check`：通过。
- 聚焦测试：`internal/pkg/apicompat`、`internal/service/openai_ws_v2`、`internal/server/routes` 通过；`internal/service` 已完成编译，但 manifest 的 `httptest.NewServer` 因沙箱禁止监听本地端口而失败（`listen tcp6 [::1]:0: bind: operation not permitted`），不是代码断言失败。
- 按线上紧急发布指令停止继续验证；未运行完整 unit、integration、vet、golangci-lint，交由 GitHub CI 后续校验。
- #3781 审计：`2a3dcb499`、`9b75c7b76`、`fa01aec80` 不在当前 `dev`；其语义仅调整空映射 OpenAI OAuth 对异族模型的 scheduler 过滤，明确不改变 API Key 语义，本 topic 未额外吸收。
- 静态 `DefaultModels` 与 `codexModelMap` 已确认包含 `gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna`，未改数据库账号显式 `model_mapping`。
