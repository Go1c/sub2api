---
name: knowledge
description: 项目知识库导航——查"某事怎么做"(standards)、"某功能怎么设计的"(features)、部署运维(operations)或历史决策(records)时,从这里找到对应 .md
metadata:
  type: index
---

# Knowledge(项目知识库 · 导航)

本文件是 `knowledge/` 下所有 .md 的导航 meta:一行描述 + 路径,按需下钻。新增 / 修改文档后必须回这里同步一行。

## standards/(开发规范 · 要遵守的「怎么做」)

| 文档 | 一句话 |
|------|--------|
| [`standards/workflow.md`](standards/workflow.md) | 分支角色 / 提交 / PR / 上游同步 / 发布——动手改代码、开 PR、发版前查 |
| [`standards/testing.md`](standards/testing.md) | 本地必跑命令、CI 流水线、集成测试 build tag、DoD——实现功能 / 修 bug / 提 PR 前查 |
| [`standards/code-style.md`](standards/code-style.md) | 代码风格与 LumioAPI 视觉识别(品牌色 / 字体 / 类)、前端约定——写代码 / 改样式时查 |
| [`standards/dev-environment.md`](standards/dev-environment.md) | 本地环境、技术栈、常见坑点与解决方案——搭环境、排查本地构建 / 数据库问题时查 |

## features/(功能设计与记录 · 供了解)

| 文档 | 一句话 |
|------|--------|
| [`features/payment.md`](features/payment.md) | 内置支付系统:服务商接入、Webhook、订单状态、可信余额履约限流边界、管理端集成 Admin API |
| [`features/subscription-pricing.md`](features/subscription-pricing.md) | 订阅套餐计价口径与计算器工具——定价 / 折扣 / 日周上限怎么算 |
| [`features/subscription-credit-pool.md`](features/subscription-credit-pool.md) | 用户级订阅额度池:多订阅按时间扣费、额度优先消费、不足拆分扣余额、ledger 审计；用户自助重置周限（每订阅周期一次） |
| [`features/subscription-admin.md`](features/subscription-admin.md) | issue #47 订阅管理三项修复:撤销不计浪费、限额刷新时间可配、套餐入口迁移 |
| [`features/recharge-invoice-balance-gate.md`](features/recharge-invoice-balance-gate.md) | 充值页开票须知提示 + 首次使用余额门槛文案优化 |
| [`features/admin-invoice-export.md`](features/admin-invoice-export.md) | 管理员发票记录页 Excel 导出(全部 / 正在开票) |
| [`features/affiliate-signup-bonus.md`](features/affiliate-signup-bonus.md) | 邀请注册赠送纳入返利余额历史框架 + 独立管理员记录页 |
| [`features/affiliate-tier-rebate.md`](features/affiliate-tier-rebate.md) | 阶梯式邀请返利：管理员配置 L1-L4；用户 GET /user/aff 返回阶梯进度与运行规则 |
| [`features/real-lottery.md`](features/real-lottery.md) | 后端驱动多用户抽奖,中奖兑换码经站内信发放 |
| [`features/site-messages.md`](features/site-messages.md) | 站内信(类轻量邮件):收发读回复、未读红点、管理员开关与发信；补偿批次 amount 为成功发出合计 |
| [`features/balance-low-websocket-notify.md`](features/balance-low-websocket-notify.md) | （已废弃）用户侧浏览器 WebSocket 通知，已由 Webhook 替代 |
| [`features/webhook-balance-robot-notify.md`](features/webhook-balance-robot-notify.md) | 个人资料 Webhook 通知：余额/站内信/公告 HTTPS POST（无 WebSocket） |
| [`features/balance-low-site-message-notify.md`](features/balance-low-site-message-notify.md) | （作废方向）曾误写为站内信通道；以 websocket 文档为准 |
| [`features/support-chat.md`](features/support-chat.md) | 站内 AI 客服浮窗:外部 gateway 接入、登录态修复、品牌配色、附件上传 |
| [`features/user-request-monitoring.md`](features/user-request-monitoring.md) | 管理员定向监控指定用户请求、限时抓取原始请求体(非阻塞网关) |
| [`features/external-auth-handoff.md`](features/external-auth-handoff.md) | 外部应用带用户来登录、登录后把 access token 回跳的接入协议与实现 |
| [`features/auth-cross-domain-bridge.md`](features/auth-cross-domain-bridge.md) | 主站 access JWT 经 `/auth/bridge` 换成控制台 localStorage 会话（与 external-auth-handoff 方向相反） |
| [`features/admin-settings-idempotency.md`](features/admin-settings-idempotency.md) | 管理员设置保存的隐式幂等、Banner 单字段更新、PostgreSQL fallback，以及 fork 字段必须经 parseSettings 读回 |
| [`features/compliance-geo-block.md`](features/compliance-geo-block.md) | 中国大陆 IP 网页访问拦截(屏蔽网站、保持 API 开放)的实现与验收 |
| [`features/account-error-history.md`](features/account-error-history.md) | 账号错误历史:多来源 best-effort 异步记录、去重/节流/裁剪、账号「更多」菜单懒加载弹窗查看 |
| [`features/account-error-alert.md`](features/account-error-alert.md) | 账号异常 Telegram 告警:后台定时聚合 `ops_error_logs`,通知最近窗口内异常账号 |
| [`features/group-fallback.md`](features/group-fallback.md) | 分组级兜底:A 分组账号不可用时切到管理员配置的 B 分组重试 |
| [`features/model-market.md`](features/model-market.md) | 公共模型广场页面与后台配置:按平台 / 分组 / 计费类型展示可用模型,支持展示计费模式覆盖 |
| [`features/api-key-model-restriction.md`](features/api-key-model-restriction.md) | 用户 API 密钥模型允许列表:从模型广场按分组选择模型,网关拦截未授权模型请求 |
| [`features/user-subscription-purchase-ban.md`](features/user-subscription-purchase-ban.md) | 管理员按用户禁止购买订阅：字段、编辑开关、CreateOrder 硬拦截、「无权限购买」 |
| [`features/user-access-token.md`](features/user-access-token.md) | 用户长效 opaque Access Token：密钥管理 + 只读用量/余额/订阅；活跃数上限、用户 RPM、usage 查询护栏 |
| [`features/balance-debit-wallet.md`](features/balance-debit-wallet.md) | 多外部站共用站内余额：双重身份原子扣款、永久幂等账本、本人流水、管理端 wallet_debit 合并流水、用户「我的订单」可见本人扣款 |
| [`features/lumio-desktop-client.md`](features/lumio-desktop-client.md) | Lumio Codex 桌面客户端服务端契约：公开配置、账号级唯一 Key、一次性支付交接，以及 GET /v1/models 不查余额；接入桌面启动或充值流程时查 |
| [`features/codex-download-nav.md`](features/codex-download-nav.md) | 首页与登录后顶栏的 Codex 下载外链，跳转到 bestcodex.app；改导航入口或下载引导时查 |
| [`features/daily-checkin.md`](features/daily-checkin.md) | 独立每日签到：原子发奖、连续周期、全站每日预算，以及用户和管理员流水（含日/周/月/累计发放统计） |
| [`features/openai-hidden-luna-autoreview.md`](features/openai-hidden-luna-autoreview.md) | 对用户隐藏 GPT-5.6 Luna；默认把 Auto-review / luna 改写到 Terra，仅显式 Luna 键或 Auto-review→Luna 值可打真 Luna |
| [`features/frontend-seo-meta.md`](features/frontend-seo-meta.md) | 前端 GEO/SEO：Go embed 注入 description/OG/Twitter/JSON-LD，以及 `#app` 内爬虫可见简介 |
| [`features/umami-public-tracking.md`](features/umami-public-tracking.md) | 前台 Umami：SPA nonce inline 条件插入官方 defer 脚本、直开 /admin 不加载、docs 静态页直接埋、CSP script-src 放行 data.lumio.games |
| [`features/_TEMPLATE.md`](features/_TEMPLATE.md) | 新功能文档模板——新增功能记录时照此建,放对 领域 / 模块 |

## operations/(部署与运维)

| 文档 | 一句话 |
|------|--------|
| [`operations/deployment.md`](operations/deployment.md) | Docker / 安装方式、datamanagementd、迁移护栏（ent 列须有迁移、DROP 延后）、CDN 缓存规则——部署或排查线上时查 |

## records/(历史记录 · 归档供回看)

| 文档 | 一句话 |
|------|--------|
| [`records/upstream-sync-v0177-codex-compact-group-usage.md`](records/upstream-sync-v0177-codex-compact-group-usage.md) | 2026-08-15：main 快进 0.1.177 后，Codex turn-state、native compaction v2、分组用量日 rollup 的选择性同步（待 Review 合入 dev） |
| [`records/upstream-sync-grok-v0176-jwt-xsearch-pricing.md`](records/upstream-sync-grok-v0176-jwt-xsearch-pricing.md) | 2026-08-13：Grok JWT 档位、独立 /x_search、分组 model_pricing、SuperGrokPro、用量快照与 4.5/4.6 官方价卡的选择性同步（待 Review 合入 dev） |
| [`records/prod-frozen-balance-migration-gap-20260728.md`](records/prod-frozen-balance-migration-gap-20260728.md) | 2026-07-28：publish 引入 `users.frozen_balance` Ent 字段却无迁移，登录 503；手改恢复后补 920 + CI 护栏 |
| [`records/upstream-sync-20260720-v0162.md`](records/upstream-sync-20260720-v0162.md) | main→0.1.162：#234–#249 已合（含 Grok/WS turn/调度冷却）；B/C/E/F 大块默认跳过 |
| [`records/upstream-sync-20260715.md`](records/upstream-sync-20260715.md) | v0.1.156 ordered 同步台账：全量 unit 已绿、VERSION 0.1.156；integration/前端/commit 切分仍待（进行中） |
| [`records/upstream-sync-20260710.md`](records/upstream-sync-20260710.md) | v0.1.150 日更同步台账：18 个上游落点的 fork 适配、例外与验证结果（2026-07-10） |
| [`records/upstream-sync-review-20260616.md`](records/upstream-sync-review-20260616.md) | v0.1.136 同步上线前独立代码复审与 go/no-go 结论(2026-06-16) |
| [`records/upstream-sync-acceptance-20260615.md`](records/upstream-sync-acceptance-20260615.md) | v0.1.136 同步验收清单:合了什么、风险在哪、怎么验(2026-06-15) |
| [`records/upstream-sync-20260615.md`](records/upstream-sync-20260615.md) | v0.1.136 同步评估台账与分批处理进度(2026-06-15) |
| [`records/upstream-sync-grok-20260711.md`](records/upstream-sync-grok-20260711.md) | Grok/xAI 上游功能链的移植范围、排除项、fork 兼容决策、验证与 dev→publish 交付记录(2026-07-11) |
| [`records/grok-apikey-upstream-20260711.md`](records/grok-apikey-upstream-20260711.md) | Grok `type=apikey`（URL + API Key）上游透传：实现边界、与 youxuanxue 差异、getezo/`grok-4.5` 实测(2026-07-11) |
| [`records/main-vs-dev-assessment-20260529.md`](records/main-vs-dev-assessment-20260529.md) | main→dev 分叉深度评测、按主题吸收策略与首轮 selective sync 记录(2026-05-29) |
| [`records/memory-risk-analysis-20260509.md`](records/memory-risk-analysis-20260509.md) | 内存暴涨 / 泄漏、DoS 资源耗尽与 DB 增长风险分析(2026-05-09) |
| [`records/bugwall-zeabur-migration-checksum-20260423.md`](records/bugwall-zeabur-migration-checksum-20260423.md) | Zeabur 部署 migration 110/112 checksum mismatch 故障墙(2026-04-23) |

## assets/(随知识保留的二进制)

- `assets/ai-exit-mainland-china-compliance-checklist.pdf` —— 退出中国大陆合规清单,见 [`features/compliance-geo-block.md`](features/compliance-geo-block.md)。

---

新增 / 修改 / 维护知识文档(放哪、frontmatter、同步本导航)→ 用 `spec-steward` 技能。
