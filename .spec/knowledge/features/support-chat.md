---
name: support-chat
description: 站内 AI 客服浮窗——接入外部 gateway、登录态请求修复、品牌配色刷新、附件上传需求时查这篇。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# AI 客服（Support Chat）

简介：站内 AI 客服聊天浮窗。`sub2api` 只保留接入层（前端浮窗 + 公开配置读取），真正的 AI 客服网关在独立仓库 `lumio-ai-support-chat`。本篇汇总：DocsGPT 支持代理的架构边界、登录态请求 400 修复与品牌配色刷新（已落地），以及附件上传能力需求（设计中，待网关与前端实现）。

## 背景 / 目标

用户希望在站内与 AI 客服对话获取帮助；并能上传图片或文件让 AI 结合附件内容回答。同时需修复登录用户聊天 400 失败，并让浮窗配色与首页品牌系统一致。

### 架构边界（DocsGPT Support Agent）

AI 客服的独立服务代码和完整 DocsGPT 部署文档已迁移到独立仓库：

```text
https://github.com/Go1c/lumio-ai-support-chat
```

当前 `sub2api` 项目只保留接入层：

- LumioAPI 后端保存公开的 AI 客服开关、gateway URL 和展示文案。
- 管理员后台提供「站点设置 → AI 客服」配置入口。
- `frontend` 读取公开设置并连接外部 support gateway。

部署或维护 DocsGPT、Agent API key、CORS、限流和 SSE 转发时，在 `lumio-ai-support-chat` 仓库中处理。**绝不**把模型 API key、DocsGPT Agent API key 或对象存储密钥暴露给浏览器。

## 设计

### 网关请求修复（登录态 400）

**根因**：support gateway 接受匿名请求和字符串用户标识，但当 `user.id` 以数字发送时拒绝登录用户请求。旧前端直接转发 `auth.user.id`，导致登录用户命中 `POST /chat/stream -> 400 invalid json body`，浮窗回退到「不可用」状态。

**证据**：
- `GET /widget-config?locale=zh-CN` 返回 `200`。
- `POST /chat/stream` 带 `{ "message": "hello", "locale": "en-US" }` 返回 `200`。
- 同请求带 `user: { "id": 123, ... }` 返回 `400 invalid json body`；将 `user.id` 改为 `"123"` 返回 `200`。

**修复**：在前端 support-chat API 边界把 `user.id` 序列化为字符串（保持其余 payload 形状不变），这是最小安全改动，不触碰应用 auth 类型，也不动 `lumio-ai-support-chat` 网关仓库。匿名聊天行为、locale 映射、会话处理保持不变。

```ts
function normalizeSupportChatRequest(request: SupportChatRequest): SupportChatRequest {
  if (!request.user || request.user.id === undefined || request.user.id === null) return request
  return { ...request, user: { ...request.user, id: String(request.user.id) } }
}
// streamSupportChat 中：body: JSON.stringify(normalizeSupportChatRequest(request))
```

### 品牌配色刷新

把浮窗的交互强调色从旧的 teal primary 切换到首页的蓝-靛-紫品牌系统（`HomeView.vue` 已使用），保留白色/玻璃态会话面以保证可读性：

- 切换气泡（toggle）：蓝-靛-紫渐变 + 冷色辉光
- 发送按钮：同渐变族保持视觉一致
- 用户消息气泡：渐变强调色取代纯 teal，例如 `bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 text-white shadow-[0_12px_28px_rgba(99,102,241,0.28)]`
- 来源 pill 与 focus 状态：靛/紫色调表面
- 支持链接 hover/focus：蓝-紫品牌强调色

UX 约束：不降低助手消息与正文对比度；保持移动端布局与浮窗宽度不变；保留现有重试/错误流程。

**涉及文件**：
- `frontend/src/api/supportChat.ts` / `frontend/src/api/supportChat.spec.ts`
- `frontend/src/components/support/SupportChatWidget.vue` / `frontend/src/components/support/SupportChatWidget.spec.ts`

**测试策略**：API 级单测证明数字 `user.id` 以字符串发送；widget spec 断言登录请求带字符串 ID；UI 断言 toggle/send 按钮的新品牌类；运行定向 Vitest、`vue-tsc --noEmit` 与生产构建。

### 附件上传能力（需求，设计中）

让用户在客服会话中上传图片或文件给 AI 看。当前 `lumio-ai-support-chat` 只有 `POST /chat/stream` 文本流接口，入参 `{ message, conversationId?, locale?, user? }`，请求体 `32KB` JSON 限制，无 multipart / 附件上传 / 文件存储 / 附件转发协议。

**非目标**：不做管理员知识库上传；不做长期文件管理后台；不向浏览器暴露任何密钥。

**支持范围（首版）**：
- 图片 `image/png`、`image/jpeg`、`image/webp`；文档 `text/plain`、`text/markdown`、`application/pdf`。
- 单次最多 3 个附件，单文件最大 10MB，总大小最大 20MB。
- 不支持的类型在前端和网关都拒绝并返回清晰错误。

**接口方案**：
- **方案 A（推荐，首版采用）**——单接口 multipart 流式发送。扩展 `POST /chat/stream` 支持 `Content-Type: multipart/form-data` + `Accept: text/event-stream`，字段 `message`（必填）、`conversationId`、`locale`、`user`（JSON 字符串）、`attachments`（重复文件字段 1-3 个）。响应继续用现有 SSE 事件 `answer` / `id` / `error` / `end`。前端只需把文本 JSON 请求切为 `FormData`，无需先上传再发消息。改动少、保留流式体验、无需持久保存附件。
- **方案 B**——两步上传。`POST /attachments`（multipart）返回 `{ attachments: [{ id, name, mimeType, size }] }`，再 `POST /chat/stream`（JSON）带 `attachments: ["att_..."]`。更适合未来对象存储、病毒扫描、附件复用与严格审计，但首版成本更高。

## 已决策

- 客服网关代码、DocsGPT 部署、密钥/CORS/限流/SSE 转发全部归 `lumio-ai-support-chat` 仓库；`sub2api` 只做接入层。
- 登录态 400 修复采用「前端边界把 user.id 转字符串」最小改动，不改网关仓库与 auth 类型。
- 浮窗交互色统一到首页蓝-靛-紫品牌系统。
- 附件首版采用方案 A（单接口 multipart 流式）。

## 待解决

附件能力尚未实现，按以下计划落地。

### 实现：网关侧（`lumio-ai-support-chat`）

- 保持原 JSON `POST /chat/stream` 兼容，未传附件时现有前端不受影响。
- multipart 请求限制总读取大小，避免内存被大文件打满。
- 校验文件数量、大小、扩展名和 MIME 类型；不信任浏览器传入的 MIME，至少用文件头做基础检测。
- 图片附件转为上游模型可理解的图片输入或 DocsGPT 支持的格式；PDF/文本提取文本后追加到本轮问题上下文，提取失败返回 SSE `error`。
- 不在 SSE 或日志输出附件原文、base64、密钥或用户敏感信息；附件只用于当前会话回合，临时文件响应结束后清理。
- CORS `Access-Control-Allow-Headers` 继续允许 `Content-Type` 和 `Accept`；限流按客户端 IP 生效，附件请求计入。

### 实现：前端侧（`SupportChatWidget.vue` + `supportChat.ts`）

- 输入框左侧增加附件按钮（用现有 `Icon` 的 `upload` 图标）；选择后在输入框上方展示文件名、大小和移除按钮。
- 前端执行与网关一致的数量/大小/类型校验。
- 有附件时用 `FormData` 调 `/chat/stream`，无附件时继续用现有 JSON 请求；发送中禁用附件与移除按钮。
- 流式回答、失败重试、会话 ID 行为不变；失败重试保留上一次文本和附件，除非用户手动清除。

### 验收标准

- 用户上传一张截图问「这个报错是什么意思」，AI 能结合截图回答。
- 上传不支持的类型时前端立即提示，网关也拒绝绕过前端的请求。
- 浏览器网络面板看不到 DocsGPT Agent API key、模型 API key 或对象存储密钥。
- 现有无附件聊天不回归。

### 测试要求

- 网关：JSON 文本请求保持兼容；multipart 能接收文本和图片附件；超出数量/大小/类型限制返回 400 并给出原因；上游请求不泄露 Agent API key；临时文件请求结束后清理。
- `sub2api`：无附件仍发 JSON；有附件发 `FormData`（含 `message` / `locale` / `conversationId` / `user` / `attachments`）；前端拒绝超限文件；上传状态按钮禁用；流式回复照常展示。

## 相关

- 外部网关仓库：[lumio-ai-support-chat](https://github.com/Go1c/lumio-ai-support-chat)
- 前端接入：`frontend/src/api/supportChat.ts`、`frontend/src/components/support/SupportChatWidget.vue`
- 站点设置入口：管理后台「站点设置 → AI 客服」
