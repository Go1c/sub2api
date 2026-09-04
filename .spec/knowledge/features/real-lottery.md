---
name: real-lottery
description: 后端驱动的多用户抽奖，中奖兑换码经站内信发放
metadata:
  type: doc
  level: L2
  status: 设计中
---

# 真实抽奖活动
简介：将仅前端的抽奖 mock 替换为后端驱动、多用户共享的抽奖系统；当唯一活跃活动还有名额时，所有已登录用户都会看到登录弹窗，中奖者通过站内信领取兑换码。

## 背景 / 目标
当前抽奖逻辑只存在于前端：

- `frontend/src/stores/lottery.ts` 把活动、抽奖状态和中奖者存在 `localStorage`。
- `frontend/src/components/lottery/LotteryPromptManager.vue` 读取本地状态并打开弹窗。
- `frontend/src/views/admin/LotteryView.vue` 在同一浏览器里创建 mock 活动。
- 没有任何抽奖相关的后端路由、仓储、迁移或 API 文件。

结果是管理员创建的活动只在管理员自己的浏览器里可见，其他用户看不到。

目标：在后端持久化抽奖活动、兑换码和抽奖记录。前端向后端询问是否应给当前已认证用户展示弹窗，并把抽奖请求 POST 给后端。后端拥有参与人数上限、重复抽奖防护、奖品分配和站内信发放。

技术栈：Go、Gin、Ent、PostgreSQL 迁移、Vue 3、Pinia、Axios、Vitest、Go tests。

## 设计

### 产品规则
- 系统最多只有一个活跃活动。
- 活跃活动仅在 `joined_count < max_participants` 时对用户出现。
- 已登录用户在以下全部满足时看到弹窗：有一个活跃活动、活动仍有名额、用户在该活动未抽过、用户在当前浏览器会话未关闭过该活动。
- 关闭弹窗只在 `sessionStorage` 存会话级 dismissal，不创建后端记录。
- 抽奖会创建后端记录；抽奖后该用户在任何浏览器都不再看到该活动。
- 中奖者收到包含兑换码的站内信。
- 结果弹窗**不展示兑换码**，只提示码已经通过站内信发放。
- 所有名额用完后，后端结束活动。
- 若奖品码在名额用完前耗尽，后端可保持活动活跃直到名额用尽，但之后的抽奖都不中。
- 创建新活动会结束任何已存在的活跃活动。
- 创建或激活活动要求站内信功能已启用，因为中奖发放依赖它。
- 转盘视觉格子与奖品数量脱钩：固定 8 格（奖品 / 谢谢参与），即使 `prize_count` 为 100 也不再画 100+ 格。中不中仍由后端概率决定。
- **必中保证：** 当 `remainingPrizes >= remainingSlots`（含 100 奖 / 100 人，以及前期有人未中导致剩余奖不少于剩余名额）时，中奖率固定为 100%，前期加权 / 后期衰减 / 充值加权都不得把它压低。只有剩余奖少于剩余名额时才按概率抽取。
- 每场活动可配一套关注引导素材：`promo_text` + `promo_image_url`（仅 `https://` 公开图链）。三处共用：抽奖前可跳过的关注页、中奖结果、中奖站内信。未配置则行为与旧版相同。
- 图片放在现有备份 S3 / R2 桶（建议 key 前缀 `lottery/`），管理端只粘贴该对象的公开 HTTPS 直链。不存 data URI，不把备份用的 attachment 预签名 URL 当 `<img src>`。

### 数据模型
新增三个 Ent schema 和一个 SQL 迁移（`backend/migrations/140_add_lottery.sql`）。

**`lottery_campaigns`**：`id`、`name varchar(120)`、`subtitle varchar(240)`、`status varchar(20)`（`active` / `finished`）、`prize_count`、`max_participants`、`joined_count default 0`、`winner_count default 0`、`early_boost_participant_percent`、`recharge_boost_cap_percent`、`promo_text varchar(240)`（可空）、`promo_image_url varchar(2048)`（可空，仅 https）、`created_by`、`created_at`、`updated_at`、`finished_at null`。索引：`status`、`created_at desc`。应用层不变量：只有一个活动可为 `active`，服务通过在创建新活跃活动前把现有活跃活动置 `finished` 来强制。

**`lottery_codes`**：`id`、`campaign_id`、`code varchar(128)`、`assigned_user_id null`、`assigned_draw_id null`、`assigned_at null`、`created_at`、`updated_at`。索引：`(campaign_id, code)` 唯一、`(campaign_id, assigned_at)`。

**`lottery_draws`**：`id`、`campaign_id`、`user_id`、`won boolean`、`lottery_code_id null`、`site_message_id null`、`result_label varchar(80)`、`created_at`。索引：`(campaign_id, user_id)` 唯一（跨设备和并发标签页防重复抽奖）、`(campaign_id, created_at)`、`(user_id, created_at)`。

### 后端服务
`LotteryService` 职责：

- 管理员生命周期：`CreateCampaign`（校验、确认站内信启用、结束旧活跃活动、创建活动和码行）、`ListCampaigns`、`GetCampaign`（含码与中奖详情）、`FinishCampaign`。
- 用户流程：`GetActiveForUser`（返回弹窗 payload）、`Draw`（事务内执行）。

**Draw 算法**（必须在数据库事务内）：
1. `FOR UPDATE` 锁定活动行。
2. 活动非 active 则拒绝。
3. `joined_count >= max_participants` 则拒绝。
4. 按 `(campaign_id, user_id)` 查已有抽奖；存在则返回 already-drawn。
5. 计算 `remainingPrizes = prize_count - winner_count`、`remainingSlots = max_participants - joined_count`。若 `remainingPrizes >= remainingSlots` 则中奖率 = 1；否则 `winProbability = remainingPrizes / remainingSlots`，再乘前期/后期系数、加充值加权，最后 clamp 到 `[0, 1]`。
6. 随机决定结果。
7. 若中奖，用 `FOR UPDATE SKIP LOCKED` 选一个未分配码，分配给用户，并创建站内信。
8. 创建 draw 行。
9. `joined_count` +1；仅中奖时 `winner_count` +1。
10. 若 `joined_count >= max_participants`，标记活动 finished。
11. 提交并返回结果。

若随机判定中奖但无码可用，按未中处理——防止数据损坏并保持事务安全。

测试注入确定性：`LotteryService` 持有 `now func() time.Time` 和 `randFloat func() float64`，测试用 `randFloat` 强制中 / 不中。

**站内信发放**：使用 `SiteMessageService.SystemSendToUser` 或等价内部方法——绕过普通用户每日发送上限、校验站内信启用、以管理员或系统发送者创建消息、返回 `site_message_id`。消息文案：标题 `恭喜中奖：{活动名}`；正文 `你在「{活动名}」中中奖。` / `兑换码：{code}` / `请复制该兑换码前往兑换页面使用。`；若活动配置了 `promo_text` / `promo_image_url`，追加文案，并把 https 图片 URL 单独成行供站内信详情渲染为图片。若消息创建失败，draw 事务回滚——绝不在没有可发放码消息的情况下记录中奖者。

**转盘 segments**：`BuildLotterySegments` 固定返回 8 格，不再随 `prize_count` 膨胀。Draw 结果的 `index` 指向任意奖品格或任意谢谢参与格。

### 后端 API
用户 API：
- `GET /api/v1/lottery/active` — 返回 `{ "campaign": {...} }`（含 `segments` 标签数组）或不该弹窗时 `{ "campaign": null }`。
- `POST /api/v1/lottery/:id/draw` — 返回 `won`/`index`/`label`/`message`，中奖含 `site_message_id`。

管理员 API：
- `GET /api/v1/admin/lottery/campaigns` — 列出活动，最新在前。
- `POST /api/v1/admin/lottery/campaigns` — 创建。校验：`name` 必填且 ≤120 runes；`subtitle` 默认 `登录就有机会，转一转赢取兑换码`；`prize_count >= 1`；`max_participants >= prize_count`；`len(codes) >= prize_count`；codes 去空白、非空、请求内唯一；站内信必须启用。`promo_text` 可选 ≤240 runes；`promo_image_url` 可选，必须是 `https://` 且不是 javascript/data 协议。
- `GET /api/v1/admin/lottery/campaigns/:id` — 详情、码、中奖者、抽奖计数。
- `POST /api/v1/admin/lottery/campaigns/:id/finish` — 标记结束。

### 前端
- **API 层**：新增 `frontend/src/api/lottery.ts` 与 `frontend/src/api/admin/lottery.ts`，映射用户和管理员端点。
- **Store**：把 `frontend/src/stores/lottery.ts` 重写为 API 驱动。State：`activeCampaign`、`loadingActive`、`drawing`、`lastResult`。Actions：`fetchActive()`、`draw(campaignId)`、`clearActive()` 及管理员创建 / 列表 / 详情 / 结束。不再向 `localStorage` 写活动或抽奖结果。
- **弹窗管理**：`LotteryPromptManager.vue` 监听认证和用户 ID，登录后拉活跃活动，检查 `sessionStorage` dismissal key `lottery_dismissed_v2`，仅在后端返回活动且会话 key 不含该活动 ID 时打开对话框，仅在抽奖前关闭时写 dismissal，抽奖完成后清空活跃活动。
- **对话框**：`LotteryDialog.vue` 在配置了 `promo_image_url` 时先展示可跳过的关注页，再进入转盘；中奖结果面板不渲染码，显示 `恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。`，并再次展示同一套海报；未中不展示海报。提供打开 `/site-messages` 的按钮。
- **管理员页**：`LotteryView.vue` 调管理员 API 创建活动、从后端加载历史、显示活跃 / 结束状态、参与人数和中奖者、详情面板显示未分配码和中奖记录；创建时可填引导文案与备份 S3 公开图链；站内信禁用时禁用创建并显示清晰错误。体验预览页：`frontend/public/lottery-ux-preview.html`。

### 错误处理
- 无活跃活动：`campaign: null`，非错误。
- 已抽奖：HTTP 409 `LOTTERY_ALREADY_DRAWN`，前端关闭弹窗。
- 活动满或已结束：HTTP 409 `LOTTERY_CAMPAIGN_CLOSED`，前端关闭弹窗并 toast。
- 创建时站内信禁用：HTTP 400 `LOTTERY_SITE_MESSAGES_DISABLED`。
- 后端日志含 campaign ID 和 user ID，但绝不记录兑换码值（除 debug-local 测试外）。

## 已决策
- 后端持久化取代前端存储，保持单一活跃活动。
- Draw 事务覆盖重复抽奖、参与上限、码分配和站内信发放。
- 中奖码只走站内信，结果弹窗不展示码。
- 转盘视觉固定 8 格，与奖品数量脱钩。
- 剩余奖品不少于剩余名额时必中（100 人 100 奖即为全中）。
- 关注引导 / 中奖广告共用一套素材；图片只接受备份 S3 / R2 的公开 HTTPS 直链（方式 A），不上传 data URI，本期不做管理端自动 PutObject。
- 抽奖前关注可跳过；中奖结果与中奖站内信展示同一套素材；未中不展示、不另发广告信。
- 不在本次范围：加权奖品、多活跃活动、公开免登录抽奖、抽奖奖品的邮件发放、自动兑换、跨设备持久化关闭 / dismiss 状态、微信关注真校验、独立图床上传 API。

## 实现
按 TDD 分六个任务（均先写失败测试 → 验证红灯 → 实现 → 验证绿灯）：

1. **后端服务契约 + TDD**：`domain/lottery.go`、`service/lottery.go`、`service/lottery_service.go` 及测试；修改 `site_message.go` / `site_message_service.go` 增加内部系统 / 管理员发送。用内存 stub 测试 `LotteryRepository`、`LotterySiteMessageSender`、`LotterySettingsReader`；先实现创建校验，再实现 draw 逻辑。
2. **持久化与生成的 Ent 代码**：三个 schema + `140_add_lottery.sql` + `repository/lottery_repo.go`；`go generate ./ent`；repository 含供 `Draw` 使用的事务方法。
3. **后端 handler / 路由 / wire**：`dto/lottery.go`、`lottery_handler.go`、`admin/lottery_handler.go`；给 `handler.Handlers` / `handler.AdminHandlers` 加 `Lottery` 字段；在 Wire provider set 注册；`go generate` 生成 `wire_gen.go`。
4. **前端 API store 与弹窗**：`api/lottery.ts`、`api/admin/lottery.ts`、改 store / `LotteryPromptManager.vue` / `LotteryDialog.vue`，会话 dismissal 留在 prompt manager。
5. **前端管理员页**：改 `LotteryView.vue` 接入管理员 API + i18n（zh/en/zh-Hant）。
6. **全量验证**：后端聚焦测试（`Lottery|SiteMessage`）、DI 相关包测试（`cmd/server`、`server/...`、`service`、`repository`）、前端聚焦测试、`typecheck` + `build`、`git status --short`。

## 相关
- 代码路径（后端）：`backend/internal/service/lottery*.go`、`backend/internal/repository/lottery_repo.go`、`backend/internal/handler/{,admin/}lottery_handler.go`、`backend/ent/schema/lottery_*.go`、`backend/migrations/140_add_lottery.sql`
- 代码路径（前端）：`frontend/src/stores/lottery.ts`、`frontend/src/api/{,admin/}lottery.ts`、`frontend/src/components/lottery/{LotteryPromptManager,LotteryDialog}.vue`、`frontend/src/views/admin/LotteryView.vue`
- 依赖站内信发放能力（`SiteMessageService.SystemSendToUser`），结果弹窗链接到 `/site-messages`。
