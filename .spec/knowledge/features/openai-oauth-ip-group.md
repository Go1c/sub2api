---
name: openai-oauth-ip-group
description: OpenAI OAuth 账号 IP 组：默认单代理；选组后同一套指纹换出口，对话钉 IP，每账号每 IP 并发；IP 级 429/过载扫两遍组内 IP 后换号，不把账号打 429
metadata:
  type: doc
  level: L2
  status: 已交付
---

# OpenAI OAuth IP 组

简介：管理端在代理页建 **IP 组**，OpenAI OAuth 账号配 IP 时二选一（单独一条 IP / 一个 IP 组）。选组后出站只换代理，**不改** Codex 绑机指纹。同一条对话钉死一个 IP；只有新对话或出口死亡才换组内另一条。

## 背景 / 目标

- 一个 ChatGPT OAuth 号希望同时走多条出口（多国家 IP），每条 IP 对这个号有本地并发上限；满了只让**新对话**换下一条。
- 官方社区 PR [#6650](https://github.com/Wei-Shaw/sub2api/pull/6650) 做的是「出站模板 + 设备槽 + 换身份」，不能整包合进本 fork。
- 本仓库已有编译期 Codex CLI 0.154.0 Ubuntu 绑机指纹，必须保持。见 [`openai-codex-fingerprint.md`](openai-codex-fingerprint.md)。

## 设计

### 配置面

- IP 管理里先建 **IP 组**（中文不要叫「分组」，以免和渠道分组混淆）。英文标识用 `proxy_ip_group`。
- 组挂已有 `proxies` 行；组上一个统一的 `per_ip_concurrency`（例如 10）。
- 仅 **OpenAI + OAuth** 账号在配 IP 时二选一：
  1. 单独一条 IP → 现有 `accounts.proxy_id`，行为与现在完全相同。
  2. 一个 IP 组 → `accounts.proxy_ip_group_id`；出站不再用 `proxy_id`。
- 选组本身就是开关。不选组 = 默认现在这套。
- Claude / Gemini / Grok / OpenAI API Key 账号 UI 与出站都不出现 IP 组。

### 运行面

- 调度仍先选账号，再解析该号的出口 URL。
- 指纹、UA、installation、session 全部走现有 `enforceCodexIdentityHeaders` / 编译期常量，**零改动**。
- 「一条对话」= 现有 sticky 信号：`session_id` / `conversation_id` / `prompt_cache_key`，以及同一条 WS 连接。
- 同一条对话钉在第一次选中的 IP。新对话才在组里挑「该账号下还未打满」的 IP。
- 并发两把锁都要过：账号 `concurrency` 总闸 + `(account_id, proxy_id)` 每 IP 锁。多账号共用同一条法国 IP 时各算各的上限。
- 绑定 IP **死亡**（禁用 / 非 active / 过期 / 已从组里删除）时，这条对话才允许换组内另一条可用 IP 并改绑。
- 不算死亡、因此不换：IP 并发满、账号总并发满、单次延迟探测失败。这些走现有排队 / 失败。
- **IP 级瞬时故障**（上游 429 且不是额度耗尽、overloaded、可重试 processing error）：当前请求内按 `proxy_id` 升序扫一遍组内活 IP，全失败再扫第二遍；两遍仍失败才换号。没有下一个号则回客户端错误。这条路径**不得** `SetRateLimited` / 运行时 429 熔断该账号。真额度耗尽（`usage_limit_reached` 或 5h/7d `used_percent >= 100`）仍走账号级 429。见 [`openai-capacity-shed-retry.md`](openai-capacity-shed-retry.md)。

### 实现面（约定，实现时按计划落地）

- 新表 + `accounts.proxy_ip_group_id`，迁移走 fork `9xx`（当前最高 `944`，下一条 `945`）。禁止使用官方 / #6650 的 `234`–`240` 文件名。
- 对话→IP 绑定优先 Redis，TTL 对齐现有 sticky session；不要引入 #6650 的设备槽 / profile / template 表。
- 现有 `proxy_fallback_origin_id` 是代理**过期改写账号 proxy_id**，与 IP 组无关，不要复用。
- Spark shadow 的代理从父号继承：父号改组时要同步 `proxy_ip_group_id`（对照现有 `propagateProxyToShadows`）。
- OAuth 授权/换票需要一条出口时，从组里挑一条 **active 且未过期** 的成员即可，不单独做对话绑定。

## 已决策

- 不整包合并 #6650；只允许参考其「对话绑槽 + 每槽并发」思路。
- 默认单 IP；选组才多 IP。指纹不变。
- 对话钉 IP；新对话才换；出口死亡才允许进行中的对话换绑。
- IP 级 429/过载：同请求扫组内 IP 两遍，再换号；无号则报错；不把账号打 429。
- 账号并发 = 总闸；每 IP 上限统一配在组上；计数维度是账号 × IP。
- v1 只对 OpenAI OAuth 生效。
- 一个号不能同时既有生效的单 IP 又有组；选组后 `proxy_id` 必须清空（或保存时清掉），出站只看组。

## 待解决

- 无。实现细节见 [`.spec/plans/2026-09-11-openai-oauth-ip-group.md`](../../plans/2026-09-11-openai-oauth-ip-group.md)。

## 相关

- [`openai-codex-fingerprint.md`](openai-codex-fingerprint.md)
- [`openai-capacity-shed-retry.md`](openai-capacity-shed-retry.md)
- 上游社区 PR（**禁止整包合入**）：https://github.com/Wei-Shaw/sub2api/pull/6650
- 实现计划：`.spec/plans/2026-09-11-openai-oauth-ip-group.md`
- 开工提示词：`.spec/plans/2026-09-11-openai-oauth-ip-group-AGENT-PROMPT.md`
- 迁移：`backend/migrations/945_proxy_ip_groups.sql`
- 运行时选 IP：`backend/internal/service/openai_ip_group_proxy.go`
- Admin API：`/api/v1/admin/proxy-ip-groups`（`backend/internal/server/routes/admin.go`）
- 前端：`frontend/src/components/admin/proxy/ProxyIPGroupsDialog.vue`、`frontend/src/components/account/OpenAIAccountProxyFields.vue`
