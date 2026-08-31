---
status: completed
title: 模型广场增加视频计费展示（480p/720p/1080p）
---

# 模型广场增加视频计费展示（480p / 720p / 1080p）

模型广场后台候选/自定义模型的展示计费覆盖目前只有 token / per_request / image（图片分档 1K/2K/4K）。管理员要把 `grok-imagine-video` 一类模型改成视频计费，并按 **480P / 720P / 1080P** 三个档填展示单价。前台 `/model-market` 左侧「计费类型」筛选也要多一个「视频计费」，卡片按已填分档展示。

这是**展示覆盖**，只影响模型广场公开输出，不改网关真实计费。分组级 `video_price_480p/720p/1080p` 已经存在，不要改 GroupsView / BillingService。

## 做什么

镜像现有 image 分档实现，新增 `billing_mode = video`：

1. 后台「系统设置 → 模型广场」候选行和自定义模型：计费下拉增加「按次视频计费」，选中后出现 480P / 720P / 1080P 三个单价输入；空档不保存、不展示。
2. `PUT /api/v1/admin/settings/model-market` 接受并持久化该覆盖；`GET` 公开/管理接口原样回读。
3. 前台 `/model-market`：计费类型筛选增加「视频计费」；视频模型卡片按已配置分档展示单价（单位「/ 秒」或等价「/ s」），未填分档不出现。
4. 覆盖仍只作用于广场展示。

## 涉及范围

- 后端：`backend/internal/service/model_market.go`、`backend/internal/service/model_market_test.go`
- 前台公开页：`frontend/src/views/ModelMarketView.vue`、`frontend/src/views/__tests__/ModelMarketView.spec.ts`
- 后台设置：`frontend/src/views/admin/SettingsView.vue`
- 常量已有 `BILLING_MODE_VIDEO = 'video'`：`frontend/src/constants/channel.ts`
- i18n：`frontend/src/i18n/locales/zh.ts`、`zh-Hant.ts`、`en.ts` 的 `admin.settings.modelMarket.*`
- 知识：`.spec/knowledge/features/model-market.md`

## 实现约束

- `tier_label` 存网关口径：`480p` / `720p` / `1080p`（`VideoBillingResolution*`）。UI 文案可写 480P / 720P / 1080P。
- 分档复用 `pricing.intervals[].tier_label` + `per_request_price`，与图片 1K/2K/4K 同一结构。
- 后端 `isModelMarketBillingMode` 必须接受 `video`，否则保存会被拒、公开覆盖也不会生效。
- 空分档不写入 intervals；公开页不展示空档。
- 不要改 GroupsView、渠道定价、网关扣费。
- 最小改动，不顺手重构。

## 验收标准

- [x] 候选模型可把计费切到 video，填写任意 1–3 个分辨率单价并保存；刷新后仍在。
- [x] 自定义模型同样可配 video 三分档。
- [x] `GET /api/v1/model-market/public` 对 video 覆盖返回 `billing_mode=video` 和对应 intervals；未填分档不出现。
- [x] 非法 `billing_mode` 仍 400。
- [x] 前台筛选出现「视频计费」，计数等于 video 模型数；点选只显示 video 模型。
- [x] 视频卡片按 480P/720P/1080P 展示已填单价，不显示 token 的折扣/订阅倍率行。
- [x] 图片 1K/2K/4K 行为不变。
- [x] 后端 `go test -tags=unit ./internal/service/ -run ModelMarket`
- [x] 前端相关 vitest + `pnpm typecheck`
- [x] `model-market.md` 写明 video 分档覆盖。

## 依赖

无
