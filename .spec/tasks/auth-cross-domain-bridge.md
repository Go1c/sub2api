---
status: completed
title: BestCodex 跨域登录交接 /auth/bridge
---

# BestCodex 跨域登录交接（`/auth/bridge`）

只做 BestCodex 契约「改动一：跨域登录交接」。不要做 logo / `site_home_url` / 404 回主站 / 桌面端 / BestCodex 仓库 / 其它 SSO。

契约原文：`/Users/cui/Sites/lumio-codex/docs/upstream/lumioapi-portal-integration.md` 的「改动一」。

## 背景

BestCodex 门户的 `lumio_at` 就是本站签发的 access JWT。收银台 `https://api.lumio.games/purchase` 需要控制台 localStorage 会话，和 `bestcodex.app` Cookie 隔离。打开 `/purchase` 会 302 `/login?redirect=/purchase`。

## 必须做

### 1. `POST /api/v1/auth/bridge`

- `Authorization: Bearer <现有 access token>`
- 200，信封沿用 `response.Success`：`{ code: 0, message: "success", data: { access_token, refresh_token, expires_in, token_type, user } }`
- 校验复用现有 JWT access 中间件（挂在 `backend/internal/server/routes/auth.go` 的 `authenticated` 组，与 `/auth/me` 同组）
- Handler 复用 `respondWithTokenPair`：换发**新**令牌对，不回显入站 token
- 只接受未过期 access JWT。refresh 是 opaque（`rt_` 前缀），现有 `ValidateToken` 会 401，不要另开 refresh 直调路径
- **不要**把本路径加入 `isUserAccessTokenAllowedPath`；`uat_` 必须继续 403 `ACCESS_TOKEN_SCOPE_DENIED`
- 可选：给该路径加与 refresh 同量级的限流（每分钟 30、Redis fail-close）。不要改其它鉴权语义

### 2. 前端公开页 `/auth/bridge`

- 新路由：`path: '/auth/bridge'`，`requiresAuth: false`，`title: '登录中…'`
- 新页面：`frontend/src/views/auth/BridgeView.vue`（最小「登录中…」页，不是首页壳，不是 404）
- 放在 `/:pathMatch(.*)*` 之前，和其它 `/auth/*` 公开路由一起
- 把 `/auth/bridge` 加入 `BACKEND_MODE_CALLBACK_PATHS`，避免 backend mode 把交接页打回登录页
- 读 `location.hash` 的 `t`（token）和 `r`（目标路径，默认 `/purchase`）
- `r` 只允许站内相对路径：以 `/` 开头且不含 `//`（先 decode 再校验）。非法 / 缺失 → 当 `/purchase`
- 成功：按 `stores/auth.ts` 写入控制台会话。把内部 `setAuthFromResponse` 暴露为 public `applyAuthResponse`，Bridge 必须走它，不要手写 localStorage
- 先 `history.replaceState` 抹掉 `#t=`（pathname+search，不要留下 hash），再 `router.replace(r)`
- 失败（缺 t / 伪造 / 过期 / 网络失败）：同样先抹 hash，再 `router.replace({ path: '/login', query: { redirect: r } })`
- **不要**用 `apiClient` 调 bridge。`frontend/src/api/client.ts` 的 request 拦截器会用 localStorage token **覆盖** Authorization；401 拦截器会 refresh / `window.location.href='/login'` 且丢掉 `redirect`。用 `fetch`（参照 `probeCookieSession`）显式带 `Authorization: Bearer <t>`，自己解 `{code,data}` 信封
- 不要把 token 打到 console / toast / 错误日志
- 不要把 refresh token 放进 URL

### 不要做

- 不要改 `api.lumio.games` 主机名
- 不要做 logo / `site_home_url` / 404 回主站
- 不要改桌面端或 BestCodex 仓库
- 不要做其它 SSO / 统一网关
- 不要在当前 `dev-2` worktree / release 分支上改；本任务只改 `/Users/cui/orca/workspaces/sub2api/auth-bridge`

## TDD

必须先红后绿。建议循环：

1. 后端：`POST /api/v1/auth/bridge` 路由存在；未认证 401；已认证换发新 token 对且 `access_token` ≠ 入站 token；refresh / `uat_` 不能当 access 用
2. 前端 helper：hash `t`/`r` 解析、`r` 安全校验、默认 `/purchase`
3. `BridgeView`：成功写会话 + 抹 hash + `replace(/purchase)`；伪造 t → `/login?redirect=/purchase`
4. 路由：`/auth/bridge` 公开、title `登录中…`

测试参考：

- `backend/internal/handler/auth_session_revocation_test.go`（`NewAuthService` + `userHandlerRepoStub` + refresh cache stub + `respondWithTokenPair` 依赖）
- `backend/internal/server/middleware/user_access_token_auth_test.go`（断言 `POST /api/v1/auth/bridge` 不在白名单）
- `frontend/src/router/__tests__/wechat-route.spec.ts`
- `frontend/src/views/auth/__tests__/LoginView.spec.ts`
- `frontend/src/stores/__tests__/auth.spec.ts`

## 沉淀

用 spec-steward 新增 `.spec/knowledge/features/auth-cross-domain-bridge.md`，并在 `.spec/knowledge/README.md` 加一行。`external-auth-handoff.md` 是「外部来本站登录再回跳」，方向相反，不要混写。

## 验收

- [x] `POST /api/v1/auth/bridge` 不再 404
- [x] 有效 access JWT → 200，新 `access_token` / `refresh_token` / `expires_in` / `user`，不等于入站 token
- [x] 缺 Authorization / 伪造 / 过期 → 401；refresh 直调 → 401；`uat_` → 403
- [x] `GET /auth/bridge` 是交接页（title 登录中…），不是首页壳 / 404
- [x] 伪造或过期 `t` 落到 `/login?redirect=<r>`
- [x] 地址栏和 history 不残留 `#t=`
- [x] `/auth/bridge#t=<token>&r=/purchase` 成功后进入 `/purchase` 且为登录态
- [x] `cd backend && go test -tags=unit ./internal/handler ./internal/server/routes ./internal/server/middleware`
- [x] `cd frontend && pnpm typecheck`；相关 vitest 通过
- [x] `cd frontend && pnpm build`
- [x] 知识文档已更新

## 验证后回报

主 loop 需要：实际上线路径、成功/失败响应信封示例、是否已部署到 api.lumio.games（coder 只实现，不要自己发版）。
