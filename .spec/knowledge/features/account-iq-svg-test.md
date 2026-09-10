---
name: account-iq-svg-test
description: 管理端 OpenAI/Codex 账号智商检测：自定义 SVG 提示词走测试连接 SSE，前端抽出完整 SVG 渲染成图
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 账号智商检测（SVG 成图）

管理员在 OpenAI/Codex 账号「更多」菜单里发一条自定义提示词（默认 pelican 骑自行车），用现有账号测试 SSE 拿模型回复，从正文抽出完整 `<svg>`，渲染成图片卡片。用来肉眼看这个号的模型能不能画出像样的 SVG，不是网关计费功能。

## 背景 / 目标

- 测试连接只发 `"hi"`，看不出模型作图能力。
- Codex 可能先思考再吐 SVG，正文里还会夹 markdown 围栏或前言；弹窗要把**最终 SVG 当成一张图**，而不是代码块。
- 逻辑必须独立成文件、只留一个入口，避免以后同步 upstream 时在 `account_test_service.go` / handler 上大面积冲突。

## 设计

- **交互面**：仅 `platform === 'openai'`（OAuth / API Key / 影子）显示「智商检测」。弹窗上半是模型 + 可编辑提示词，下半是终端原文；抽出完整 SVG 后在终端下方用 `<img data:image/svg+xml>` 预览。排除 `gpt-image-*`（那条路是生图 API）。
- **实现面（后端）**：继续 `POST /admin/accounts/:id/test`，**不改 handler**。弹窗传 `mode: "iq"`（`normalizeAccountTestMode` 会把它当成 default，不会走进 compact）。`TestAccountConnection` 必须在 normalize **之前**记住原始 mode，否则公开入口会先把 `iq` 收成 `default`，payload 仍发 `"hi"`，Astra 就会回 `Hi! What would you like to work on?`。`account_test_service.go` 只留三处钩子：记住原始 mode、Responses payload、Chat Completions payload。真正改 payload 的唯一入口是 `ApplyOpenAIAccountIQTestPayload`（`backend/internal/service/account_iq.go`）：非 iq 原样返回；iq 时覆盖 prompt、Responses 写 `reasoning.effort=high`（OAuth 补 `include`）、Chat Completions 写 `reasoning_effort=high`。回归必须走 `TestAccountConnection`，只测内层 `testOpenAIAccountConnection` 测不出这次丢 mode。
- **实现面（前端）**：抽取与请求体在 `frontend/src/components/admin/account/iqTest.ts`。弹窗只调 `buildIqTestRequest` 和 `applyIqTestOutput`。完整 `<svg>…</svg>`（优先围栏）经 `sanitizeSvg` 做成 data URL。Codex reasoning 事件本来就不会进测试 SSE 的 `content`。

## 已决策

- 提示词用户可改，默认 `Generate an SVG of a pelican riding a bicycle`。
- 默认模型优先 `gpt-6-astra`（没有则 `gpt-6` / `gpt-5.4`），思考强度 `high`。
- 成图用 `<img>` + data URL，不用 `v-html`，避免把模型输出当 HTML 执行。
- 不改 `createOpenAITestPayload` 签名，避免误伤用量探测和国模 adaptive。
- 用已有 `mode` 字段表达 iq，不加 `reasoning_effort` JSON 字段，handler 保持与 upstream 同形。

## 待解决

- 生成复杂 SVG 可能要一两分钟；沿用测试连接的 HTTP/代理超时，没有单独放宽。

## 相关

- 测试连接：`frontend/src/components/admin/account/AccountTestModal.vue`、`backend/internal/service/account_test_service.go`
- 弹窗：`frontend/src/components/admin/account/AccountIqTestModal.vue`
- 入口：`backend/internal/service/account_iq.go`、`frontend/src/components/admin/account/iqTest.ts`
