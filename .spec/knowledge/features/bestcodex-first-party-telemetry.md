---
name: bestcodex-first-party-telemetry
description: BestCodex 第一方遥测：公开 ingest、管理 stats、注册/登录权威事件、账号 first/last touch 衔接与 Umami 对照口径
metadata:
  type: doc
  level: L2
  status: 已交付
---

# BestCodex 第一方遥测

简介：Sub2API 承接 lumio-codex 已上报的第一方遥测。客户端事件是观察；注册 / 登录 / 首次启动等权威计数以本仓库事件表为准，不以 Umami PV 或 `users.last_login_at` 为准。

## 背景 / 目标

lumio-codex 已向 `POST /api/v1/telemetry/events` 上报。服务端原先没有路由，客户端静默失败。需要：

- 公开 ingest，未登录可写，失败不得挡住注册 / 登录
- 管理端只读聚合 `GET /api/v1/telemetry/stats`
- 账号 first-touch 只写一次、last-touch 更新，桌面新 `bc_aid` 能继承落地页 campaign（如 t1299）
- 注册成功 / 登录成功由 AuthService 再写一条 `ingest_source=server` 的权威行

不改 `users.signup_source`、GeoBlock、跟踪脚本、前端 SPA。

## 设计

### 契约

- `POST /api/v1/telemetry/events`：无需 JWT。Body `{ "event": string, "ts": number, "props": { [k: string]: string } }`。成功信封 `data` 仅为 `{ "accepted": true }`，不回显 props。非法 event / 非法 JSON → 400。重复成功事件仍 200 accepted。
- 非法或过期 Bearer 当作匿名，不 401。
- IP 限流 120/min，Redis 故障 fail-open。
- `GET /api/v1/telemetry/stats`：精确路径，`adminAuth` + `auditLog`。匿名 401。`from`/`to` 为必填 unix ms；可选 `client_source`、`campaign`（过滤 `first_touch_campaign`）、`event`。缺参或 from>to → 400；跨度上限 90 天。
- stats `data.authority` 固定 `first_party_ingest`。`event_count = COUNT(*)`，`unique_attribution_ids = COUNT(DISTINCT attribution_id) WHERE attribution_id <> ''`。指定 `event` 时补零行；未指定时只返回 count>0。
- HTTP 响应与 stats JSON **不得**出现 `user_id`、email、token、原始 props。`telemetry_events.user_id` 仅内部可空 FK，供衔接；stats SQL 不 SELECT 该列。

### 白名单

事件：`signup_page_view`、`login_page_view`、`verify_code_*`、`auth_register_*`、`auth_login_submit` / `_2fa_required` / `_success` / `_failure`、`download_dialog_open`、`download_start`、`app_first_launch`、`app_login_success`、`codex_setup_success`、`claude_setup_success`、`app_ready`。

props 只留：`client_source`（`bestcodex_web` | `bestcodex_desktop_codex` | `bestcodex_desktop_claude` | `unknown`）、`route`（pathname，最长 128，去掉 query/hash，绝对 URL 只留 path，含 `..` 丢弃）、`auth_method`（email|2fa）、`platform`（mac_arm|mac_intel|windows）、`destination`（cdn|github）、`error_code`（`UNKNOWN` 或 `^(AUTH|ACCOUNT|KEY|SERVICE)_[A-Z0-9_]{1,48}$`）、`attribution_id`（`^bc_[a-z0-9]{16,32}$`）、first/last touch `source/medium/campaign`（`^[a-z0-9][a-z0-9._-]{0,31}$`，含 `sb.sb` / `referral` / `t1299` / `direct` / `none`）。

丢弃 email / password / token / user_id / url / invite / fingerprint / code 以及未知键。

`ts` 为 unix ms；0 / 缺失 / 超过 7 天前或未来 1 小时 → now；落在 1e9–1e10 的秒级时间戳 ×1000。

### 去重

成功类：`auth_register_success`、`auth_login_success`、`app_first_launch`、`app_login_success`、`*_setup_success`、`app_ready`。

- 终身唯一：`auth_register_success`、`app_first_launch` → `event:attribution_id`
- 其余按 2 分钟桶（`occurred_at` unix ms / 120000）
- 无 `attribution_id` 但有 `user_id` 时用 `u:{id}`
- unique 冲突视为 accepted duplicate
- 服务端权威行与随后带 Bearer 的客户端成功事件按 `user_id` / `attribution_id` 合并：补空的 campaign / attribution，不新增第二行

### CORS（仅 ingest）

只对 `OPTIONS`/`POST` `/api/v1/telemetry/events` 额外放行，**不**改 `/api/v1/auth/*` 的 CORS，也**不**把 `bestcodex.app` 加进全局 `cors.allowed_origins`。

- 浏览器：`https://bestcodex.app`、`https://www.bestcodex.app`、`https://*.bestcodex.app`（预发）
- 本地：`localhost` / `127.0.0.1` 任意端口的 http/https
- 桌面：缺 Origin、`Origin: null`、`tauri://localhost`、`https://tauri.localhost`、`https://asset.localhost`
- 桌面事件不靠 Origin 才能入库：POST 即使 Origin 不在白名单也继续 ingest；可用 `client_source=bestcodex_desktop_codex|bestcodex_desktop_claude`，登录后可带 Bearer 衔接账号
- ingest 持久化失败 fail-open（仍 200 `accepted`），避免被前端当成注册/登录失败
- `GET /api/v1/telemetry/stats` **不对** bestcodex.app 开放（内部/管理员）
- Caddy 只反代，不另设 CORS

### first / last touch 与桌面衔接

合法 Bearer：内部写入 `user_id`，upsert `user_first_party_attribution`。first_touch 与 `first_attribution_id` 空则写、非空不改；last_touch 与 `last_attribution_id` 随新值更新。

`app_first_launch` / `app_login_success` / `*_setup_success` / `app_ready`：若请求 first_touch 为空，从账号拷贝，使桌面新 `bc_aid` 继承落地页 campaign（例如 t1299）。

### 服务端权威事件

- `RecordSuccessfulLogin` 在 `touchUserLogin` 之后 best-effort `auth_login_success`（`ingest_source=server`），拷贝已存 attribution。
- `postAuthUserBootstrap` 且 `touchLogin==true` 时 best-effort `auth_register_success`。
- 遥测失败只打日志，不失败注册 / 登录。权威次数看事件表行，不能只改 `last_login_at`。

### 实现面

独立包 `backend/internal/telemetry/`，对照 `internal/checkin`：raw SQL + `*sql.DB`，不改 Ent。迁移 `937_bestcodex_first_party_telemetry.sql`。Auth 通过 `AuthService.SetFirstPartyTelemetry` 注入，不改 `NewAuthService` 签名。

## Umami 对照（文档配方，不改脚本）

BestCodex 站点脚本：`https://data.lumio.games/script.js`，website id `d7056c07-8220-4d1e-9c01-a6ac15aa62ac`。仓库不能登录托管看板；运营需在 Umami UI 自行配置。

- 按自定义事件名过滤；按 `client_source`、`route`、`auth_method`、`platform`、`destination`、`error_code`、`attribution_id`、first/last touch 字段拆解。
- Funnel：`signup_page_view` → `verify_code_*` → `auth_register_submit` → `auth_register_success`
- Funnel：`login_page_view` → `auth_login_submit` → `auth_login_2fa_required` → `auth_login_success`
- Funnel：`download_dialog_open` → `download_start`（点击，不是安装）
- PV 不是转化；打开 `/#downloads` 或 `/login` 不等于注册 / 下载
- 权威 register/login/first-launch 以 Sub2API `telemetry_events` 为准，Umami 只作旁路观察

## 已决策

- 客户端事件是观察，权威计数来自事件表。
- 不改 `signup_source`，不用 `last_login_at` 当历史登录次数。
- `user_id` 只存内部表，永不出现在 HTTP / stats JSON。
- CORS 开口仅 ingest 路径。

## 待解决

- 运营在 Umami UI 按上文配方建看板与 funnel（仓库无法代做）。

## 相关

- 代码：`backend/internal/telemetry/`、`backend/internal/server/telemetry.go`、`backend/internal/server/middleware/cors.go`
- 迁移：`backend/migrations/937_bestcodex_first_party_telemetry.sql`
- 任务卡：`.spec/tasks/bestcodex-first-party-telemetry.md`
- 旁路跟踪：[`umami-public-tracking.md`](umami-public-tracking.md)（主站另一 website id，勿混）
