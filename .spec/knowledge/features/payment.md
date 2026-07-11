---
name: payment
description: Sub2API 内置支付系统的配置与流程——服务商接入、Webhook、订单状态、外部支付集成 Admin API 时查这篇。
metadata:
  type: doc
  level: L2
  status: 已上线
---

# 支付系统

简介：Sub2API 内置支付系统，支持用户自助充值，无需部署独立支付服务。支持 EasyPay、Mapay、支付宝官方、微信官方、Stripe 五类服务商，多实例负载均衡，支付成功后自动充值到用户余额。

## 背景 / 目标

让站点无需外挂独立支付服务即可完成用户自助充值。全部配置统一在 Sub2API 管理后台完成，支持多服务商、多实例、负载均衡与风控。

### 快速开始

1. 进入管理后台 → **设置** → **支付设置** 标签页
2. 开启 **启用支付**
3. 配置基本参数（金额范围、超时时间等）
4. 在 **服务商管理** 中添加至少一个服务商实例
5. 用户即可在前端页面进行充值

## 设计

### 支持的支付方式

| 服务商 | 支付方式 | 说明 |
|--------|---------|------|
| **EasyPay（易支付）** | 支付宝、微信支付 | 兼容易支付协议的第三方聚合支付 |
| **Mapay（码支付）** | 支付宝、微信支付 | 兼容码支付 EasyPay 风格接口，支持上游返回差异化实付金额 |
| **支付宝官方** | 桌面二维码扫码、移动端支付宝跳转 | 直接对接支付宝开放平台，桌面端返回二维码，移动端返回 WAP/唤起链接 |
| **微信官方** | Native 扫码、H5、公众号/JSAPI 支付 | 直接对接微信支付 APIv3，按终端环境自动分流 |
| **Stripe** | 银行卡、支付宝、微信支付、Link 等 | 国际支付，支持多币种 |

支付宝官方 / 微信官方、EasyPay、Mapay 可以同时作为后台服务商实例存在，但前台始终只展示 `支付宝`、`微信支付` 两个可见按钮。管理员需分别为这两个按钮选择唯一支付来源：官方、EasyPay 或 Mapay。官方渠道直接对接 API，资金直达商户账户，手续费更低；聚合支付接入门槛更低。

> **易支付服务商推荐**（均为兼容易支付协议的第三方聚合支付，按资金通道与结算方式选择，安全/稳定/合规请自行鉴别，本项目不背书）：
> - **国内渠道 / 人民币结算** — [ZPay](https://z-pay.cn/?uid=23808)：支付宝/微信官方 API 直连，手续费 1.6%，资金直达商家账户，T+1 自动到账；支持个人用户（无营业执照）每日 1 万元以内交易。链接含 [Sub2ApiPay](https://github.com/touwaeriol/sub2apipay) 原作者 [@touwaeriol](https://github.com/touwaeriol) 邀请码，介意可去掉。
> - **国际渠道 / USDT 或美元结算** — [启润支付 Kyren Topup](https://kyren.top/?code=SUB2API)：低门槛国际收款，支持国际版微信/支付宝，本地货币支付、美元结算。手续费微信 2%、支付宝 2.5%，提现 0.1%（最低 40 / 最高 150 美元），以 USDT 或美元到账；无资质审核、注册即用，但提现门槛略高。开户费 200 美元，通过含 [@Wei-Shaw](https://github.com/Wei-Shaw) 邀请码的链接注册可免开户费，介意可去掉。

### 系统设置

在管理后台 **设置 → 支付设置** 中配置。

#### 基本设置

| 设置项 | 说明 | 默认值 |
|--------|------|--------|
| 启用支付 | 启用或禁用支付系统 | 关闭 |
| 商品名前缀 | 支付页面显示的商品名前缀 | - |
| 商品名后缀 | 商品名后缀（如"元"） | - |
| 最低金额 | 单笔最低充值金额 | 1 |
| 最高金额 | 单笔最高充值金额（留空 = 不限制） | - |
| 每日限额 | 每用户每日累计充值上限（留空 = 不限制） | - |
| 订单超时时间 | 订单超时分钟数，至少 1 分钟 | 30 |
| 最大待支付订单数 | 同一用户最大并行待支付订单数 | 3 |
| 负载均衡策略 | 多服务商实例时的选择策略 | 轮询 |

#### 前台可见支付方式路由

当前版本对用户统一展示支付方式，不区分官方渠道还是易支付：

- **支付宝**：后台启用后，需额外指定该按钮路由到 `支付宝官方`、`易支付支付宝` 或 `码支付支付宝`
- **微信支付**：后台启用后，需额外指定该按钮路由到 `微信官方`、`易支付微信` 或 `码支付微信`
- 同一可见支付方式在同一时刻只能路由到一个来源
- 支付来源未选择时，即使对应按钮被开启，前台也不会暴露该支付方式

#### 负载均衡策略

| 策略 | 说明 |
|------|------|
| 轮询（round-robin） | 按顺序轮流分配到各服务商实例 |
| 最少金额（least-amount） | 优先分配到当日累计金额最少的实例 |

#### 取消频率限制

防止用户频繁创建并取消订单。可配置：启用限制（开关）、窗口模式（滚动 / 固定窗口）、时间窗口（窗口长度）、窗口单位（分钟 / 小时）、最大次数（窗口内允许的最大取消次数）。

#### 帮助信息

- **帮助图片**：充值页面显示的客服二维码等图片（支持上传，最大 1 MB）；在订阅页会显示在套餐列表上方作为定价说明图，并支持点击预览。
- **帮助文本**：充值页面显示的说明文字。

### 服务商配置

每种服务商需要不同凭证和参数，在 **服务商管理 → 添加服务商** 中选择类型后填写。

> **回调地址自动生成**：添加服务商时，异步回调地址（Notify URL）和同步跳转地址（Return URL）由系统根据站点域名自动拼接，无需手动填写，管理员确认域名正确即可。

#### EasyPay（易支付）

兼容任何 EasyPay 协议的支付服务商。必填：商户 ID（PID）、商户密钥（PKey）、API 地址。可选：支付宝通道 ID、微信通道 ID。

#### Mapay（码支付）

使用 `/xpay/epay/` 下的 EasyPay 风格接口。默认优先调用 `mapi.php` 获取二维码或支付链接；若服务商实例选择 `弹窗` 模式，则改用 `submit.php` 托管页面。必填：商户 ID（PID）、商户密钥（PKey）、API 地址（站点基础地址或 `/xpay/epay` 地址，系统自动补齐接口路径）。可选：通用通道 ID、支付宝通道 ID、微信通道 ID（专用通道优先于通用通道）。

> **精确金额要求**：Mapay 通过实际到账金额识别订单，创建订单后上游可能返回类似 `10.03` 的实付金额。前端用醒目动态提示展示该金额，用户必须一分不差支付；多付或少付都可能无法自动到账。

#### 支付宝官方

直接对接支付宝开放平台。移动端走支付宝手机网站支付跳转；桌面端优先使用当面付返回扫码串，若商户未开通当面付则回退到电脑网站支付，并将收银台链接同时返回给前端用于渲染二维码或直接打开支付页。必填：AppID、应用私钥（RSA2）、支付宝公钥。

#### 微信官方

直接对接微信支付 APIv3，支持 Native 扫码、H5、微信环境内公众号/JSAPI 支付。必填：AppID、商户号（MchID）、商户 API 私钥（PEM）、APIv3 密钥（32 位）、微信支付公钥（PEM）、微信支付公钥 ID、商户证书序列号。

#### Stripe

国际支付平台，支持多种支付方式和币种。必填：Secret Key（`sk_live_...` / `sk_test_...`）、Publishable Key（`pk_live_...` / `pk_test_...`）、Webhook Secret（`whsec_...`）。可选：

- **Currency**（配置键 `currency`）：实收币种，当前支持 `cny` / `usd`，默认 `cny`。
- **Balance Recharge Multiplier**（配置键 `balanceRechargeMultiplier`）：该 Stripe 实例的余额到账倍率。例如 `currency=usd` 且 `balanceRechargeMultiplier=7` 时，用户支付 `1.00 USD` 后账户余额到账 `7.00`。

### 服务商实例管理

同一种服务商可创建**多个实例**，实现负载均衡和风控：

- **多实例负载均衡** — 按轮询或最少金额策略分流订单
- **独立限额** — 每个实例可独立配置单笔最小/最大金额和每日限额
- **独立启停** — 可单独启用/禁用某实例，不影响其他实例
- **退款控制** — 每个实例可单独开启或关闭退款功能
- **支付方式** — 每个实例可选择支持的支付方式子集
- **排序** — 拖拽调整实例顺序

实例限额项：单笔最小金额、单笔最大金额、每日累计交易上限。负载均衡时，系统自动跳过超出限额的实例。

### Webhook 配置

支付回调是支付系统的核心环节。添加服务商时，系统自动根据站点域名拼接回调地址：

| 服务商 | 回调路径 |
|--------|---------|
| EasyPay | `https://your-domain.com/api/v1/payment/webhook/easypay` |
| Mapay | `https://your-domain.com/api/v1/payment/webhook/mapay` |
| 支付宝官方 | `https://your-domain.com/api/v1/payment/webhook/alipay` |
| 微信官方 | `https://your-domain.com/api/v1/payment/webhook/wxpay` |
| Stripe | `https://your-domain.com/api/v1/payment/webhook/stripe` |

EasyPay / 支付宝 / 微信的回调地址在添加服务商时自动填入。**Stripe 需手动设置**：登录 Stripe Dashboard → Developers → Webhooks → 添加端点填写回调地址 → 订阅事件 `payment_intent.succeeded`、`payment_intent.payment_failed` → 将生成的 Webhook Secret（`whsec_...`）填入服务商配置。

注意事项：回调地址必须 HTTPS（Stripe 强制，其他强烈推荐）；确保防火墙允许支付平台回调；系统自动签名验证防止伪造；支付成功后自动充值，无需人工干预。

### 支付流程

```
用户选择充值金额和支付方式
       │
       ▼
  创建订单 (PENDING)
  ├─ 校验金额范围、待支付订单数、每日限额
  ├─ 负载均衡选择服务商实例
  └─ 调用服务商获取支付信息
       │
       ▼
  用户完成支付
  ├─ EasyPay    → 扫码 / H5 跳转
  ├─ Mapay      → 优先内嵌二维码，必要时弹窗/跳转，显示上游要求的精确实付金额
  ├─ 支付宝官方  → 桌面扫码单（当面付优先，电脑网站支付回退）/ 移动端支付宝跳转
  ├─ 微信官方    → 桌面 Native 扫码 / 非微信 H5 / 微信内 JSAPI
  └─ Stripe     → Payment Element（银行卡/支付宝/微信等）
       │
       ▼
  支付回调验签 → 订单 PAID
       │
       ▼
  自动充值到用户余额 → 订单 COMPLETED
```

#### 订单状态

| 状态 | 说明 |
|------|------|
| `PENDING` | 待支付，等待用户完成支付 |
| `PAID` | 已支付，等待充值到账 |
| `COMPLETED` | 已完成，余额已到账 |
| `EXPIRED` | 已过期，超时未支付 |
| `CANCELLED` | 已取消，用户主动取消 |
| `FAILED` | 下单或支付前失败 |
| `FULFILLMENT_FAILED` | 支付已确认，但充值/开通履约失败；系统自动重试，仍失败后由管理员人工处理或重试 |
| `REFUND_REQUESTED` | 已申请退款 |
| `REFUNDING` | 退款处理中 |
| `REFUNDED` | 已退款 |

#### 超时与兜底

- 订单超时后，后台任务先查询上游支付状态再标记过期；用户实际已支付但回调延迟时，系统通过查询补单。
- 后台任务每 60 秒执行一次超时检查。
- 支付回调验签成功后先确认订单已支付，再执行充值/开通履约；履约失败不向支付服务商返回 500，也不在前端显示为"支付失败"。
- 履约失败内部重试节奏：首次立即执行，然后依次等待 5 秒、10 秒、1 分钟、2 分钟、5 分钟；仍失败则保留 `FULFILLMENT_FAILED` 供管理员人工处理。

#### 余额履约与兑换失败限流边界

- 用户手动兑换使用每用户独立的 Redis Key `redeem:ratelimit:<userID>`；累计 20 次无效兑换后限流，Key TTL 为 1 小时。一个用户的失败次数不会影响其他用户。
- 验签并校验支付金额后的余额履约走 service 包内的可信入口，只跳过手动兑换失败次数的读取和写入；该选项不暴露为 HTTP 请求参数。
- 支付履约仍执行兑换码存在性、状态/过期状态、Redis 分布式锁、数据库事务、兑换码已使用标记、余额更新、订单状态和幂等保护。重试时已使用的订单兑换码作为到账锚点，避免重复增加余额。
- 邀请返利仍由支付订单侧独立处理并通过订单审计/返利账本去重，不由普通兑换流程发放，避免履约重试重复返利。
- 部署前已经存在的 24 小时限流 Key 不会因代码常量变化自动缩短；需要按用户按需删除 Key 或调整现有 Key 的 TTL。

### 管理端集成 API

用于对接外部支付系统（如 `sub2apipay`）与 Sub2API 的 Admin API，覆盖支付成功后充值、用户查询、人工余额修正、前端购买页参数透传。

**基础地址**：生产 `https://<your-domain>`；Beta `http://<your-server-ip>:8084`。

**认证**（推荐请求头）：`x-api-key: admin-<64hex>`、`Content-Type: application/json`，幂等接口额外传 `Idempotency-Key`。管理员 JWT 也可访问 admin 路由，但服务间调用建议使用 Admin API Key。

**1) 一步完成创建并兑换** `POST /api/v1/admin/redeem-codes/create-and-redeem`

原子完成"创建兑换码 + 兑换到指定用户"。请求头需 `x-api-key` + `Idempotency-Key`。请求体：

```json
{ "code": "s2p_cm1234567890", "type": "balance", "value": 100.0, "user_id": 123, "notes": "sub2apipay order: cm1234567890" }
```

幂等语义：同 `code` 且 `used_by` 一致 → `200`；同 `code` 但 `used_by` 不一致 → `409`；缺少 `Idempotency-Key` → `400`（`IDEMPOTENCY_KEY_REQUIRED`）。

**2) 查询用户**（可选前置校验） `GET /api/v1/admin/users/:id`

**3) 余额调整**（已有接口） `POST /api/v1/admin/users/:id/balance`，支持 `set` / `add` / `subtract`：

```json
{ "balance": 100.0, "operation": "subtract", "notes": "manual correction" }
```

**4) 购买页 / 自定义页面 URL Query 透传**（iframe / 新窗口一致）：当 Sub2API 打开 `purchase_subscription_url` 或用户侧自定义页面 iframe URL 时，统一追加 `user_id`、`token`、`theme`（`light`/`dark`）、`lang`（如 `zh`/`en`）、`ui_mode`（固定 `embedded`）。示例：

```text
https://pay.example.com/pay?user_id=123&token=<jwt>&theme=light&lang=zh&ui_mode=embedded
```

**5) 失败处理建议**：支付成功与充值成功分状态落库；回调验签成功后立即标记"支付成功"；支付成功但充值失败的订单允许后续重试；重试保持相同 `code` 并使用新的 `Idempotency-Key`；内置支付会将已支付但履约失败的订单标记为 `FULFILLMENT_FAILED` 并按"立即、5 秒、10 秒、1 分钟、2 分钟、5 分钟"节奏重试，仍失败转人工。

**6) `doc_url` 配置建议**：查看 `https://github.com/Wei-Shaw/sub2api/blob/main/ADMIN_PAYMENT_INTEGRATION_API.md`；下载 `https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/ADMIN_PAYMENT_INTEGRATION_API.md`。

## 已决策

- 前台只展示 `支付宝` / `微信支付` 两个统一按钮，每个按钮在某时刻只路由到唯一来源。
- 服务间集成优先使用 Admin API Key 而非管理员 JWT。
- 充值由验签成功的 Webhook 自动触发，履约失败采用固定退避重试节奏后转人工。

## 待解决

- **订阅套餐**：内置支付暂不支持订阅套餐（计划中），Sub2ApiPay 原本支持。

## 相关

- 充值后余额闸门与发票：[[recharge-invoice-balance-gate]]
- 订阅计价口径：[[subscription-pricing]]
- **从 Sub2ApiPay 迁移**：原 [Sub2ApiPay](https://github.com/touwaeriol/sub2apipay)（独立 Next.js + PostgreSQL 服务）可迁移到内置支付——在后台启用支付并用相同凭证配置服务商、更新 Webhook 回调地址、确认新订单正常处理、停用 Sub2ApiPay。注意历史订单数据不会自动迁移，建议保留一段时间以便查询。
