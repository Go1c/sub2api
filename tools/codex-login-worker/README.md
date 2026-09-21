# sub2api Codex 2FA 登录 Worker

内部 Python/Node 服务为当前 sub2api 执行已有账号的密码＋TOTP 登录。无需 Chrome 扩展；不注册账号，不购买短信，不推送 CPA。成功授权后查询 Codex 额度验证凭据可用性。

## 部署

主服务保持 Go/Alpine；登录 Worker 使用 Python/glibc 镜像，避免 curl_cffi 依赖影响主镜像。

构建包含本次改动的 sub2api 镜像，在原部署环境配置：

- `SUB2API_CODEX_LOGIN_IMAGE`：本次构建的主镜像名，不能使用未包含此功能的上游镜像。
- `CODEX_LOGIN_WORKER_TOKEN`：随机、至少 32 字符的共享密钥。
- `TOTP_ENCRYPTION_KEY`：固定 64 位十六进制密钥；已有配置时保留原值，不能替换，否则旧数据无法解密。

从仓库根在原有 compose 参数中叠加：

```sh
docker compose --env-file deploy/.env \
  -f deploy/docker-compose.yml \
  -f deploy/docker-compose.codex-login.yml up -d --build
```

沿用已有项目名、环境文件和数据卷。overlay 适用于使用 `sub2api-network` 的主／local compose；Worker 不暴露宿主端口。不用 compose 时单独运行 Worker，设置 `WORKER_TOKEN`，Go 服务设置 `CODEX_LOGIN_WORKER_URL`、`CODEX_LOGIN_WORKER_TOKEN` 和固定加密密钥。跨主机连接须使用 HTTPS。代理必须能从 Worker 容器访问，容器内 localhost 不是宿主机。

Go 启动时应用迁移 `948_codex_login_jobs.sql`。Worker 或固定密钥未配置时入口禁用，普通账号保留原行为。

## 导入

账号配置 → 更多操作 → 2FA JSON 导入。一次可选 5 个 JSON，也可上传数组文件，最多 20 个文件、100 个账号、合计 1 MB。确认账号分组和默认出口后提交，后台逐个执行。关闭弹窗不取消任务。

```json
[
  {"email":"one@example.com","password":"account-password","totp_secret":"BASE32-SECRET"},
  {"email":"two@example.com","password":"account-password","2fa_sk":"BASE32-SECRET"}
]
```

仅占位示例。支持单对象、数组、`accounts`／`data` 数组外壳。密钥别名：`totp_secret`、`2fa_secret`、`2fa_sk`、`2fa`、`otp_secret`、`secret`。兼容 `邮箱----密码----密钥` TXT。

初次授权 Team 优先，无 Team 才选个人；多个 Team 时在 JSON 指定上游 `account_id`。恢复已有账号始终绑定原 Team。相同邮箱＋Team 的已有账号更新绑定，不重复创建；重复配置超过一个时需先整理。新账号默认优先级 1、并发 8、device 指纹、内置模型映射；恢复保留原配置。

## 401 恢复

直接接在 sub2api RateLimitService，包括 Spark 影子的凭据 owner。`Token revoked (401)`／`token_invalidated` 后先标为 error 并关闭 schedulable，再由独立队列登录。成功后核对邮箱／Team、查询额度、合并认证字段、清缓存，条件恢复 active＋schedulable 并写调度 outbox。原请求按网关现有切号逻辑处理，不自动重放。

失败保持停用，五分钟后可重试或重新导入修正材料。服务中断的过期任务标失败，不无限重登。管理员修改状态或期间凭据改变时，不覆盖这些变更。PostgreSQL 租约与原 OAuth Redis 刷新锁防止重复登录／更新。

登录材料经 AES-GCM 加密存独立私有表，不进入账号导出、extra 或任务响应。Worker 认证密钥、登录材料、Cookie 均不放命令行；Cookie 通过 Node stdin。参考来源和本地补丁见 `third_party/codex_protocol/NOTICE.md`。

## 验证边界

仅对指定文件第一个账号做真实登录、Team 授权、额度查询和 RT 刷新，其余四个未登录。生产服务器出口、线上部署和真实凭据失效的完整恢复尚未线上验收。

Python 测试：`python -m pytest test_login_core.py test_worker_privacy.py`。Go 测试见 `codex_login*_test.go`；数据库测试需显式设置隔离库的 `CODEX_LOGIN_TEST_DSN`，禁止指向生产。

## 部分账号失败时排查

成功账号不需重新导入。更新主服务和 Worker 后，只重试失败项。页面显示失败阶段、HTTP 状态码和安全异常类别；旧失败记录不会自动获得详情，需重新执行任务。

TOTP 阶段 MFA 接口 403 标为 `mfa_rejected`（2FA 被上游拒绝）；400/401 仍为 `invalid_totp`。密码接口 403 是额外验证，不能据此认定密码错误。

Retry 保留原 IP 组配置；未绑定账号的失败任务换组需通过「换 IP 组并重新导入」手动选组，提供同邮箱材料后更新同一任务。已绑定账号恢复使用账号当前出口，须先编辑账号出口。

例如“Codex 额度验证；HTTP 403”表示登录之后的额度查询被拒绝，不等于密码错误；“授权初始化；网络超时”应检查 Worker 出口。不要从通用失败提示推断账号已封禁，也不要关闭身份／Team 校验来让任务变成成功。重试的冷却限制仍保留。

## 登录出口强制规则

2FA 导入默认按已启用且配置完整的粘性动态 IP 组、普通 IP 组、单个代理排序；弹窗显式提交显示的出口 ID；直接调用 API 未指定时由服务端兜底，手动选择保留。恢复使用原账号绑定的组或单代理；无绑定或绑定不可用时失败，不回退服务器直连。粘性组登录会生成新的供应商 session。

主服务仅传递组内启用且未过期的候选代理，Worker 随机打乱后逐条用 HTTP CONNECT／SOCKS 协议访问 `https://auth.openai.com/log-in`，返回 200 才进入登录。每条请求超时 5 秒、外层上限 6 秒，整个选择阶段最多 30 秒；时限内没有成功代理就返回代理选择失败及 `candidate_count` / `tried_count`，不带选中代理；预算可能在探完候选前耗尽。302 仍不算通过，不把可能的挑战跳转当作登录可达。探测不含邮箱、密码或 TOTP，登录开始后不再换 IP、也不自动重复登录。

一次任务的登录、2FA、换票、额度验证使用同一条代理。SOCKS5 转为 SOCKS5h，使目标 DNS 在代理侧解析；显式清空 libcurl NOPROXY，环境 NO_PROXY 不能绕过代理。验证只保证当时登录入口可用，不保证后续永不超时。

`codex_login_proxy_selected` 在主服务收到结果时记录实际 proxy_id，后续登录失败也记录；失败诊断带选中的代理 ID，不记录代理地址或密码。新主服务与 Worker 必须同时升级：旧 Worker 不支持候选代理列表时任务会失败关闭，不能让旧版空代理逻辑直连。
