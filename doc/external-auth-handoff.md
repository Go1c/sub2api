# 外部登录回跳接入说明

Sub2API 支持外部应用把用户带到 Sub2API 登录页，并在登录成功后把当前 access token 回跳给外部应用。外部应用拿到 token 后可以调用 Sub2API `/api/v1/auth/me` 获取邮箱、头像信息和当前余额。

## 适用场景

- 外部应用不自建登录注册 UI。
- 外部应用只需要复用 Sub2API 登录态。
- 外部应用需要读取当前用户邮箱和动态余额。

当前实现覆盖：

- 用户已登录时访问 `/login?handoff=1&return_to=...`，直接回跳外部应用。
- 用户未登录时，邮箱密码登录成功后回跳外部应用。
- 用户开启 2FA 时，2FA 成功后回跳外部应用。

当前未覆盖：

- LinuxDo、WeChat、OIDC 等 OAuth 登录链路的 handoff 上下文传递。OAuth 登录仍按原内部流程处理。

## 外部应用登录入口

外部应用点击“登录”时跳转到：

```text
https://<sub2api-domain>/login?handoff=1&return_to=<urlencoded-external-url>
```

WebImageBuilder 本地开发示例：

```text
https://api.lumio.games/login?handoff=1&return_to=http%3A%2F%2Flocalhost%3A3000%2F
```

`return_to` 必须是绝对 URL，并且协议只能是 `http:` 或 `https:`。

## 回跳格式

登录成功后，Sub2API 使用 `window.location.replace()` 回跳到 `return_to`，并把 token 放到 hash 中：

```text
<return_to>#token=<access_token>
```

如果 `return_to` 已有 hash，会追加 token：

```text
http://localhost:3000/#view=studio
```

回跳为：

```text
http://localhost:3000/#view=studio&token=<access_token>
```

Sub2API 会在回跳前清理 `return_to` 中已有的这些敏感参数，避免旧 token 被继续转发：

- `token`
- `access_token`
- `refresh_token`
- `expires_in`
- `token_type`

## 允许来源

为了防止开放重定向和 token 泄露，`return_to` 的 origin 必须在允许列表里。

默认允许：

```text
http://localhost:3000
http://127.0.0.1:3000
```

生产或其他环境需要在前端构建时配置：

```bash
VITE_EXTERNAL_AUTH_RETURN_ORIGINS=https://webimagebuilder.example.com,https://app.example.com
```

这是前端构建环境变量，修改后需要重新构建并部署前端。

## 外部应用读取用户信息

外部应用拿到 hash 中的 `token` 后，调用：

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

## 安全要求

- 只通过 HTTPS 使用生产环境 handoff。
- 外部应用不要把 token 写入服务端访问日志、错误日志、埋点或第三方统计。
- 外部应用读取 hash token 后应尽快清理浏览器地址栏中的 token。
- 如果 `return_to` 缺失、不合法或 origin 不在允许列表中，Sub2API 不会携带 token 跳转外部地址，会继续原有内部登录跳转。

## 验收

本地 WebImageBuilder 运行在：

```text
http://localhost:3000
```

从 WebImageBuilder 登录入口跳转到：

```text
https://api.lumio.games/login?handoff=1&return_to=http%3A%2F%2Flocalhost%3A3000%2F
```

预期：

- 如果 Sub2API 已登录，直接回到 `http://localhost:3000/#token=...`。
- 如果 Sub2API 未登录，邮箱密码登录成功后回到 `http://localhost:3000/#token=...`。
- 如果需要 2FA，完成 2FA 后回到 `http://localhost:3000/#token=...`。
- WebImageBuilder 调用 Sub2API `/api/v1/auth/me` 后能显示邮箱和余额。
