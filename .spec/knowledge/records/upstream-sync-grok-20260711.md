---
name: upstream-sync-grok-20260711
description: Grok/xAI 上游功能链选择性同步记录——包含移植范围、fork 兼容决策、排除项、验证与 dev→publish 交付方式。
metadata:
  type: record
  date: 2026-07-11
  status: 完成
---

# Grok/xAI 上游同步记录 — 2026-07-11

## 目标与策略

将 fork `main` 中完整的 Grok/xAI 功能链按主题移植到 LumioAPI `dev`，保留 fork 已有的支付、订阅、OpenAI 网关和前端定制；不做上游整包 merge。实现和验证完成后，先走 topic PR 合入 `dev`，再用 `origin/dev` 的精确快照经 release PR promotion 到 `publish`。

## 已纳入范围

- Grok OAuth、token refresh、quota probe、账号调度与管理员接口。
- OpenAI CLI / Messages / Responses / `count_tokens` 兼容路由，以及 Grok 专用 WebSocket → HTTP/SSE bridge。
- WebSocket usage-limit 事件的账号 failover。
- Grok 图片/视频生成、composer image bridge、分组能力门禁、媒体计价配置与视频按秒计费。
- Grok 4.5 模型与 `reasoning_effort` 透传。
- 用户平台额度、计费/缓存/flush 链路及 migrations `900`–`905`。
- 管理端 Grok OAuth、账号与额度展示、分组媒体价格、模型白名单、平台标识和中英文文案。
- 与 fork 最新 `dev` 的契约适配、路由 feature guard 和测试兼容修复。

## 明确排除

- 非 Grok 且无编译/行为依赖的 upstream commits：`e2c731abe`、`7918b1a9`、`f93a6c50`。
- batch-image 子系统。
- 与本次目标无关的大型 gateway / i18n 重构。

这些排除项避免扩大 fork 热路径冲突面，也避免为后续 upstream 同步引入无关负担。

## fork 兼容决策

- 合并最新 `origin/dev` 时，Grok Responses 继续走专用 `forwardGrokResponses`；非 Grok 请求保留 fork 的 Codex compact reasoning-effort 规范化。
- 保留 fork 的支付、订阅额度池、站内信、账号错误历史和 LumioAPI UI 定制。
- migrations `900`–`905` 保持本批编号和内容；Ent/Wire 重新生成并纳入提交。
- TLS fingerprint 集成测试访问 `tls.peet.ws` 时，网络/TLS 错误和 response read 的 `EOF` / `unexpected EOF` 视为外部服务不可用并 skip；JSON 解析错误和指纹 mismatch 仍然失败，避免掩盖真实回归。

## 验证门禁

| 门禁 | 结果 |
|---|---|
| Backend unit：`go test -tags=unit ./...` | 通过 |
| Backend integration：`go test -tags=integration ./...` | 通过 |
| Backend vet：`go vet -tags integration ./...` | 通过 |
| Backend lint：golangci-lint v2.9 | 通过，0 issues |
| Frontend tests | 145 files / 795 tests 通过 |
| Frontend typecheck / build | 通过 |
| UI dev server + curl | Vite 启动成功，首页 HTTP 200 |
| `git diff --check` | 通过 |

## 交付约束

- 业务 topic PR 的 base 只能是 `dev`。
- `publish` 只接受从当前 `origin/dev` 精确快照创建的 `release/dev-to-publish-*` PR。
- release 分支不新增修复提交；发现问题必须先回到 `dev` 修复，再重新取 dev 快照。
- 本次不创建 release tag，除非另行要求。
