# AI 客服附件上传需求

## 背景

用户希望在站内 AI 客服聊天中上传图片或文件，让 AI 客服结合附件内容回答问题。

当前 `sub2api` 只负责前端客服浮窗和公开配置读取。真正的 AI 客服网关在独立仓库 `lumio-ai-support-chat`。已检查该仓库当前实现：只有 `POST /chat/stream` 文本流接口，入参为 `{ message, conversationId?, locale?, user? }`，请求体使用 `32KB` JSON 限制，没有 multipart、附件上传、文件存储或附件转发协议。

## 目标

在 `lumio-ai-support-chat` 中补齐附件能力，然后让 `sub2api` 前端接入该协议。用户应能在客服输入框旁选择附件，发送后由 AI 客服读取附件内容并回答。

## 非目标

- 不做管理员知识库上传。该需求只处理“终端用户在客服会话中上传附件给 AI 看”。
- 不做长期文件管理后台。
- 不把模型 API key、DocsGPT Agent API key 或对象存储密钥暴露给浏览器。

## 支持范围

首版建议支持：

- 图片：`image/png`、`image/jpeg`、`image/webp`。
- 文档：`text/plain`、`text/markdown`、`application/pdf`。
- 单次最多 3 个附件。
- 单文件最大 10MB，总大小最大 20MB。

不支持的文件类型必须在前端和网关都拒绝，并返回清晰错误。

## 推荐接口

### 方案 A：单接口 multipart 流式发送

新增或扩展：

```text
POST /chat/stream
Content-Type: multipart/form-data
Accept: text/event-stream
```

字段：

- `message`：用户文本，必填。
- `conversationId`：可选。
- `locale`：可选。
- `user`：可选 JSON 字符串，结构与现有 `user` 一致。
- `attachments`：重复文件字段，可传 1 到 3 个文件。

响应继续使用现有 SSE 事件：`answer`、`id`、`error`、`end`。这样 `sub2api` 前端只需要把文本 JSON 请求切换为 `FormData`，不需要先上传再发消息。

### 方案 B：两步上传

新增：

```text
POST /attachments
Content-Type: multipart/form-data
```

返回：

```json
{
  "attachments": [
    {
      "id": "att_...",
      "name": "screenshot.png",
      "mimeType": "image/png",
      "size": 12345
    }
  ]
}
```

再调用：

```text
POST /chat/stream
Content-Type: application/json
```

入参增加：

```json
{
  "message": "请看这个截图的问题",
  "attachments": ["att_..."]
}
```

该方案更适合未来做对象存储、病毒扫描、附件复用和更严格审计，但首版实现成本更高。

## 推荐选择

首版采用方案 A。它改动少，保留现有流式聊天体验，并且无需持久保存用户附件。网关可以在一次请求内完成文件校验、提取内容或转发给上游。

## 网关实现要求

- 保持原有 JSON `POST /chat/stream` 兼容，未上传附件时现有前端不受影响。
- multipart 请求必须限制总读取大小，避免内存被大文件打满。
- 校验文件数量、大小、扩展名和 MIME 类型。
- 不信任浏览器传入的 MIME 类型；至少用文件头做基础检测。
- 图片附件应转为上游模型可理解的图片输入，或转成 DocsGPT 支持的格式。
- PDF 和文本附件应提取文本后追加到本轮问题上下文；提取失败时返回 SSE `error`。
- 不在 SSE 或日志中输出附件原文、base64、密钥或用户敏感信息。
- 附件只用于当前会话回合；如需临时文件，响应结束后清理。
- CORS `Access-Control-Allow-Headers` 继续允许 `Content-Type` 和 `Accept`。
- 限流策略继续按客户端 IP 生效，附件请求也必须计入。

## 前端接入要求

在 `sub2api` 的 `SupportChatWidget.vue` 和 `supportChat.ts` 中接入：

- 输入框左侧增加附件按钮，使用现有 `Icon` 的 `upload` 图标。
- 选择文件后在输入框上方展示文件名、大小和移除按钮。
- 前端执行同网关一致的数量、大小和类型校验。
- 有附件时用 `FormData` 调 `/chat/stream`；无附件时继续使用现有 JSON 请求。
- 发送中禁用附件按钮和移除按钮。
- 流式回答、失败重试和会话 ID 行为保持不变。
- 失败重试应保留上一次文本和附件，除非用户手动清除。

## 测试要求

`lumio-ai-support-chat`：

- JSON 文本请求保持兼容。
- multipart 请求能接收文本和图片附件。
- 超出数量、大小或类型限制时返回 400，并通过 SSE 或普通错误响应给出原因。
- 上游请求不泄露 Agent API key。
- 临时文件在请求结束后被清理。

`sub2api`：

- 无附件时仍发送 JSON 请求。
- 有附件时发送 `FormData` 请求，包含 `message`、`locale`、`conversationId`、`user` 和 `attachments`。
- 前端拒绝超限文件。
- 上传状态下按钮禁用。
- 流式回复照常展示。

## 验收标准

- 用户能上传一张截图并询问“这个报错是什么意思”，AI 能结合截图内容回答。
- 用户上传不支持的文件类型时，前端立即提示，网关也会拒绝绕过前端的请求。
- 浏览器网络面板看不到 DocsGPT Agent API key、模型 API key 或对象存储密钥。
- 现有无附件聊天不回归。
