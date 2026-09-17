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
| [`features/account-error-alert.md`](features/account-error-alert.md) | 账号异常 Telegram 告警：后台聚合 `ops_error_logs`，按账号 extra 开关/关键字/规则推送 |
| [`features/account-traffic-control.md`](features/account-traffic-control.md) | 账号可选流量控制：严格 RPM（滑动 60 秒 + 突发）与自适应并发（observe/automatic），extra 键默认关 |
| [`features/account-request-health.md`](features/account-request-health.md) | 账号列表最近 N 次请求健康条：单 IP 一根、IP 组按出口拆条，Redis 环形缓冲 |
| [`features/account-pool-auto-inspect.md`](features/account-pool-auto-inspect.md) | 号池自动巡检：成功率过低则加入指定分组并去掉指定模型，401 停调度可通知 Telegram |
| [`features/openai-oauth-ip-group.md`](features/openai-oauth-ip-group.md) | OpenAI OAuth IP 组选路：会话粘性、随机选择未尝试出口与两轮重试 |
