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
| [`features/subscription-credit-pool.md`](features/subscription-credit-pool.md) | 用户级订阅额度池:多订阅按时间扣费、额度优先消费、不足拆分扣余额、ledger 审计 |
| [`features/subscription-admin.md`](features/subscription-admin.md) | issue #47 订阅管理三项修复:撤销不计浪费、限额刷新时间可配、套餐入口迁移 |
| [`features/recharge-invoice-balance-gate.md`](features/recharge-invoice-balance-gate.md) | 充值页开票须知提示 + 首次使用余额门槛文案优化 |
| [`features/admin-invoice-export.md`](features/admin-invoice-export.md) | 管理员发票记录页 Excel 导出(全部 / 正在开票) |
| [`features/affiliate-signup-bonus.md`](features/affiliate-signup-bonus.md) | 邀请注册赠送纳入返利余额历史框架 + 独立管理员记录页 |
| [`features/affiliate-tier-rebate.md`](features/affiliate-tier-rebate.md) | 阶梯式邀请返利：管理员配置 L1-L4 门槛与返利比例，用户页展示后端计算进度 |
| [`features/real-lottery.md`](features/real-lottery.md) | 后端驱动多用户抽奖,中奖兑换码经站内信发放 |
| [`features/site-messages.md`](features/site-messages.md) | 站内信(类轻量邮件):收发读回复、未读红点、管理员开关与发信 |
| [`features/support-chat.md`](features/support-chat.md) | 站内 AI 客服浮窗:外部 gateway 接入、登录态修复、品牌配色、附件上传 |
| [`features/user-request-monitoring.md`](features/user-request-monitoring.md) | 管理员定向监控指定用户请求、限时抓取原始请求体(非阻塞网关) |
| [`features/external-auth-handoff.md`](features/external-auth-handoff.md) | 外部应用带用户来登录、登录后把 access token 回跳的接入协议与实现 |
| [`features/admin-settings-idempotency.md`](features/admin-settings-idempotency.md) | 管理员设置保存的服务端隐式幂等保护(防重复 / 重试重复写) |
| [`features/compliance-geo-block.md`](features/compliance-geo-block.md) | 中国大陆 IP 网页访问拦截(屏蔽网站、保持 API 开放)的实现与验收 |
| [`features/account-error-history.md`](features/account-error-history.md) | 账号错误历史:多来源 best-effort 异步记录、去重/节流/裁剪、账号「更多」菜单懒加载弹窗查看 |
| [`features/account-error-alert.md`](features/account-error-alert.md) | 账号异常 Telegram 告警:后台定时聚合 `ops_error_logs`,通知最近窗口内异常账号 |
| [`features/group-fallback.md`](features/group-fallback.md) | 分组级兜底:A 分组账号不可用时切到管理员配置的 B 分组重试 |
| [`features/model-market.md`](features/model-market.md) | 公共模型广场页面与后台配置:按平台 / 分组 / 计费类型展示可用模型,支持展示计费模式覆盖 |
| [`features/api-key-model-restriction.md`](features/api-key-model-restriction.md) | 用户 API 密钥模型允许列表:从模型广场按分组选择模型,网关拦截未授权模型请求 |
| [`features/_TEMPLATE.md`](features/_TEMPLATE.md) | 新功能文档模板——新增功能记录时照此建,放对 领域 / 模块 |

## operations/(部署与运维)

| 文档 | 一句话 |
|------|--------|
| [`operations/deployment.md`](operations/deployment.md) | Docker / 安装方式、datamanagementd、CDN 缓存规则——部署或排查线上时查 |

## records/(历史记录 · 归档供回看)

| 文档 | 一句话 |
|------|--------|
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
