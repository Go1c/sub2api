# 用户 Access Token API

> 与个人资料「访问令牌」为**同一功能**：一个 `uat_` 可管密钥，并只读查日志 / 钱包余额 / 订阅额度。

Base：`https://api.lumio.games`
鉴权：`Authorization: Bearer <token>`

| Token | 用途 |
|-------|------|
| `uat_...` | 管理自己的 API Key + 只读：使用日志、钱包余额、订阅额度 |
| `sk-...` | 调模型 |

---

## 第一步：创建 Access Token

**网页：** 登录 → **个人资料** → 滚到最底 → **访问令牌** → 填名称/天数（默认 7，最长 30）→ 创建 → **立刻复制** `uat_...`（只显示一次）

**或接口（需登录 JWT）：**

```bash
curl -X POST https://api.lumio.games/api/v1/user/access-tokens \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"name":"ci","expires_in_days":7}'
```

管理：`GET/POST /api/v1/user/access-tokens`，`DELETE /api/v1/user/access-tokens/:id`（仅 JWT，`uat_` 调会 403）

---

## 第二步：用 uat_ 管理 Key / 只读查账

```bash
UAT=uat_xxx

# 查分组
curl -H "Authorization: Bearer $UAT" \
  https://api.lumio.games/api/v1/groups/available

# 建 Key
curl -X POST -H "Authorization: Bearer $UAT" \
  -H "Content-Type: application/json" \
  -d '{"name":"gpt-50","group_id":2,"quota":50}' \
  https://api.lumio.games/api/v1/keys

# 钱包余额（profile 中的 balance / frozen_balance）
curl -H "Authorization: Bearer $UAT" \
  https://api.lumio.games/api/v1/user/profile

# 使用日志
curl -H "Authorization: Bearer $UAT" \
  "https://api.lumio.games/api/v1/usage?page=1&page_size=20"

# 本人钱包扣款流水（uat_ 只读，不能扣款）
curl -H "Authorization: Bearer $UAT" \
  "https://api.lumio.games/api/v1/user/balance/transactions?page=1&page_size=20"

# 订阅额度 / 进度
curl -H "Authorization: Bearer $UAT" \
  https://api.lumio.games/api/v1/subscriptions/summary
curl -H "Authorization: Bearer $UAT" \
  https://api.lumio.games/api/v1/subscriptions/progress
```

| 字段（建 Key） | 说明 |
|------|------|
| `name` | 必填 |
| `group_id` | 来自 available |
| `quota` | USD 额度，0=不限 |

### 白名单（同一 token 的权限范围）

| 路径 | 说明 |
|------|------|
| `/keys*` | 管理自己的 API Key |
| `/groups/available`、`/groups/rates` | 可选分组与倍率 |
| `GET /user/profile` | 钱包余额（`balance` / `frozen_balance` 等） |
| `GET /usage*`、`POST /usage/dashboard/api-keys-usage` | 使用日志 / 统计；分销增量对账用 `GET /usage/export`，见 [reseller-usage-export.md](./reseller-usage-export.md) |
| `GET /user/balance/transactions*` | 本人成功扣款流水；不含扣款权限 |
| `GET /subscriptions`、`/active`、`/progress`、`/summary` | 订阅额度与进度 |

其它路径 403。不开放：改资料、管理 access token、重置周限、支付、admin。

---

## 第三步：用 sk_ 调模型

```bash
curl -X POST https://api.lumio.games/v1/chat/completions \
  -H "Authorization: Bearer $SK" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}'
```

---

## 注意

- `uat_` 不能调模型；`sk-` 才行
- 只能操作自己的资源（Key / 日志 / 余额 / 订阅）
- `uat_` 明文只返回一次；丢了就撤销重建
