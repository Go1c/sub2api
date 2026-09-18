---
name: codex-2fa-import-recovery
description: Codex 2FA JSON 导入、Team 优先授权，以及 sub2api 上游 401 停调度后的后台重新登录恢复
metadata:
  type: doc
  level: L2
  status: 已实现
---

# Codex 2FA 导入与恢复

入口为账号配置「更多操作 → 2FA JSON 导入」。Go 后端将加密登录材料与任务保存至 `codex_login_jobs`，内部 Python/Node Worker 完成 HTTP 授权；不依赖 Chrome 扩展、不推送 CPA。

初次 Team 优先，无 Team 才使用个人空间；恢复绑定原 Team。相同邮箱与 Team 的现有账号优先更新，否则通过 AdminService 创建配置。新账号并发 8、device 指纹、内置模型映射，支持既有分组与代理／IP 组。

401 在 RateLimitService 凭据 owner 分支先停调度后入队，覆盖 token_invalidated／Token revoked。独立后台任务在停调度状态下仍可运行，验证新凭据可用后仅合并认证字段、失效缓存、条件恢复 active＋schedulable，并原子写调度 outbox。失败保持停用，有冷却和手工重试；管理员状态及并发凭据变更不被覆盖。

材料使用现有 AES-GCM 加密器与固定密钥，不进入账号导出、extra 或任务响应。部署与输入契约见 [Worker 文档](../../../tools/codex-login-worker/README.md)。

后端：`codex_login*.go`、`account_codex_login.go`、`account_codex_2fa.go`；前端：`Codex2FAImportModal.vue`；迁移：`948_codex_login_jobs.sql`。

验证边界：第一个真实账号完成登录和 Team 凭据查询／刷新；401 恢复走回归测试，任务队列用隔离 PostgreSQL 验证。未部署生产、未制造真实凭据失效，其余四个账号未登录。
