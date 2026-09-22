---
name: account-request-health
description: 账号列表最近 N 次请求健康条：单 IP 一根、IP 组按出口拆条，红块点开报错
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 账号最近请求健康度

账号列表上展示每个账号最近 N 次请求的成败条，用来判断出口是否健康。它和 Telegram 异常告警、自定义错误码无关。

## 背景 / 目标

- 管理员要在账号页直接看到最近请求是绿还是红，不必另开代理页。
- 单 IP 账号一根条；OpenAI OAuth IP 组按出口 IP 各一根，超过 2 根在单元格内滚动；单元格不重复显示 IP / 最近请求 / 限制状态表头。
- 红条必须能点开看报错。列表自动刷新不能被这次查询拖慢。

## 设计

- **写路径**：成功记在网关 `RecordUsage`，失败记在 `OpsService.RecordError(Batch)`。OpenAI IP 组在解析出口后把 `EgressProxyID` 写入请求上下文，失败/成功都能落到对应 IP。usage worker 显式复制出口 ID；错误队列保留每次尝试的出口快照，并在 JSON 限长后保持索引对应，不改写持久化的运维事件。单 IP 读取账号汇总，避免遗漏无代理快照的失败。
- **存储**：Redis 环形缓冲，账号 key `reqhealth:acct:{id}`，账号×代理 key `reqhealth:acct:{id}:proxy:{proxyID}`；`LPUSH` + `LTRIM 0 19`，TTL 7 天。列表按最旧在左、最新在右返回。
- **读路径**：`POST /admin/accounts/request-health/batch`，只查当前页账号。运行态通过 Kin 现有 `ConcurrencyCache` 计数（包含过期槽位清理与 live lease），IP 组复用 `conc:acct-proxy` 和 `openai:ip_group_cooldown`，单 IP 使用账号级并发，不扫 `usage_logs` / `ops_error_logs`。
- **展示**：绿成功、红失败、灰空槽；状态优先级 冷却中 > 过载 > 限流 > 执行中 > 空闲。窗口 8/12/16/20，默认 12，存在 localStorage。列默认显示。事件可带短分类（身份变化、票据 312、出口变化、模型改写、传输失败、资源限制），红块能看到分类，看不到票文或 Cookie。

## 已决策

- 不做独立代理页改写；健康度只挂在账号列表。
- `usage_logs` 没有 proxy/status、`ops_error_logs` 没有成功记录，所以必须旁路写 Redis，不能每次刷新扫库。
- extra 缺省不参与本功能；与 `extra.error_alert` 分离。

## 待解决

- 暂无。

## 相关

- 后端：`backend/internal/service/account_request_health.go`
- 管理接口：`backend/internal/handler/admin/account_request_health.go`
- 前端：`frontend/src/components/account/RequestHealthLines.vue`
- 账号列表：`frontend/src/views/admin/AccountsView.vue`
- IP 组：[[openai-oauth-ip-group]]
- Telegram 告警（不要混）：[[account-error-alert]]
