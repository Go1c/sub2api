---
status: completed
title: BestCodex 第一方遥测 ingest / stats / 账号衔接
---

# BestCodex 第一方遥测

## 做什么

在 Sub2API（`api.lumio.games`）落地 BestCodex 第一方遥测：公开 `POST /api/v1/telemetry/events`、管理只读 `GET /api/v1/telemetry/stats`、注册/登录服务端权威写事件表、账号 first/last touch 衔接。lumio-codex 前端已按契约上报；服务端缺路由时客户端静默失败，不得挡注册/登录。

## 涉及范围

- 新包 `backend/internal/telemetry/`（对照 `internal/checkin`：SQL 仓库 + handler + module，不走 Ent）
- `backend/migrations/937_bestcodex_first_party_telemetry.sql`
- CORS：`backend/internal/server/middleware/cors.go`（仅 ingest 路径特判）
- 路由 / wire：`routes`、`router.go`、`http.go`、`cmd/server/wire_gen.go`
- 登录/注册权威写：`AuthService.RecordSuccessfulLogin`、`postAuthUserBootstrap`
- 知识：`knowledge/features/bestcodex-first-party-telemetry.md` + README 导航
- Umami：只写后台查询配方，不改跟踪脚本

## 验收标准

- [x] `POST /api/v1/telemetry/events` 未登录可写；非法 event 400；非法 props 键丢掉；响应不含 email/token/user_id/原始 props
- [x] 白名单事件/props 按任务卡校验；成功类事件按 attribution_id 去重
- [x] CORS：`https://bestcodex.app` 可 OPTIONS/POST ingest；缺 Origin / tauri Origin 不丢桌面事件；GET stats 不对 bestcodex.app 开放
- [x] `GET /api/v1/telemetry/stats` 需 admin；聚合 `event_count` + `unique_attribution_ids`；campaign 过滤 `first_touch_campaign`
- [x] `utm_campaign=t1299` 注册成功后 `stats?campaign=t1299` 的 `auth_register_success` 为 1/1
- [x] 同一登录 2FA 重试不让 `auth_login_success` 变成 2
- [x] `RecordSuccessfulLogin` 必须插事件行，不能只改 `last_login_at`
- [x] 不改 `signup_source`；不把 `last_login_at` 当历史次数；不改 GeoBlock / 商店上架
- [x] unit 测试覆盖 sanitize / CORS / dedup / stats 鉴权 / 非法 props；`go test -tags=unit` 相关包通过；`go vet -tags integration ./...` 通过

## 依赖

无
