# DocsGPT AI 客服接入方案

## Summary

采用 **DocsGPT 自托管 + 独立 support-gateway + 管理后台配置 + LumioAPI Vue 气泡组件**。

DocsGPT 负责 AI Agent、FAQ/Markdown 知识库、OpenAI-compatible 模型调用；support-gateway 隐藏 DocsGPT Agent API key，负责 CORS、限流和流式转发；LumioAPI 管理后台负责公开开关、gateway URL 和展示文案；`frontend-dashboard` 只展示右下角客服气泡并调用网关。

部署目标为 Zeabur。同平台可行，但不能直接部署 Docker Compose YAML，需要将 DocsGPT 拆成多个 Zeabur 服务。

## Architecture

请求链路：

```text
Browser
  -> frontend-dashboard SupportChatWidget
  -> LumioAPI backend /settings/public
  -> support-gateway /chat/stream
  -> DocsGPT /stream
  -> OpenAI-compatible model endpoint
```

密钥边界：

- 模型 API key 放在 DocsGPT 服务环境变量中。
- DocsGPT Agent API key 放在 support-gateway 服务环境变量中。
- LumioAPI 后台只保存公开的开关、gateway URL 和展示文案。
- `frontend-dashboard` 只读取公开 settings，不放任何 key。

## Key Changes

### frontend-dashboard

- 在 `App.vue` 全站挂载 `SupportChatWidget.vue`。
- 新增 `src/api/supportChat.ts`，优先读取后端公开 settings，再调用 support-gateway。
- 保留本地/旧部署 fallback 环境变量：
  - `VITE_SUPPORT_CHAT_ENABLED`
  - `VITE_SUPPORT_CHAT_GATEWAY_URL`
- 聊天窗口支持：
  - 右下角气泡入口。
  - 流式回复。
  - 错误状态和重试。
  - 清空会话。
  - 来源展示。
  - 邮件和官方支持快捷入口。
- 客服语言跟随前端当前 `locale`：
  - `en-US` 返回英文。
  - `zh-CN` 返回简体中文。
  - `zh-Hant` 返回繁体中文。
- 全站显示，包括营销页、登录/注册页、控制台页面；登录后可附带用户 `id`、`email` 给 support-gateway。

### LumioAPI admin/backend

- 后端设置新增公开字段：
  - `support_chat_enabled`
  - `support_chat_gateway_url`
  - `support_chat_title`
  - `support_chat_welcome_message`
  - `support_chat_official_contact_text`
- 管理员后台“站点设置 -> AI 客服”负责开启和配置。
- 启用时校验 gateway URL 必须是完整 HTTP(S) 地址。

### support-gateway

独立服务，公开给浏览器调用。

接口：

- `GET /healthz`：健康检查。
- `GET /widget-config?locale=...`：返回可公开配置，如客服标题、欢迎语、支持邮箱、官方支持链接；未配置覆盖文案时按 locale 返回默认文案。
- `POST /chat/stream`：接收 `{ message, conversationId?, locale?, user? }`，服务端带 `DOCSGPT_AGENT_API_KEY` 调 DocsGPT `/stream`，以 SSE 返回。

配置通过环境变量或 YAML 管理：

- `DOCSGPT_API_BASE_URL`
- `DOCSGPT_AGENT_API_KEY`
- `ALLOWED_ORIGINS`
- `SUPPORT_EMAIL`
- `SUPPORT_URL`
- `OFFICIAL_CONTACT_TEXT`
- `WIDGET_TITLE`
- `WELCOME_MESSAGE`
- `RATE_LIMIT_WINDOW_SECONDS`
- `RATE_LIMIT_MAX_REQUESTS`

### DocsGPT

- 使用 DocsGPT 自托管作为 AI 客服底座。
- 创建一个 LumioAPI Support Agent。
- 绑定 FAQ/Markdown 知识库。
- 系统提示词要求：
  - 优先基于知识库资料回答。
  - 不确定时说明限制，不编造。
  - 涉及账户、支付、不可验证的问题，引导用户发送邮件或联系官方支持。
  - 回答保持简洁，可用中文优先。

模型配置通过 DocsGPT 服务环境变量设置：

```env
LLM_PROVIDER=openai
OPENAI_BASE_URL=https://your-model-api.example.com/v1
API_KEY=sk-your-model-key
LLM_NAME=your-model-name
```

support-gateway 配置示例：

```env
DOCSGPT_API_BASE_URL=https://your-docsgpt-service
DOCSGPT_AGENT_API_KEY=your-docsgpt-agent-key
ALLOWED_ORIGINS=https://your-site.com
SUPPORT_EMAIL=support@example.com
SUPPORT_URL=https://your-support-page.com
OFFICIAL_CONTACT_TEXT=联系官方支持
```

管理员后台配置公开开关和 gateway URL；frontend-dashboard 的变量只作为本地/旧部署 fallback：

```env
VITE_SUPPORT_CHAT_ENABLED=false
VITE_SUPPORT_CHAT_GATEWAY_URL=
```

## Zeabur Deployment

Zeabur 官方文档说明当前不支持直接从 Docker Compose YAML 部署，所以不能直接丢 DocsGPT 官方 compose 文件。

在 Zeabur 同项目中拆分服务：

- `docsgpt-postgres`：托管 PostgreSQL。
- `docsgpt-redis`：托管 Redis。
- `docsgpt-backend`：DocsGPT API，暴露内部地址给 gateway。
- `docsgpt-worker`：同镜像不同启动命令，用于文档处理和索引。
- `docsgpt-admin-ui`：仅管理员使用，用于上传 FAQ/Markdown、创建 Agent。
- `support-gateway`：公网暴露，前端只连接它。
- `backend`：LumioAPI 后端，公开 AI 客服开关和展示配置。
- `frontend-dashboard`：现有 LumioAPI 前端静态站点。

免费计划可用于验证，但生产要注意：

- Zeabur Free Plan 可能有自动休眠、无 SLA、无自动数据库备份或日志转发等限制。
- 模型 API 调用费用不属于免费托管范围。
- DocsGPT、PostgreSQL、Redis、worker 长期运行可能超出免费额度，需要上线前实测资源占用。

## Test Plan

本地验证：

- DocsGPT 能用自定义 `OPENAI_BASE_URL`、`API_KEY`、`LLM_NAME` 返回答案。
- 上传 FAQ/Markdown 后，Agent 能回答价格、API key、充值、模型配置、联系方式问题。
- support-gateway 不向浏览器暴露 DocsGPT Agent key。
- `POST /chat/stream` 能稳定转发 `answer`、`source`、`error`、`end` 事件。

前端验证：

- 未登录、登录后、控制台页面均显示右下角气泡。
- 移动端窗口不遮挡主要操作，桌面端固定右下角。
- 流式回复、失败重试、清空会话、邮件和官方支持链接正常。
- 管理后台开启、关闭和修改文案后，前端刷新后按公开 settings 生效。
- 浏览器网络面板中只能看到 support-gateway URL，看不到 DocsGPT Agent key 或模型 API key。

部署验证：

- Zeabur 服务间环境变量和内部地址可用。
- DocsGPT backend、worker、PostgreSQL、Redis 均健康。
- 前端生产环境指向 LumioAPI 后端，AI 客服由管理员后台开启和配置。
- DocsGPT admin UI 不暴露给普通用户，或至少通过强认证保护。

## Assumptions

- 首版不做人工坐席或工单系统；AI 客服开关和公开展示配置放在 LumioAPI 管理后台。
- 首版知识来源为 FAQ/Markdown，不做自动抓站或复杂爬取。
- 本站使用自研 Vue 气泡，不嵌 DocsGPT 官方 widget。
- 模型接口先按 OpenAI-compatible 规划。
- 如后续发现 Zeabur 免费额度不足，保留迁移到独立 VPS Docker Compose 的备选方案。

## References

- DocsGPT GitHub and MIT license: https://github.com/arc53/DocsGPT
- DocsGPT Docker deployment: https://docs.docsgpt.cloud/Deploying/Docker-Deploying
- DocsGPT Agent API: https://docs.docsgpt.cloud/Agents/api
- DocsGPT Chat Widget: https://docs.docsgpt.cloud/Extensions/chat-widget
- Zeabur docs: https://zeabur.com/docs
