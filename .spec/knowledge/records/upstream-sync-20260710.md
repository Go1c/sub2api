---
name: upstream-sync-20260710
description: upstream v0.1.150 日更同步台账——记录 18 个主线落点在 LumioAPI fork 中的选择性适配、例外与验证结果。
metadata:
  type: record
  date: 2026-07-10
  status: 归档
---

# Upstream Sync 台账 — 2026-07-10

## 范围与结论

- 基线：`v0.1.149`（`12d811bd76572836d6df6e1fa8aa5ff91be3b12e`）。
- 范围：`upstream/main` 于 2026-07-10（Asia/Singapore）落入主线的 18 个 first-parent 落点。
- 方法：按 T1 OpenAI 兼容、T2 计费与恢复、T3 管理端与设置选择性移植；保留 fork 的 Codex、图像、订阅、支付和前端定制，不整包合并 upstream。
- 结果：17 个落点已直接移植、等价覆盖或按 fork 布局适配；Grok 落点完成适用性审计，但当前 fork 缺少对应网关基础，留给独立 Grok 同步任务处理。版本更新至 `0.1.150`。

## 18 个上游落点

| upstream | 主题 | fork 处理结果 |
|---|---|---|
| `309936207` | compact SSE raw output item preservation | 已适配 compact raw output、unary JSON→SSE bridge 与失败终止事件；保留 fork 流式处理结构。 |
| `066a85949` | Codex client version bump | fork 已有等价且更新的 Codex client version/header 实现，无需重复移植。 |
| `72ef0b388` | GPT-5.6 compact max reasoning | 已适配 GPT-5.6 max reasoning 并补聚焦测试。 |
| `9389503c5` | English admin i18n keys | 已适配到 fork 的单文件 locale，并补齐 zh / zh-Hant 对应文案。 |
| `cddcf4904` | parallel tool calls compatibility | 已移植并按 fork apicompat 测试契约调整。 |
| `301c99a26` | reasoning effort model candidates | 已移植 reasoning model candidates。 |
| `ddb1a210c` | concurrency and recovery audit | 已移植 payment/redeem 原子恢复、subscription CAS 重校验；保留 fork 的 lazy window activation 与余额回退语义。 |
| `312ab1f0b` | WebSocket passthrough reasoning candidates | 已移植 WS passthrough reasoning metadata/candidates。 |
| `5c9748cef` | UserBreakdown request type filter | 已移植后端过滤、请求类型契约与测试。 |
| `8b96acde9` | GPT-5.6 cache billing stats | 已移植 cache billing/pricing 与统计测试。 |
| `0dec1ad29` | consolidate GPT-5.6 helper | 已统一到 fork 的 `openai_model_alias.go`。 |
| `9a2f11b4e` | VERSION 0.1.150 | 已更新 `backend/cmd/server/VERSION`。 |
| `6a2063b37` | setup-token auto refresh | 已移植 token refresh 与测试。 |
| `d0f6f27d5` | image_gen namespace handling | 已按 fork 图像工具意图识别与隐藏路由布局适配。 |
| `6dd3274aa` | GPT-5.6 billing usage integrity | 已移植 usage integrity，保留 fork 计费三桶语义。 |
| `815516d8d` | Grok responses reasoning effort | 已审计；当前 daily sync 的 fork 网关基础不适用，交由独立 Grok 同步任务处理，未生搬上游代码。 |
| `5260a42a0` | user-scoped Fast/Flex policy | 已移植后端策略、admin settings API/UI 与多语言文案。 |
| `deff3123d` | Codex identity pairing | 已按 fork Codex identity/header 管线适配并补测试。 |

## Fork 适配要点

- compact 请求既支持原始 SSE 输出，也支持 unary compact JSON 桥接为 SSE；异常结束时合成 `response.failed`，避免客户端悬挂。
- GPT-5.6 计费保留 input/cache read/cache write 三桶完整性，不覆盖 fork 已有显式价格与模型别名逻辑。
- payment/redeem/subscription 恢复逻辑保持原子与幂等，同时保留 fork 的订阅窗口延迟激活、余额兜底和月窗缺失兼容。
- public settings 加入并发请求合并，Fast/Flex 政策扩展到用户级，管理端前后端契约同步。
- Codex WebSocket、identity、image namespace 与 Windows reset 处理均按 fork 现有网关分层落位。

## 验证

以下检查在同步分支最终代码上通过：

- `go test -tags=unit ./...`
- `go test -tags=integration ./...`
- `go vet -tags integration ./...`
- CI 同版本 golangci-lint v2.9：`0 issues`
- `pnpm typecheck`
- `pnpm build`
- `src/composables/__tests__/useModelWhitelist.spec.ts`
- `src/stores/__tests__/app.spec.ts`
- OpenAI transport/identity/reasoning、subscription/recovery、middleware 等聚焦测试
- `git diff --check`

## 发布纪律

同步内容必须先经 PR 合入 `dev`；随后从远端最新 `dev` 的精确快照创建 `release/dev-to-publish-20260711`，再经 PR promotion 到受保护的 `publish`。不得直接推送 `publish`，不得在 release 分支补提交。
