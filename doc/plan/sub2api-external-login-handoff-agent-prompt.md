# Sub2API 外部登录回跳开发 Prompt

## 背景

外部项目 WebImageBuilder 不再自建登录注册 UI，而是点击右上角“登录”后直接跳转到 Sub2API：

```text
https://api.lumio.games/login?return_to=<WebImageBuilder当前页面URL>&handoff=1
```

WebImageBuilder 已经支持从回跳 URL 读取 Sub2API token：

- query: `?token=<access_token>` 或 `?access_token=<access_token>`
- hash: `#token=<access_token>` 或 `#access_token=<access_token>`

WebImageBuilder 拿到 token 后会调用自己的 `/api/sub2api/session`，再用 Sub2API `/auth/me` 获取邮箱和余额。

当前 Sub2API `/login` 的问题：

- 用户已经登录时，访问 `/login` 会被路由守卫带到 `/dashboard`，不会回到外部项目。
- 用户手动登录成功后，只执行内部 `router.push(redirect || '/dashboard')`，`redirect` 也是 Sub2API 内部路径，不会携带 token 回到外部项目。

## 目标

在 Sub2API 支持一个安全的外部登录 handoff：

1. 访问 `/login?return_to=<external-url>&handoff=1` 时，如果用户已经登录，立即回跳到 `return_to`，并附带当前 access token。
2. 如果用户未登录，保持现有登录页；邮箱密码登录成功后回跳到 `return_to` 并附带 access token。
3. 如果登录需要 2FA，2FA 成功后同样回跳。
4. 不影响现有内部登录、注册、OAuth、后台模式、dashboard 跳转逻辑。
5. 必须防止开放重定向和 token 泄露到不可信域名。

## 推荐实现

新增一个前端工具模块，例如：

```text
/Users/cui/Sites/sub2api/frontend/src/utils/externalAuthHandoff.ts
```

职责：

- 读取 route query 中的 `handoff=1` 和 `return_to`。
- 校验 `return_to`：
  - 必须是绝对 `http:` 或 `https:` URL。
  - origin 必须在允许列表中。
  - 开发环境至少允许 `http://localhost:3000` 和 `http://127.0.0.1:3000`。
  - 生产环境允许列表建议从配置或 public settings 读取，例如 `external_auth_return_origins`。
- 构造外部回跳 URL。
  - 推荐用 hash 携带 token，避免 token 进入服务端 access log：

```text
<return_to>#token=<access_token>
```

  - 如果 `return_to` 已有 hash，例如 `#view=studio`，追加为：

```text
<return_to>#view=studio&token=<access_token>
```

  - 回跳前清理 `return_to` 内已有的 `token/access_token/refresh_token/expires_in/token_type` 参数，避免重复转发旧 token。
- 执行回跳时使用 `window.location.replace(url)`。
- 不要把 token 输出到 console、toast 或错误日志。

## 需要改的关键位置

### 1. 路由守卫处理“已经登录”

文件：

```text
/Users/cui/Sites/sub2api/frontend/src/router/index.ts
```

当前逻辑中已登录访问 `/login` 或 `/register` 会跳到 dashboard。需要在这之前增加判断：

- 如果 `to.path === '/login'`
- 且 `handoff=1`
- 且 `return_to` 合法
- 且 `authStore.isAuthenticated && authStore.token`

则执行外部 handoff，直接回到 `return_to#token=<authStore.token>`，不要再进 `/dashboard`。

### 2. 邮箱密码登录成功后处理外部 handoff

文件：

```text
/Users/cui/Sites/sub2api/frontend/src/views/auth/LoginView.vue
```

在 `handleLogin()` 中，非 2FA 登录成功后：

- 如果当前 route query 是外部 handoff，使用返回的 `access_token` 或 `authStore.token` 回跳外部 URL。
- 否则保持现有逻辑：`router.push(redirect || '/dashboard')`。

### 3. 2FA 成功后处理外部 handoff

同一文件 `LoginView.vue`：

- 在 `handle2FAVerify()` 成功后，如果存在合法 handoff，回跳外部 URL。
- 否则保持现有内部跳转。

### 4. 可选：OAuth 登录链路

如果 Sub2API 登录页的 LinuxDo / WeChat / OIDC 入口会从 `/login` 发起 OAuth，请确保 `return_to` 和 `handoff=1` 不丢失：

- OAuth 发起前保存 handoff 上下文，或把它传入 OAuth state。
- OAuth callback 完成并拿到 `access_token` 后，如果存在合法 handoff，则回跳外部 URL。
- 如果本轮先只支持邮箱密码和 2FA，请在代码注释和测试中明确 OAuth 暂不支持，避免误以为已完成。

## 错误处理

- `return_to` 缺失或不合法：不要携带 token 跳转外部 URL，按原逻辑回 dashboard。
- origin 不在允许列表：显示清晰错误，例如“外部登录回跳地址未被允许”，并按原逻辑处理。
- 没有 access token：不回跳外部 URL，显示明确错误。

## 测试用例

请至少补这些测试：

1. `externalAuthHandoff` helper：
   - 合法 `return_to` + token 会生成 `<return_to>#token=<token>`。
   - `return_to` 已有 hash 时会追加 token，不覆盖原 hash 参数。
   - `return_to` 中已有 `token/access_token` 会先清理。
   - 非 http/https、相对路径、未允许 origin 都返回 invalid。

2. 路由守卫：
   - 已登录用户访问 `/login?handoff=1&return_to=http%3A%2F%2Flocalhost%3A3000%2F`，不会跳 `/dashboard`，而是触发外部 handoff。
   - 已登录用户访问普通 `/login`，仍保持原有 dashboard 行为。
   - 未登录用户访问 handoff 登录页，仍能看到登录页。

3. `LoginView.vue`：
   - 邮箱密码登录成功并带合法 handoff 时，调用 `window.location.replace()` 到外部 URL，URL 内包含 token。
   - 邮箱密码登录成功但无 handoff 时，仍跳 `/dashboard` 或内部 `redirect`。
   - 2FA 登录成功并带合法 handoff 时，调用外部回跳。
   - `return_to` 不合法时不向外部发送 token，并显示明确错误或回退内部跳转。

4. 回归测试：
   - 现有登录失败提示不变。
   - 现有内部 `redirect=/dashboard` 或其他内部路径不变。
   - backend mode 下非管理员登录逻辑不被破坏。

## 验收方式

本地 WebImageBuilder 运行在：

```text
http://localhost:3000
```

点击 WebImageBuilder 右上角“登录”后应打开：

```text
https://api.lumio.games/login?return_to=http%3A%2F%2Flocalhost%3A3000%2F&handoff=1
```

验收结果：

- 如果 Sub2API 已登录：应直接回到 `http://localhost:3000/#token=...`，随后 WebImageBuilder 右上角显示邮箱和余额。
- 如果 Sub2API 未登录：输入账号密码登录后应回到 `http://localhost:3000/#token=...`，随后 WebImageBuilder 右上角显示邮箱和余额。
- 如果需要 2FA：完成 2FA 后同样回到 WebImageBuilder。
