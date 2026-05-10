# DocsGPT Support Agent 部署说明

AI 客服的独立服务代码和完整 DocsGPT 部署文档已迁移到独立仓库：

```text
https://github.com/Go1c/lumio-ai-support-chat
```

当前 `sub2api` 项目只保留接入层：

- LumioAPI 后端保存公开的 AI 客服开关、gateway URL 和展示文案。
- 管理员后台提供“站点设置 -> AI 客服”配置入口。
- `frontend-dashboard` 读取公开设置并连接外部 support gateway。

部署或维护 DocsGPT、Agent API key、CORS、限流和 SSE 转发时，请在 `lumio-ai-support-chat` 仓库中处理。
