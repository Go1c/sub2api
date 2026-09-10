---
name: channel-iq-detect
description: 渠道管理「智商检测」看板：按分组纳入 OpenAI 账号、并发跑 SVG 成图、支持定时自动检测
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 渠道智商检测

管理端「渠道管理 → 智商检测」看板。把指定分组里的 OpenAI/Codex 账号放进检测列表，一次并发跑智商检测（SVG 成图），并可按间隔自动再跑。

## 背景 / 目标

- 账号页智商检测一次只能测一个号，看不出一组渠道号的作图能力。
- 目标：一张列表同时看到每个号最近一次成图、模型、思考强度、耗时和 token；右上角选分组纳入列表，一键全部检测，可选 30 分钟自动检测。

## 设计

- **交互面**：侧栏「渠道管理」下新增「智商检测」（`/admin/channels/iq`）。右上角「设置」选分组 / 自动检测间隔，「全部检测」并发跑列表里所有号。每行：空闲/检测中/成功/失败、测试次数、SVG 缩略图、模型、`low`、耗时与 token。点缩略图放大。只纳入所选分组里 `platform=openai` 的活跃账号（分组变动后列表跟着变）。行内「移除」把账号排除出监测（不改分组），成图一并删除；设置里可恢复。
- **实现面**：独立表 `channel_iq_settings`（单行：分组、排除账号、自动开关、间隔、模型、提示词）和 `channel_iq_results`（每账号最近一次结果）。**SVG 只存在 `channel_iq_results.svg` 文本列，不写磁盘目录。** 每次检测覆盖该账号上一张图；移除账号时删行。跑检测复用 `TestAccountConnection` + `mode=iq`（含过载重试）。后台 `POST /admin/channel-iq/run` 立刻返回，并发上限 8。定时任务只在持有 leader lock 的实例上跑。不走管理员全局 settings 的 parse 链。
- **设计面**：默认模型 `gpt-6-astra`、提示词 pelican、思考 `low`、间隔 1800 秒。401 不重试。普通测试连接不受影响。

## 已决策

- 列表跟分组实时成员走，不另做账号快照表——「分组抓到的号」要跟着调度分组变。
- 结果落独立表而不是 `accounts.extra`，避免 SVG 把账号缓存打爆。
- 全部检测并发但有上限 8，避免一次打爆上游。
- 部分号不用监测：排除名单（`excluded_account_ids`），不改调度分组。
- SVG 不落盘，只保留每账号最近一次；再测覆盖，移除则删除。

## 待解决

- 无。

## 相关

- [account-iq-svg-test.md](account-iq-svg-test.md)
- 入口：`backend/internal/service/channel_iq.go`、`frontend/src/views/admin/ChannelIqView.vue`
