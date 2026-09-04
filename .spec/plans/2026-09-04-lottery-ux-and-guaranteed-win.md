---
status: completed
---

# 抽奖必中保证 + 转盘/公众号 UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan wave by wave (hosts without subagents: its Inline Fallback section). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 奖品数不少于剩余参与名额时每次抽奖必中；转盘固定 8 格；一套 HTTPS 海报用于抽奖前、中奖结果、中奖站内信。

**Architecture:** 中奖率在相位加权之后加硬保证 `remainingPrizes >= remainingSlots → p = 1`。视觉格子与 `prize_count` 脱钩。活动新增 `promo_text` / `promo_image_url`（仅 https），三处共用。

**Tech Stack:** Go (Gin/Ent)、Vue 3、Vitest、go test -tags=unit

## Global Constraints

- 不改 `backend/` / `frontend/` 以外的无关重构；fork 红线：不直推 main/publish。
- 图片只接受 `https://` 公开直链（方式 A / 备份 S3 公开域名），不存 data URI，不做自动 PutObject。
- 不验证微信是否真关注；未中奖不展示海报、不另发广告信。
- 前端收口：`pnpm typecheck && pnpm build`；后端：覆盖本卡的 `go test -tags=unit`。
- 用户未要求则不 commit。

---

### Task 1: 后端必中保证、8 格转盘、素材字段与中奖信

**Files:**
- Modify: `backend/ent/schema/lottery_campaign.go`
- Create: `backend/migrations/938_lottery_promo.sql`
- Modify: `backend/internal/service/lottery.go`, `lottery_service.go`, `lottery_service_test.go`
- Modify: `backend/internal/service/site_message_service.go`, `site_message_service_test.go`
- Modify: `backend/internal/repository/lottery_repo.go`
- Modify: `backend/internal/handler/dto/lottery.go`, `backend/internal/handler/admin/lottery_handler.go`
- Generate: `backend/ent/` via `go generate ./ent`

**Interfaces:**
- Consumes: 现有 Draw 事务、SiteMessageService.create
- Produces: `LotteryCampaign.PromoText`, `PromoImageURL`; `LotteryActiveCampaign` 同样字段；`SendLotteryPrize(..., promoText, promoImageURL string)`；`BuildLotterySegments` 恒为 8 格

- [x] **Step 1–4:** 失败测试 → 实现 → 生成 Ent → 单测绿

### Task 2: 前端转盘 UI、管理端填 HTTPS、站内信出图

**Files:**
- Modify: `frontend/src/types/index.ts`, `frontend/src/stores/lottery.ts`
- Modify: `frontend/src/components/lottery/LotteryDialog.vue`, `LotteryPromptManager.vue`
- Modify: `frontend/src/views/admin/LotteryView.vue` + i18n
- Modify: `frontend/src/views/user/SiteMessagesView.vue` + tests
- Knowledge: `.spec/knowledge/features/real-lottery.md`

- [x] **Step 1–4:** 失败测试 → 实现 → vitest + typecheck/build

## Ledger

- Task 1: complete
- Task 2: complete
