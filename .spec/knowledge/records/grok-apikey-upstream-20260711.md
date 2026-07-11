---
name: grok-apikey-upstream-20260711
description: Grok 账号 type=apikey（URL + API Key）上游透传方案——动机、实现边界、与 youxuanxue 差异、getezo 实测与验证门禁。
metadata:
  type: record
  date: 2026-07-11
  status: 完成
---

# Grok API Key / 上游透传 — 2026-07-11

## 目标

在已有 Grok OAuth 之外，支持管理端用 **Base URL + API Key** 接入 OpenAI 兼容上游（官方 `api.x.ai` 或第三方如 getezo），经 sub2api 转发 Responses / Chat Completions / Media。

## 背景

- 上游 `Wei-Shaw/sub2api` 与多数 fork 的 Grok 仅 OAuth；管理端创建账号时没有 URL+Key 入口。
- fork `youxuanxue/sub2api` 有 `IsGrokAPIKey` + edge/TokenKey 中继形态，与本仓库网关架构不匹配。
- 本方案采用**原生 `AccountTypeAPIKey`**，不引入 edge 中继栈。

## 实现要点

| 层 | 行为 |
|----|------|
| 账号 | `IsGrokAPIKey()`、`GetGrokApiKey()`；凭证字段 `api_key` + `base_url` |
| Responses | `forwardGrokResponses` 接受 oauth \| apikey；`resolveGrokResponsesURL`：apikey 走 `buildOpenAIResponsesURL`，oauth 仍走 `xai.BuildResponsesURL` allowlist |
| Chat | `rawChatCompletionsURL` 对 apikey 走 OpenAI 兼容拼接 |
| Media | `upstreamURLForAccount` 对 apikey 走 `buildOpenAIEndpointURL` |
| Token | `GetAccessToken` 对 Grok apikey 读 `credentials.api_key` |
| 前端 | Create/Edit 支持 Grok OAuth \| API Key；默认 base `https://api.x.ai/v1` |

OAuth 路径保持 xAI host allowlist；仅 apikey 放开自定义 host，避免放宽官方 OAuth 安全面。

## 与 youxuanxue 的差异

- **不做** TokenKey / edge relay 专用链路。
- 复用现有 OpenAI 兼容 URL 组装与网关转发，改动面更小，便于后续 upstream 同步。

## 实测（第三方上游，密钥不入库）

针对 OpenAI 兼容上游（示例 host 仅作测试，不写生产密钥）：

| 场景 | 结果 |
|------|------|
| `GET /v1/models` | 含 `grok-4.5`、`grok-4.3` 等 |
| `POST /v1/responses` `model=grok-4.5` 非流式 / 流式 | HTTP 200；响应 model 可能回写为 `grok-4.5-build` |
| `POST /v1/chat/completions` `grok-4.5` 非流式 / 流式 | HTTP 200；流式含 `reasoning_content` + `[DONE]` |
| 裸名 `model=grok` 直连上游 | 可能 503（无 channel）；网关侧别名映射到 `grok-4.5` 后转发 |

**运维建议**：账号侧默认/映射模型写 **`grok-4.5`**；客户端直连第三方上游时不要依赖裸 `grok`。

## 验证门禁（本 topic）

| 门禁 | 结果 |
|------|------|
| Backend：`go test ./internal/service/ -run 'Grok\|APIKey\|...'` | 通过 |
| Frontend：`vue-tsc --noEmit` | 通过 |
| 第三方上游 `grok-4.5` 直连 smoke | 通过（responses / chat / stream） |
| 密钥进仓检查 | 仅测试占位 `sk-test` / `sk-upstream`，无生产密钥 |

## 交付约束

- 业务 topic PR base：`dev`。
- 不把第三方 API Key 写入仓库、知识库或 commit message。
- `publish` 仍须经 `release/dev-to-publish-*` 从 `origin/dev` 快照 promotion。
