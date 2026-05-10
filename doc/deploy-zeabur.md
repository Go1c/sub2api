# 部署到 Zeabur

部署目标：Zeabur（托管 PaaS），源码从 `publish` 分支拉取，构建 `frontend-dashboard/`。

## 前置

- [ ] Zeabur 账号，已关联 GitHub。
- [ ] 本地已 `pnpm build` 验证过 `frontend-dashboard/dist/` 正常生成。
- [ ] `publish` 分支上已有需要的 commits（从 `dev` 合入），见 `doc/branching.md`。

## 推荐部署方式（静态站点）

`frontend-dashboard/` 是纯前端 SPA，最简单是把构建产物作为静态站点部署。

### 方式 A：Zeabur 识别 Vite 自动构建

1. Zeabur 控制台 → New Project → Deploy from GitHub。
2. 选择本仓库 `Go1c/sub2api`，分支 **`publish`**。
3. Root Directory 设为 `frontend-dashboard`。Zeabur 会自动识别 Vite。
4. Build 配置（通常自动）：
   - Install: `pnpm install --frozen-lockfile`
   - Build: `pnpm build`
   - Output: `dist`
5. Environment variables（Settings → Variables）：
   ```
   VITE_API_BASE_URL=https://your-backend.zeabur.app    # 若连接真后端
   VITE_USE_MOCK=false
   VITE_SITE_NAME=LumioAPI
   ```
   留空 `VITE_API_BASE_URL` 会启用 mock（不推荐生产，但适合先上线看前端）。
6. Save，触发第一次部署。

### 方式 B：`zeabur.json` 显式声明

如果自动识别有问题，在 `frontend-dashboard/` 根目录加：

```json
{
  "framework": "vite",
  "install": "pnpm install --frozen-lockfile",
  "build": "pnpm build",
  "output": "dist"
}
```

## 与后端联调

若 Zeabur 上也部署了 sub2api 后端：

1. 后端服务的内部/公开域名填入前端 `VITE_API_BASE_URL`。
2. 后端需开 CORS 允许前端域名（见 `backend/internal/config/cors.yaml` 或等效配置，参考根 `README.md` "CORS 允许来源" 一节）。
3. 若用相同根域的子域，可在 Zeabur 配置 rewrite 把 `/api/*` 指回后端，前端 `VITE_API_BASE_URL` 留空、保留相对路径即可。

## DocsGPT AI 客服

AI 客服由四层服务协作：

1. DocsGPT 自托管服务：保存知识库和 Agent，持有模型 API key。
2. `github.com/Go1c/lumio-ai-support-chat`：独立 Go 服务，持有 DocsGPT Agent API key，向浏览器暴露 `/widget-config` 和 `/chat/stream`。
3. LumioAPI 后端：保存公开的客服开关、gateway URL 和展示文案。
4. `frontend-dashboard/`：读取公开设置并渲染右下角客服气泡。

Zeabur 部署 `lumio-ai-support-chat` 时直接选择该仓库根目录，可使用仓库内的 Dockerfile。必填环境变量：

```bash
DOCSGPT_API_BASE_URL=https://your-docsgpt-service
DOCSGPT_AGENT_API_KEY=your-docsgpt-agent-key
ALLOWED_ORIGINS=https://your-frontend-domain
SUPPORT_EMAIL=support@example.com
SUPPORT_URL=https://your-support-page.com
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_REQUESTS=20
```

`/widget-config?locale=zh-CN|zh-Hant|en-US` 会按前端当前语言返回默认客服文案。LumioAPI 后台中填写的客服标题、欢迎语、人工联系按钮文案会覆盖这些默认文案；gateway 里的 `WIDGET_TITLE`、`WELCOME_MESSAGE`、`OFFICIAL_CONTACT_TEXT` 只作为旧部署或本地联调 fallback。

Doc Agent 服务的完整部署步骤见 `lumio-ai-support-chat` 仓库文档。这里仅列 Zeabur 侧最小接线：

1. 先部署 DocsGPT：`docsgpt-postgres`、`docsgpt-redis`、`docsgpt-backend`、`docsgpt-worker`、`docsgpt-admin-ui`。
2. 在 DocsGPT admin UI 中上传 LumioAPI FAQ/Markdown，创建 `LumioAPI Support` Agent，复制 Agent API key。
3. 部署 `github.com/Go1c/lumio-ai-support-chat`，把 `DOCSGPT_API_BASE_URL` 指向 `docsgpt-backend`，把 `DOCSGPT_AGENT_API_KEY` 填成 Agent API key。
4. 部署 `frontend-dashboard/`，确保 `VITE_API_BASE_URL` 指向 LumioAPI 后端，或同域代理 `/api/v1` 可用。
5. 进入 LumioAPI 管理员后台“站点设置 -> AI 客服”，打开开关并填写 `support-gateway` 公网地址。

### 使用和验证

1. DocsGPT admin UI 中创建 LumioAPI Support Agent，并上传 FAQ/Markdown 知识库。
2. 在 DocsGPT 中配置 OpenAI-compatible 模型环境变量，例如 `OPENAI_BASE_URL`、`API_KEY`、`LLM_NAME`。
3. 从 DocsGPT Agent API 获取 Agent API key，填入 `support-gateway` 的 `DOCSGPT_AGENT_API_KEY`。
4. 打开 `support-gateway` 公网域名，确认：
   ```bash
   curl https://your-support-gateway.zeabur.app/healthz
   curl 'https://your-support-gateway.zeabur.app/widget-config?locale=zh-Hant'
   ```
5. 在 LumioAPI 管理员后台“站点设置 -> AI 客服”中启用，并填写 `https://your-support-gateway.zeabur.app`。
6. 打开前端，确认右下角客服气泡出现，网络面板只出现 gateway URL，看不到 DocsGPT Agent key 或模型 API key。

### 框架设计和关键代码

```text
Browser
  -> frontend-dashboard/src/components/support/SupportChatWidget.vue
  -> frontend-dashboard/src/api/supportChat.ts
  -> LumioAPI backend /settings/public
  -> support-gateway /chat/stream
  -> DocsGPT /stream
  -> OpenAI-compatible model endpoint
```

关键实现：

| 文件 | 说明 |
|------|------|
| `github.com/Go1c/lumio-ai-support-chat/server.go` | HTTP 路由、CORS、限流、`/widget-config` 本地化、`/chat/stream` SSE 转发。 |
| `github.com/Go1c/lumio-ai-support-chat/config.go` | 从环境变量加载 DocsGPT URL、Agent key、允许来源、支持入口、限流参数。 |
| `github.com/Go1c/lumio-ai-support-chat/Dockerfile` | Zeabur 可直接构建的 Go gateway 镜像。 |
| `github.com/Go1c/lumio-ai-support-chat/.env.example` | Gateway 部署变量模板。 |
| `backend/internal/service/setting_service.go` | 公开 AI 客服开关、gateway URL 和展示文案。 |
| `frontend/src/views/admin/SettingsView.vue` | 管理员后台 AI 客服配置入口。 |
| `frontend-dashboard/src/api/supportChat.ts` | 读取公开设置、连接 gateway，并解析 DocsGPT SSE 事件。 |
| `frontend-dashboard/src/components/support/SupportChatWidget.vue` | 右下角气泡 UI，按当前站点语言传 `locale`。 |

语言策略：

- 前端当前 locale 会发给 `/widget-config?locale=...`，用于返回默认欢迎语和按钮文案。
- 每次 `POST /chat/stream` 都带 `locale`，gateway 会映射为 `language` 和 `language_instruction` 透传给 DocsGPT。
- 用户在同一会话中切换语言后，下一条消息会使用新的前端语言。

## 域名与 HTTPS

- Zeabur 默认给 `*.zeabur.app` 域名，已启 TLS。
- 自定义域名：Zeabur 控制台 → Domains → Add。

## 发布流程

```bash
# 1. 在 dev 上充分测试
git checkout dev && pnpm --dir frontend-dashboard build
# 2. 合入 publish
git checkout publish
git merge --no-ff dev -m "release: vX.Y.Z"
# 3. 打 tag 并推送
git tag -a vX.Y.Z -m "release X.Y.Z"
git push origin publish --tags
# 4. Zeabur 检测到 publish 更新，自动部署
```

## 回滚

Zeabur 控制台 → Deployments → 选择上一个绿色 deploy → Redeploy。
或 `git revert <bad-sha>` 到 `publish`，push 重新触发。

## 常见坑

| 现象 | 原因 | 解决 |
|------|------|------|
| 部署成功但页面白屏 | SPA 路由没配 rewrite | Zeabur Settings → 勾选 "Rewrite to index.html" 或加 `_redirects: /* /index.html 200` |
| 字体加载慢 | Google Fonts CDN 被网络限制 | 换成本地托管 —— 把 WOFF2 放 `public/fonts/` 并在 `tokens.css` 用 `@font-face` |
| 环境变量不生效 | 变量名没有 `VITE_` 前缀 | Vite 只暴露 `VITE_*` 前缀的环境变量给客户端 |
| build 失败 `Cannot find module 'node'` | 镜像缺 Node 22+ | package.json 里 `engines.node >= 18` 已声明；若 Zeabur 仍取了旧版本，Settings 里强制指定 |
