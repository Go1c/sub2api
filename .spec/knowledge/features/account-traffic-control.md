---
name: account-traffic-control
description: 账号可选流量控制：严格 RPM（滑动 60 秒 + 令牌桶突发）与自适应并发（observe/automatic），extra 键 account_traffic_control
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 账号可选流量控制

账号级、默认关闭的两种独立开关：**严格 RPM**（本地滑动 60 秒硬上限 + 瞬时突发额度）与**自适应并发**（按上游 429/5xx 表现给出建议并发，observe 只展示、automatic 自动降速缓慢恢复）。配置存放在 `accounts.extra.account_traffic_control`，缺键 = 两开关都关，存量账号不受影响。

## 背景 / 目标

- 部分上游对账号维度的请求频率敏感，管理员希望在本网关先挡住超额请求，而不是等上游 429 后再被动降速。
- 只做「上限内的整形」，不改身份 / TLS 指纹 / 代理 / IP 组；固定并发上限（`account.Concurrency`）仍是天花板，自适应只在 [最低并发, 固定上限] 区间内调整。
- 60/5/1 与 3 次/60 秒阈值是可改的起点参数，不是上游真实额度。

## 设计

- **策略**：`AccountTrafficPolicy`（`strict_rpm_enabled, rpm, burst, adaptive_enabled, adaptive_mode=observe|automatic, min_concurrency, failure_threshold, failure_window_seconds, recovery_seconds`）。`Enabled()` = 任一开关开；`Enforces()` = 严格 RPM 开或 adaptive+automatic——只有 Enforces 才会拒绝请求，observe 模式断连/缓存故障都不影响流量。
- **准入（Redis Lua，键 `kin:acct_tc:{id}:state` HASH + `:minute` ZSET，hash-tag 同槽）**：滑动 60 秒 RPM + 令牌桶突发一次原子判定；`rev<oldrev && oldsig!=sig` 时，旧会话在新策略不再 Enforces 的情况下 skip-observation 放行（reason 4），硬限制开启后的旧策略必须拒绝（reason 3）。
- **与 Kin 并发缓存的关系（映射 (a)）**：外层并发槽位仍由 `ConcurrencyCache.AcquireAccountSlot` 强制；流量 Lua **不做** in-flight 拒绝、无 leases ZSET。`Refresh` 只对 state HASH 续期（permit 每 30 秒心跳）；`Snapshot.in_flight` 读 `ConcurrencyCache.GetAccountConcurrency`。automatic 的降速通过 `EffectiveAccountTrafficConcurrency`（`min(hardLimit, recommended)`）下调外层 slot 上限实现，读失败退回 hardLimit。
- **建议值演化（Finish Lua）**：429/5xx 在失败窗口内达到阈值 → recommended 减半（不低于 min_concurrency，10 秒调整冷却）；距上次失败稳定 recovery 周期且连续 3 次健康成功 → +1（不超过 hard）。SSE / WS 终端帧事件（`response.failed` 带 rate_limit/server_error 等）参与同一分类，非 EOF 中断的 2xx 不算恢复证据。
- **计费覆盖**：请求经 `WithAccountTrafficRequest(req, account)` 挂计划，`NewControlledHTTPUpstream` 的 `Do/DoWithTLS` 统一走 `traffic.DoHTTP`（覆盖响应体全生命周期，covered marker 防插件/重试双计费）；GET/HEAD 探测不占预算；quota/balance/models 等探测路径不包。
- **WS / 实时语音**：`forwardOpenAIWSV2` 与 ingress `sendAndRelay` 每轮 begin/finish；passthrough 帧连接 `wrapAccountTrafficFrameConn` 只对 `response.create` 计一轮，终端事件释放。Grok 实时语音 Enforces 时拨号前 400 拒绝（不支持逐轮硬限制），observe-only 才 Begin。
- **失败语义**：`AccountTrafficLimitError.As` 转为 `UpstreamFailoverError`：RequestScopedTransient、`Reason=account_traffic_limit`、`NextAccountStop`、Retry-After 头。`ShouldReportAccountScheduleFailure` 对该 reason 返回 false——限流是本账号固定策略，不进调度健康；failover 耗尽路径透传客户端状态码与文案。
- **管理面**：`GET/PUT /admin/accounts/:id/traffic-control`（AccountHandler 方法，`SetAccountTrafficHandler` setter 注入）。PUT 走 `UpdateAccountExtra` key 级 merge，随后 `traffic.Sync` 推 revision/signature；遥测失败仍返回可编辑 policy（`state_available: false`）。Create/Update/BatchCreate/BulkUpdate/ApplyOAuth 均校验该 extra 键。
- **前端**：`AccountTrafficControls.vue` 嵌入 EditAccountModal，随账号一起保存（无独立保存按钮）；草稿按账号 ID 缓存，同账号重开不冲掉未保存编辑；「填入建议参数」只填数字不改开关。`accountTraffic.ts` 提供类型、默认值、校验与 API。

## 已决策

- 不移植新站的 leases ZSET、Mode1EffectiveConcurrency、智能测试 intelligentContext、保护机与 integrity——Kin 用 ConcurrencyCache/account.Concurrency/现有测试链路替代。
- `NewAccountTrafficCache(rdb, concurrency)` 双参构造；snapshot 的 in-flight 单一事实来源是 ConcurrencyCache。
- 新命名空间 `kin:acct_tc`，不复用现有 `rpm:{accountID}:{minute}`，也不与新站 `account_traffic:{id}:*` 冲突。
- 插件（OpenAI OAuth 能力绑定）出站也计入预算：plugin transport 外层套 `traffic.DoHTTP`。

## 关键文件

- `backend/internal/service/account_traffic_policy.go` / `account_traffic_service.go` / `account_traffic_events.go` / `account_traffic_ws.go`
- `backend/internal/repository/account_traffic_cache.go`（Lua + ConcurrencyCache 快照）
- `backend/internal/repository/http_upstream.go`（`NewControlledHTTPUpstream`，traffic nil 回落 uncontrolled）
- `backend/internal/handler/admin/account_traffic_handler.go`、`backend/internal/server/routes/admin.go`
- `frontend/src/api/admin/accountTraffic.ts`、`frontend/src/components/account/AccountTrafficControls.vue`
