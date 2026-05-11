# 登录失败后邮箱验证码二次校验方案

## 背景

当前 `/api/v1/auth/login` 已有 IP 级 Redis 限流，但它只能限制单个来源的高频请求。若攻击者知道用户邮箱，可以通过低频或分布式方式持续尝试密码。需要增加账号维度的失败保护，同时避免把邮箱验证码变成免密码登录或邮件轰炸入口。

## 目标

- 所有邮箱密码登录用户适用，不只管理员。
- 同一账号连续输错密码 5 次后，账号进入邮箱验证码二次校验模式。
- 二次校验模式持续 15 分钟；成功登录后立即清除该状态和失败计数。
- 二次校验模式下必须同时满足密码正确和邮箱验证码正确。
- 邮箱验证码不自动发送，必须由用户主动点击发送，并受现有限流保护。
- 对外错误信息避免泄露账号是否存在、是否被保护等敏感状态。

## 非目标

- 不做“只靠邮箱验证码登录”。
- 不替代 TOTP 2FA。已启用 TOTP 的用户仍需在密码和邮箱验证码通过后完成 TOTP。
- 不改 API Key 网关鉴权。
- 不改其他前端入口；本功能落在上游后端和主 Vue 前端登录页。

## 推荐交互

普通登录流程保持不变：

1. 用户输入邮箱和密码。
2. 密码错误时，后端记录该邮箱的连续失败次数。
3. 第 5 次失败后，后端返回一个需要邮箱验证码的状态。
4. 前端展示邮箱验证码输入框和“发送验证码”按钮。
5. 用户再次提交邮箱、密码、邮箱验证码。
6. 后端先校验密码，再校验邮箱验证码，再继续现有 TOTP 判断。

二次校验模式下，如果密码仍错误，仍返回通用登录失败，不消耗邮箱验证码。

## 后端设计

新增 Redis 状态，使用邮箱规范化值作为账号维度 key：

- `login_failures:{email}`：连续密码失败计数，TTL 15 分钟。
- `login_email_challenge:{email}`：是否要求邮箱验证码，TTL 15 分钟。

登录接口请求体增加可选字段：

```json
{
  "email": "user@example.com",
  "password": "password",
  "email_code": "123456"
}
```

登录接口响应在需要邮箱验证码时返回业务错误码，例如：

```json
{
  "error": {
    "code": "EMAIL_CODE_REQUIRED",
    "message": "additional verification required"
  }
}
```

后端行为：

- 用户不存在时不创建账号维度 key，仍返回通用登录失败。
- 密码错误时记录失败次数；达到 5 次后设置 challenge key。
- challenge key 存在且密码正确但邮箱验证码缺失或错误时，拒绝登录。
- challenge key 存在且密码正确、邮箱验证码正确时，清除失败计数和 challenge key。
- 普通密码登录成功时清除失败计数和 challenge key。
- Redis 不可用时采用 fail-close，拒绝登录并返回服务暂不可用，避免保护失效。

## 邮箱验证码

复用现有邮箱验证码能力，但登录二次校验应有独立用途或 key 前缀，避免和注册、找回密码、TOTP 绑定互相消费验证码。

发送接口建议新增：

```text
POST /api/v1/auth/login/send-email-code
```

请求只需要邮箱。接口行为：

- 仅当该邮箱当前处于 `login_email_challenge` 状态时发送。
- 返回通用成功响应，即使邮箱不存在或未处于 challenge 状态，也不暴露账号状态。
- 受 IP 级限流和账号级冷却限制，防止邮件轰炸。

## 前端设计

上游 `frontend/` 登录页增加邮箱验证码状态：

- 正常情况下不显示验证码输入框。
- 收到 `EMAIL_CODE_REQUIRED` 后显示邮箱验证码输入框和发送按钮。
- 再次提交时带上 `email_code`。
- 若登录后还需要 TOTP，继续使用现有 TOTP 弹窗。

当前只有主前端 `frontend/` 作为本功能入口。

## 测试

后端单元测试：

- 连续 4 次错误仍只返回普通登录失败。
- 第 5 次错误进入邮箱验证码模式。
- challenge 模式下密码正确但验证码缺失会拒绝。
- challenge 模式下密码错误不会消耗验证码。
- challenge 模式下密码和验证码都正确后登录成功，并清空 Redis 状态。
- Redis 不可用时登录保护 fail-close。

前端测试：

- 登录接口返回 `EMAIL_CODE_REQUIRED` 后展示邮箱验证码区域。
- 提交时携带 `email_code`。
- 邮箱验证码通过后仍能进入现有 TOTP 流程。

验证命令：

```bash
cd backend && go test ./internal/service ./internal/handler ./internal/server/routes
cd frontend && pnpm typecheck && pnpm build
```
