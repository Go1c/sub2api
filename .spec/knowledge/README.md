---
name: knowledge
description: 项目知识库导航——查"某事怎么做"(standards)、"某功能怎么设计的"(features)时从这里找到对应 .md
metadata:
  type: index
---

# Knowledge(项目知识库 · 导航)

本文件是 `knowledge/` 下文档的导航。新增 / 修改文档后必须回这里同步一行。

## features/(功能设计与记录 · 供了解)

| 文档 | 一句话 |
|------|--------|
| [`features/proxy-probe-quality.md`](features/proxy-probe-quality.md) | 渠道子页集中配置探测出口与 Turn-State 策略；账号开关缺省缺票原样转发，显式 enforce 才换号 |
| [`features/account-error-alert.md`](features/account-error-alert.md) | 账号异常 Telegram 告警：后台聚合 `ops_error_logs`，按账号 extra 开关/关键字/规则推送 |
| [`features/account-traffic-control.md`](features/account-traffic-control.md) | 账号可选流量控制：严格 RPM（滑动 60 秒 + 突发）与自适应并发（observe/automatic），extra 键默认关 |
| [`features/account-request-health.md`](features/account-request-health.md) | 账号列表最近 N 次请求健康条：单 IP 一根、IP 组按出口拆条，Redis 环形缓冲 |
| [`features/account-pool-auto-inspect.md`](features/account-pool-auto-inspect.md) | 号池自动巡检：成功率过低则加入指定分组、去掉指定模型并关闭 429 豁免，401 停调度可通知 Telegram |
| [`features/openai-oauth-429-exemption.md`](features/openai-oauth-429-exemption.md) | OpenAI OAuth 429 拉闸豁免：extra 键缺省豁免 Retry-After 冷却，手动或巡检降级时关闭，真实耗尽不受影响 |
| [`features/openai-oauth-ip-group.md`](features/openai-oauth-ip-group.md) | OpenAI OAuth IP 组选路：会话粘性、随机选择未尝试出口与两轮重试 |
| [`features/openai-import-batch-scheduling.md`](features/openai-import-batch-scheduling.md) | OpenAI 账号按首次入库固定时间批次优先调度，粘性会话优先且持续报错账号降级 |
| [`features/codex-2fa-import-recovery.md`](features/codex-2fa-import-recovery.md) | Codex 2FA JSON 导入、Team 优先授权，以及 sub2api 上游 401 停调度后的后台重新登录恢复 |
| [`features/sticky-ip-turn-state.md`](features/sticky-ip-turn-state.md) | 粘性 IP 组将账号采票出口与 Turn-State 成对绑定，保持到期后采新票并一起换新 |
