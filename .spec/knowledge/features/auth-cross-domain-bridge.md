---
name: auth-cross-domain-bridge
description: 主站 access JWT 经 /auth/bridge 换成控制台 localStorage 会话——排查跨域登录交接、开放重定向或 token 残留时查这篇
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 跨域登录交接（`/auth/bridge`）

简介：BestCodex / 主站持有的 `lumio_at` 就是本站签发的 access JWT，但不能写入 `api.lumio.games` 的 localStorage。`/auth/bridge` 用 URL **片段**接过 token，后端换发新令牌对，前端写入控制台会话后再进收银台。

方向与 [`external-auth-handoff.md`](./external-auth-handoff.md) **相反**：handoff 是「外部来本站登录，再把 token 带回外部」；bridge 是「主站已登录，把已有 access token 交给本站控制台」。不要混写。

## 背景 / 目标

主站会话 cookie 在 `.lumiogame.com`，控制台会话在 `api.lumio.games` 的 `auth_token` / `refresh_token`。直接打开 `/purchase` 会 302 `/login?redirect=/purchase`。

目标：主站登录后点「充值」→ `https://api.lumio.games/auth/bridge#t=<access_token>&r=/purchase` → 控制台已登录并进入 `/purchase`；地址栏与 history 不残留 `#t=`。

## 设计

### 交互面

主站（有会话）跳：

```text
https://api.lumio.games/auth/bridge#t=<access_token>&r=/purchase
```

无会话时主站仍直链 `/purchase`（本仓不改主站）。

页面 `/auth/bridge` 公开，`requiresAuth: false`，title 精确为 `登录中…`，已加入 `BACKEND_MODE_CALLBACK_PATHS`。

`r` 先 decode 再校验：必须以 `/` 开头且不含 `//`，否则默认 `/purchase`。

无论成败都先 `history.replaceState` 抹掉 hash（只留 pathname+search），再 `router.replace`：

- 成功 → `r`
- 缺 `t` / 伪造 / 过期 / 网络失败 → `/login?query.redirect=r`

### 实现面

**后端** `POST /api/v1/auth/bridge`

- 挂在 `backend/internal/server/routes/auth.go` 的 **authenticated** 组，复用 JWT 中间件
- Handler `AuthHandler.Bridge` 读 `AuthSubject`，`userService.GetByID` 后走 `respondWithTokenPair`（换发新 access+refresh，不回显入站 token）
- 不支持 refresh 直调：`rt_` 走 `ValidateToken` → 401
- **不要**把本路径加入 `isUserAccessTokenAllowedPath`：`uat_` → 403 `ACCESS_TOKEN_SCOPE_DENIED`
- 限流与 refresh 同量级：每分钟 30，Redis fail-close

成功信封（`response.Success`）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "<new jwt>",
    "refresh_token": "rt_<opaque>",
    "expires_in": 3600,
    "token_type": "Bearer",
    "user": { "id": 1, "email": "user@example.com" }
  }
}
```

失败（JWT 中间件，`code` 是字符串）：

| 情况 | HTTP | `code` |
|------|------|--------|
| 缺 Authorization | 401 | `UNAUTHORIZED` |
| 伪造 / refresh (`rt_`) | 401 | `INVALID_TOKEN` |
| 过期 access JWT | 401 | `TOKEN_EXPIRED` |
| `uat_` | 403 | `ACCESS_TOKEN_SCOPE_DENIED` |

**前端**

- `frontend/src/utils/authBridge.ts`：hash 解析 + `fetch` 换发（**禁止** `apiClient`：请求拦截器会覆盖 `Authorization`，401 会 `window.location.href='/login'` 丢掉 redirect）
- `frontend/src/views/auth/BridgeView.vue`：最小「登录中…」页
- 会话写入必须走 `useAuthStore().applyAuthResponse`（内部仍是 `setAuthFromResponse`）
- 不要把 token 打到 console / toast / 错误对象

## 已决策

- token 只放 URL 片段，不进 query，避免进访问日志。
- 换发新令牌对，不回显入站 token。
- `r` 只允许站内相对路径，防开放重定向。
- refresh / `uat_` 不能当 bridge access 用。
- 本轮只做改动一，不做 logo / `site_home_url` / 404 / 桌面端 / 其它 SSO。

## 待解决

- 尚未部署到 `api.lumio.games`（实现完成后由发布流程上线）。
- backend mode 下页面可打开，但 API 仍走 `BackendModeUserGuard`：非管理员有效 JWT 会 403（沿用既有鉴权语义）。
- 桌面端 / CC 不走 bridge，无浏览器会话时落登录页是预期。

## 相关

- 任务卡：`.spec/tasks/auth-cross-domain-bridge.md`
- 反向协议：[`external-auth-handoff.md`](./external-auth-handoff.md)
- `uat_` 白名单：[`user-access-token.md`](./user-access-token.md)
- 契约原文：lumio-codex `docs/upstream/lumioapi-portal-integration.md`（改动一）
- 代码：`backend/internal/handler/auth_handler.go`（`Bridge`）、`backend/internal/server/routes/auth.go`、`frontend/src/utils/authBridge.ts`、`frontend/src/views/auth/BridgeView.vue`、`frontend/src/stores/auth.ts`、`frontend/src/router/index.ts`
