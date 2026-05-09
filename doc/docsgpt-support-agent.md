# DocsGPT Support Agent 部署说明

本文说明如何部署 LumioAPI AI 客服依赖的 Doc Agent 服务。这里的 Doc Agent 指 DocsGPT 自托管服务中的 Support Agent，不是前端客服气泡，也不是 `support-gateway`。

## 服务拆分

生产链路分为三层：

```text
frontend-dashboard
  -> support-gateway
  -> DocsGPT backend /stream
  -> DocsGPT worker + PostgreSQL + Redis + 文档存储
  -> OpenAI-compatible model endpoint
```

密钥边界：

- 模型 API key 只放在 DocsGPT backend / worker。
- DocsGPT Agent API key 只放在 `support-gateway`。
- `frontend-dashboard` 只放公开的 `VITE_SUPPORT_CHAT_GATEWAY_URL`。

## 本地先验证

最稳的本地验证方式是先用 DocsGPT 官方 Docker Compose 跑通完整服务：

```bash
git clone https://github.com/arc53/DocsGPT.git
cd DocsGPT/deployment
docker compose up -d
```

启动后进入 DocsGPT 管理界面，完成：

1. 配置 OpenAI-compatible 模型。
2. 上传 LumioAPI FAQ/Markdown 知识库。
3. 创建 `LumioAPI Support` Agent。
4. 用 Agent 测试价格、充值、API key、模型配置、联系方式等问题。
5. 生成 Agent API key，后续填入 `support-gateway` 的 `DOCSGPT_AGENT_API_KEY`。

本地跑通后，再部署到 Zeabur。

## Zeabur 部署方式

Zeabur 不适合直接丢官方 compose 文件作为一个服务运行。按服务拆开部署：

| Zeabur 服务 | 类型 | 是否公网 | 说明 |
|-------------|------|----------|------|
| `docsgpt-postgres` | Managed PostgreSQL | 否 | DocsGPT 元数据、会话、Agent 配置。 |
| `docsgpt-redis` | Managed Redis | 否 | Celery broker/result backend 和缓存。 |
| `docsgpt-backend` | Docker service | 内网给 gateway，管理 UI 可访问 | DocsGPT API，提供 `/stream`。 |
| `docsgpt-worker` | Docker service | 否 | 文档上传、解析、索引任务。 |
| `docsgpt-admin-ui` | Docker/static service | 仅管理员 | 上传文档、创建 Agent、拿 Agent API key。 |
| `support-gateway` | 本仓库 `support-gateway/` | 是 | 浏览器唯一访问的 AI 客服 API。 |
| `frontend-dashboard` | 本仓库 `frontend-dashboard/` | 是 | LumioAPI 前端。 |

### 1. 创建数据库和 Redis

在同一个 Zeabur Project 中创建：

- PostgreSQL：记下连接串，例如 `postgres://...`
- Redis：记下连接串，例如 `redis://...`

后续 `docsgpt-backend` 和 `docsgpt-worker` 必须使用同一组 PostgreSQL / Redis。

### 2. 部署 `docsgpt-backend`

从 GitHub 部署 DocsGPT 官方仓库：

- Repository: `arc53/DocsGPT`
- Root Directory: `application`
- Dockerfile: 使用 `application/Dockerfile`
- Public domain: 可以先开启，方便 admin UI 调试；生产建议只允许 gateway 或管理员访问。

必填环境变量按你的实际服务填写：

```bash
# 数据层
POSTGRES_URI=postgres://user:password@host:5432/dbname
CACHE_REDIS_URL=redis://default:password@host:6379/0
CELERY_BROKER_URL=redis://default:password@host:6379/0
CELERY_RESULT_BACKEND=redis://default:password@host:6379/1

# 模型层：OpenAI-compatible
LLM_PROVIDER=openai
OPENAI_BASE_URL=https://your-model-api.example.com/v1
API_KEY=sk-your-model-key
LLM_NAME=your-model-name

# 生产建议
ENV=production
AUTO_CREATE_DB=true     # 首次部署可 true；稳定后建议改为 false，用显式迁移控制
AUTO_MIGRATE=true       # 首次部署可 true；稳定后建议改为 false，用显式迁移控制
```

如果 DocsGPT 当前版本的环境变量名和上面不同，以 `arc53/DocsGPT` 官方文档和 `.env` 示例为准。部署后确认 backend 的健康检查或首页/API 能访问。

### 3. 部署 `docsgpt-worker`

再部署一个 DocsGPT 官方仓库服务：

- Repository: `arc53/DocsGPT`
- Root Directory: `application`
- Dockerfile: 使用 `application/Dockerfile`
- Public domain: 关闭
- 环境变量：与 `docsgpt-backend` 保持一致，尤其是 PostgreSQL、Redis、模型配置、文档存储配置。

启动命令使用 DocsGPT worker/Celery 命令。不同版本命令可能变化，以官方 Docker Compose 中 worker 服务为准。原则是：worker 和 backend 使用同一镜像、同一配置、同一文档存储。

### 4. 文档存储

DocsGPT 需要保存上传的 FAQ/Markdown、索引和向量数据。Zeabur 上不要依赖容器临时文件系统。

生产二选一：

1. 给 backend / worker 绑定同一个持久化 Volume。
2. 配置 DocsGPT 支持的对象存储，例如 S3-compatible bucket。

对象存储相关变量请以 DocsGPT 官方 Settings 文档和当前版本 `.env` 示例为准。关键原则是 backend 和 worker 必须读写同一份上传文件和索引数据；如果它们看不到同一份文档存储，上传可能成功但索引或回答会失败。

### 5. 部署 `docsgpt-admin-ui`

admin UI 只给管理员使用：

- Repository: `arc53/DocsGPT`
- Root Directory: `frontend`
- 配置它指向 `docsgpt-backend` 的公开或内部 API 地址。
- 生产必须加访问控制：至少使用强账号密码，最好只临时开启或限制访问来源。

在 admin UI 中：

1. 上传 LumioAPI FAQ/Markdown。
2. 创建 `LumioAPI Support` Agent。
3. Agent prompt 建议：

```text
You are LumioAPI support. Answer using the uploaded knowledge base first.
If the answer is uncertain or requires account/payment verification, say the limit clearly and ask the user to contact official support.
Keep answers concise.
Respect the requested language passed by the integration.
Do not invent product, billing, or account facts.
```

4. 测试常见问题。
5. 创建/复制 Agent API key。

### 6. 连接 `support-gateway`

在本仓库部署 `support-gateway/`，填入：

```bash
DOCSGPT_API_BASE_URL=https://your-docsgpt-backend
DOCSGPT_AGENT_API_KEY=your-docsgpt-agent-key
ALLOWED_ORIGINS=https://your-frontend-domain
SUPPORT_EMAIL=support@example.com
SUPPORT_URL=https://your-support-page.com
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_REQUESTS=20
```

`support-gateway` 调 DocsGPT `/stream` 时会发送：

```json
{
  "question": "用户问题",
  "api_key": "DocsGPT Agent API key",
  "conversation_id": "可选会话 ID",
  "passthrough": {
    "locale": "zh-Hant",
    "language": "Traditional Chinese",
    "language_instruction": "Answer in Traditional Chinese.",
    "user_id": "可选用户 ID",
    "user_email": "可选用户邮箱"
  }
}
```

DocsGPT Agent API 会根据 `api_key` 加载 Agent 配置；`passthrough` 用于 prompt 模板里的动态变量。

验证：

```bash
curl https://your-support-gateway.zeabur.app/healthz
curl 'https://your-support-gateway.zeabur.app/widget-config?locale=zh-Hant'
```

然后在 `frontend-dashboard` 配置：

```bash
VITE_SUPPORT_CHAT_ENABLED=true
VITE_SUPPORT_CHAT_GATEWAY_URL=https://your-support-gateway.zeabur.app
```

## 语言设计

前端会把当前站点语言随每次请求发给 gateway：

- `en-US` -> `English`
- `zh-CN` -> `Simplified Chinese`
- `zh-Hant` -> `Traditional Chinese`

`support-gateway` 会把 `locale`、`language`、`language_instruction` 透传给 DocsGPT。Agent prompt 需要明确要求尊重该语言参数。

## 上线检查

- [ ] DocsGPT backend 能连接 PostgreSQL / Redis。
- [ ] DocsGPT worker 正常消费文档索引任务。
- [ ] backend 和 worker 共享同一份文档存储。
- [ ] admin UI 不向普通用户开放。
- [ ] Agent 能回答 FAQ/Markdown 中的问题。
- [ ] Agent 对账户、支付、不可验证问题会引导官方支持。
- [ ] `support-gateway` 不向浏览器返回 DocsGPT Agent API key。
- [ ] 前端网络面板只能看到 `support-gateway` URL。
- [ ] 切换 `en-US`、`zh-CN`、`zh-Hant` 后，下一条客服回复跟随当前语言。

## 关键代码

| 文件 | 说明 |
|------|------|
| `support-gateway/server.go` | `/widget-config`、`/chat/stream`、CORS、限流、DocsGPT SSE 转发、语言透传。 |
| `support-gateway/config.go` | `DOCSGPT_API_BASE_URL`、`DOCSGPT_AGENT_API_KEY`、`ALLOWED_ORIGINS` 等环境变量加载。 |
| `support-gateway/Dockerfile` | Zeabur 部署 gateway 的镜像。 |
| `frontend-dashboard/src/components/support/SupportChatWidget.vue` | 右下角客服气泡，读取当前 `locale` 并发送给 gateway。 |
| `frontend-dashboard/src/api/supportChat.ts` | 获取 widget config，解析 SSE 流。 |

## 参考

- DocsGPT GitHub: https://github.com/arc53/DocsGPT
- DocsGPT Docker deployment: https://docs.docsgpt.cloud/Deploying/Docker-Deploying
- DocsGPT settings: https://docs.docsgpt.cloud/Deploying/DocsGPT-Settings
- DocsGPT Agent API: https://docs.docsgpt.cloud/Agents/api
