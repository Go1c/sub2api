---
name: upstream-sync-v0200-astra-codex
description: 0.2.0 窗口后选择性带入 GPT-6 Astra 与 Codex 透传/bootstrap/聊天缓存身份；不整包 merge 0.2.0。
metadata:
  type: record
  date: 2026-09-05
  status: 已实现
---

# Upstream Sync 台账 — 2026-09-05 (Astra + Codex)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | 已与 `upstream/main` 快进对齐 `b1748c4ea`（0.2.0 窗口） |
| 同步方式 | merge-upstream 快进 `main`；**不**整包 merge 进 `dev` |
| 主题 | GPT-6 Astra 目录/计价 + Codex 相关 bugfix |
| **禁止** | 整包 `git merge main` → `dev`；直推 `main` / `publish` |

## 带入

| 来源 | 内容 | 适配 |
|---|---|---|
| 上游 PR [#6572](https://github.com/Wei-Shaw/sub2api/pull/6572)（未合并，作参考） | `gpt-6-astra` + 公开别名 `gpt-6`、官方价卡、白名单、OpenCode、`reasoning.max` | 隐藏 Luna；不用上游 Codex descriptor / `openAIModelFastPricingRatio`（本 fork 无对应实现）；Fast 走 `*Priority`；目录缺长上下文字段时钉 272K / 1.25x cache write |
| `#6458` / `#4936` | scheduler 快照保留 `openai_passthrough`；passthrough 账号 `IsModelSupported` 短路 | 按 fork 调度缓存改 |
| `#6470` | WS 透传：优雅关闭但尚无 terminal 视为失败 | `passthrough_relay.go` |
| `#6450` | Codex 定时自动化 bootstrap（无 `call_id`） | `handler/openai_codex_bootstrap.go`，在 channel mapping 前 |
| `#6469` | API Key 聊天补全自动派生并按租户隔离 `prompt_cache_key` | 用 fork 的三参 `isolateOpenAISessionID`；Responses 形状不自动派生 |

## 明确排除

- `#6469` 的 Agent Identity 重试用例依赖上游 `isolateOpenAIUpstreamSessionID`，未搬。
- 上游 `#6572` 的 Codex models descriptor / Fast ratio 表 / LiteLLM `above_272k` 解析。
- 0.2.0 窗口其余冲突提交（kimi、fable、group openai-fast、VERSION/sponsors 等）。

## 验证

- `go test -tags=unit`：`openai` / `service` / `handler` / `repository` / `openai_ws_v2`
- 前端 `pnpm typecheck` + `pnpm build`；白名单 / OpenCode 单测
