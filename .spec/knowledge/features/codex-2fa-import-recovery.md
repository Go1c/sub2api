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

初次 Team 优先，无 Team 才使用个人空间；恢复绑定原 Team。相同邮箱与 Team 的现有账号优先更新，否则通过 AdminService 创建配置。新账号并发 8、device 指纹、内置模型映射，支持既有分组与 IP 组。

401 在 RateLimitService 凭据 owner 分支先停调度后入队，覆盖 token_invalidated／Token revoked。独立后台任务在停调度状态下仍可运行，验证新凭据可用后仅合并认证字段、失效缓存、条件恢复 active＋schedulable，并原子写调度 outbox。失败保持停用，有冷却和手工重试；管理员状态及并发凭据变更不被覆盖。

材料使用现有 AES-GCM 加密器与固定密钥，不进入账号导出、extra 或任务响应。部署与输入契约见 [Worker 文档](../../../tools/codex-login-worker/README.md)。

后端：`codex_login*.go`、`account_codex_login.go`、`account_codex_2fa.go`；前端：`Codex2FAImportModal.vue`；迁移：`948_codex_login_jobs.sql`。

验证边界：第一个真实账号完成登录和 Team 凭据查询／刷新；401 恢复走回归测试，任务队列用隔离 PostgreSQL 验证。未部署生产、未制造真实凭据失效，其余四个账号未登录。

## 失败诊断

失败任务展示经白名单筛选的阶段（授权初始化、邮箱、密码、2FA、Team、换码、额度验证）、HTTP 状态码与异常类别。Worker HTTP 层必须保留子进程诊断字段；不得回传上游响应正文、URL、Cookie 或异常消息。重试请求被拒绝时展示后端实际原因，不能把未配置、网络失败等都描述为冷却。历史任务需更新 Worker 与主服务后重试，才会产生新诊断。

## 强制 IP 组登录

按用户明确要求禁用服务器直连。主服务读取有效 IP 组成员；Worker 随机打乱候选，通过代理协议无凭据探测登录入口，选通后固定代理完成登录及额度检查，登录失败不再换出口重复尝试。无组、无有效成员或探测耗尽均失败关闭。SOCKS DNS 交给代理，显式禁止 NO_PROXY 环境绕过。主服务记录候选数量及实际选中 proxy_id；失败文案带安全代理 ID。

本次验证采用本地 mock 代理及故障注入，不额外试登录生产账号。旧 Worker 不支持新候选协议时主服务必须报错，不接受无法确认代理 ID 的成功结果。
