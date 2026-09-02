---
name: external-auth-handoff
description: 外部应用把用户带到 Sub2API 登录、登录成功后把 access token 回跳给外部应用的接入说明与实现 / 接外部登录或排查 token 回跳时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 外部登录回跳接入（External Auth Handoff）

简介：Sub2API 支持外部应用（如 WebImageBuilder）不自建登录注册 UI，点击「登录」跳到 Sub2API 登录页，登录成功后把当前 access token 通过 URL hash 回跳给外部应用；外部应用拿 token 调 `/api/v1/auth/me` 获取邮箱、头像和实时余额。

## 背景 / 目标

外部项目 WebImageBuilder 不再自建登录注册 UI，改为点击「登录」直接跳转到 Sub2API。原 Sub2API `/login` 的问题：

- 用户已登录访问 `/login` 会被路由守卫带到 `/dashboard`，不会回到外部项目。
- 手动登录成功后只执行内部 `router.push(redirect || '/dashboard')`，`redirect` 也是内部路径，不会携带 token 回外部项目。

目标：提供安全的外部登录 handoff，覆盖「已登录直接回跳」「邮箱密码登录后回跳」「2FA 成功后回跳」，且不影响现有内部登录/注册/OAuth/dashboard 逻辑，并防止开放重定向和 token 泄露。

**适用场景**：外部应用不自建登录 UI / 只需复用 Sub2API 登录态 / 需要读取当前用户邮箱和动态余额。

**当前覆盖**：
- 用户已登录时访问 `/login?handoff=1&return_to=...`，直接回跳外部应用。
- 用户未登录时，邮箱密码登录成功后回跳。
- 用户开启 2FA 时，2FA 成功后回跳。

**当前未覆盖**：LinuxDo、WeChat、OIDC 等 OAuth 登录链路的 handoff 上下文传递。OAuth 登录仍按原内部流程处理（见「待解决」）。

## 设计

### 交互面：登录入口与回跳格式

外部应用点击「登录」跳转到：

```text
https://<sub2api-domain>/login?handoff=1&return_to=<urlencoded-external-url>
```

示例：`https://api.lumio.games/login?handoff=1&return_to=http%3A%2F%2Flocalhost%3A3000%2F`

`return_to` 必须是绝对 URL，协议只能是 `http:` 或 `https:`。

登录成功后用 `window.location.replace()` 回跳，token 放在 hash：

```text
<return_to>#token=<access_token>
```

若 `return_to` 已有 hash（如 `#view=studio`），追加而非覆盖：`http://localhost:3000/#view=studio&token=<access_token>`。

回跳前先清理 `return_to` 中已有的敏感参数，避免旧 token 被继续转发：`token` / `access_token` / `refresh_token` / `expires_in` / `token_type`。

### 允许来源（防开放重定向）

`return_to` 的 origin 必须在允许列表里。默认允许 `http://localhost:3000` 和 `http://127.0.0.1:3000`。生产或其他环境在前端构建时配置：

```bash
VITE_EXTERNAL_AUTH_RETURN_ORIGINS=https://webimagebuilder.example.com,https://app.example.com
```

这是前端构建环境变量，修改后需重新构建并部署前端。（Prompt 中亦建议生产允许列表从配置或 public settings 读取，如 `external_auth_return_origins`。）

### 实现面：前端改动点

推荐新增工具模块 `frontend/src/utils/externalAuthHandoff.ts`，职责：

- 读取 route query 的 `handoff=1` 和 `return_to`。
- 校验 `return_to`：必须是绝对 `http:`/`https:` URL；origin 在允许列表中；开发环境至少允许两个 localhost origin。
- 构造回跳 URL（hash 携带 token、追加而非覆盖已有 hash、清理已有 token 类参数）。
- 用 `window.location.replace(url)` 回跳。
- 不要把 token 输出到 console / toast / 错误日志。

需要改的关键位置：

1. **路由守卫**（`frontend/src/router/index.ts`）：已登录访问 `/login` 默认跳 dashboard。在此之前增加判断——若 `to.path === '/login'` 且 `handoff=1` 且 `return_to` 合法 且 `authStore.isAuthenticated && authStore.token`，则执行外部 handoff 回跳 `return_to#token=<authStore.token>`，不进 `/dashboard`。
2. **邮箱密码登录**（`frontend/src/views/auth/LoginView.vue` 的 `handleLogin()`）：非 2FA 登录成功后，若当前是外部 handoff，用返回的 `access_token` 或 `authStore.token` 回跳；否则保持 `router.push(redirect || '/dashboard')`。
3. **2FA 登录**（同文件 `handle2FAVerify()`）：成功后存在合法 handoff 则回跳，否则保持内部跳转。
4. **OAuth（可选 / 暂未做）**：若 `/login` 的 LinuxDo/WeChat/OIDC 入口发起 OAuth，需保证 `return_to` 和 `handoff=1` 不丢失（OAuth 发起前保存上下文或传入 OAuth state，callback 拿到 token 后回跳）。本轮先只支持邮箱密码和 2FA，需在代码注释和测试中明确 OAuth 暂不支持。

### 外部应用读取用户信息

外部应用拿到 hash 中的 `token` 后调用：

```http
GET https://<sub2api-domain>/api/v1/auth/me
Authorization: Bearer <access_token>
```

响应是标准 Sub2API envelope：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 123,
    "email": "user@example.com",
    "username": "user",
    "avatar_url": "https://cdn.example.com/avatar.png",
    "role": "user",
    "balance": 12.34,
    "run_mode": "standard"
  }
}
```

外部应用应以 `/auth/me` 返回的 `data.balance` 作为当前实时余额。

## 已决策

- **token 走 hash 不走 query**——避免 token 进入服务端 access log。
- **origin 允许列表 + 绝对 http/https 校验**——防开放重定向与 token 泄露到不可信域名。
- **回跳前清理已有 token 类参数**——避免旧 token 被重复转发。
- **用 `window.location.replace()`**——回跳不留历史记录。
- **不合法 / 缺失 / origin 不允许 / 无 token 时不携带 token 回跳**——回退原内部跳转或显示明确错误，绝不向外部发 token。

## 待解决

- **OAuth 登录链路的 handoff 传递**（LinuxDo / WeChat / OIDC）尚未覆盖，OAuth 仍走原内部流程。

### 安全要求

- 只通过 HTTPS 使用生产环境 handoff。
- 外部应用不要把 token 写入服务端访问日志、错误日志、埋点或第三方统计。
- 外部应用读取 hash token 后应尽快清理浏览器地址栏中的 token。
- `return_to` 缺失/不合法/origin 不在允许列表时，Sub2API 不携带 token 跳转，继续原内部登录跳转。

### 错误处理

- `return_to` 缺失或不合法：不带 token 跳转，按原逻辑回 dashboard。
- origin 不在允许列表：显示清晰错误（如「外部登录回跳地址未被允许」），按原逻辑处理。
- 没有 access token：不回跳外部 URL，显示明确错误。

### 建议测试用例

1. `externalAuthHandoff` helper：合法 `return_to` + token 生成 `<return_to>#token=<token>`；已有 hash 时追加不覆盖；已有 `token/access_token` 先清理；非 http/https、相对路径、未允许 origin 返回 invalid。
2. 路由守卫：已登录 + handoff 触发回跳不跳 dashboard；已登录普通 `/login` 仍跳 dashboard；未登录 handoff 仍显示登录页。
3. `LoginView.vue`：邮箱密码 + 合法 handoff → `window.location.replace()` 到含 token 的外部 URL；无 handoff 仍跳 dashboard/内部 redirect；2FA + 合法 handoff 回跳；`return_to` 不合法不发 token 并显示错误或回退。
4. 回归：登录失败提示不变；内部 redirect 不变；backend mode 下非管理员登录逻辑不破坏。

### 验收方式

本地 WebImageBuilder 运行在 `http://localhost:3000`，从其登录入口跳到 `https://api.lumio.games/login?handoff=1&return_to=http%3A%2F%2Flocalhost%3A3000%2F`，预期：

- 已登录：直接回到 `http://localhost:3000/#token=...`。
- 未登录：邮箱密码登录成功后回到 `http://localhost:3000/#token=...`。
- 需 2FA：完成 2FA 后回到 `http://localhost:3000/#token=...`。
- WebImageBuilder 调 `/api/v1/auth/me` 后显示邮箱和余额。

## 相关

- 前端：`frontend/src/utils/externalAuthHandoff.ts`、`frontend/src/router/index.ts`、`frontend/src/views/auth/LoginView.vue`
- 构建变量：`VITE_EXTERNAL_AUTH_RETURN_ORIGINS`
- 后端接口：`GET /api/v1/auth/me`
- 反向场景（主站已登录、把 token 交给本站控制台）：[`auth-cross-domain-bridge.md`](./auth-cross-domain-bridge.md)
