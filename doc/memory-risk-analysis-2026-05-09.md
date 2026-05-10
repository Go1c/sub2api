# 内存/日志/数据库风险分析结论

日期：2026-05-09

## 1. 结论摘要

- 当前现象是“内存突然暴涨，然后回落”，更像瞬时大对象分配、异步队列堆积、数据库/Go 运行时缓存与 GC 回收，不像典型“持续泄漏不回落”。
- 代码里最像长期内存风险的点，是两个只做懒清理的 `sync.Map`：
  - `openaiCompatSessionResponses`
  - `openaiCompatAnthropicDigestSessions`
- 应用文件日志会轮转，不会无限写单个文件。
- 运维监控相关日志/告警事件会定期清理。
- 风控中心日志会定期清理。
- PostgreSQL 进程内存不应无限增长，但数据库磁盘体积是否持续增长取决于各表是否有 retention/清理任务。

## 2. 更像“瞬时暴涨”的内存来源

### 2.1 大请求体完整读入内存

代码把请求体上限默认设得很高：

- `server.max_request_body_size = 256MB`
- `gateway.max_body_size = 256MB`

相关位置：

- `backend/internal/config/config.go`
- `backend/internal/server/http.go`
- `backend/internal/server/routes/gateway.go`
- `backend/internal/pkg/httputil/body.go`

风险点：

- 多个 handler 会直接调用 `ReadRequestBodyWithPrealloc` 把整个 body 读入内存。
- 如果请求带 `gzip/zstd/deflate`，代码会先把压缩后的原始 body 全读入内存，再解压到新的 buffer。
- 虽然解压后限制为 64MB，但压缩包原文仍可达到全局 body 上限，因此单请求会同时占用：
  - 原始压缩体内存
  - 解压后的新 buffer
  - 后续 JSON 解析/日志处理额外对象

这类模式在高并发下非常容易形成“突然拉高、随后被 GC 回收”的内存曲线。

### 2.2 错误日志链路会放大失败请求的内存占用

相关位置：

- `backend/internal/handler/ops_error_logger.go`
- `backend/internal/service/ops_service.go`

风险点：

- 失败请求会额外采集请求体与错误响应体。
- 中间件会在请求上下文里暂存原始请求体，直到请求收尾。
- 错误响应捕获上限为 `64KB`。
- 入队前请求体会脱敏并裁剪到 `256KB`，这有帮助，但如果瞬时失败请求很多，仍然会形成明显内存峰值。
- 运维错误日志队列虽然是有界的，但队列满之前仍可能积压较多对象。

### 2.3 usage 记录异步池与风控异步队列都偏大

相关位置：

- `backend/internal/service/usage_record_worker_pool.go`
- `backend/internal/service/content_moderation.go`

默认值：

- usage 记录池：`worker_count=128`，`queue_size=16384`，自动扩容最大 `512`
- 风控队列默认 `32768`，底层 channel 容量按最大值建立到 `100000`

这类设计的好处是抗抖动，但代价是高峰期更容易把大量任务对象、字符串、请求摘要同时挂在内存里。

## 3. 更像“长期泄漏”的候选点

### 3.1 OpenAI 兼容会话映射只做懒清理

相关位置：

- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_messages_continuation.go`
- `backend/internal/service/openai_messages_digest_session.go`

风险描述：

- 这两个 `sync.Map` 的 value 都带 `ExpiresAt`。
- 但过期删除只发生在“再次访问同一个 key”时。
- 如果 key 基数很高，而且很多 key 写入后再也不会被访问，过期项就可能一直留在内存里。

这更像慢性增长风险，而不是你截图里那种尖峰后快速回落。

## 4. 日志是否会定期清理

### 4.1 应用文件日志

会轮转。

默认配置：

- `log.rotation.max_size_mb = 100`
- `log.rotation.max_backups = 10`
- `log.rotation.max_age_days = 7`

相关位置：

- `backend/internal/pkg/logger/logger.go`
- `backend/internal/config/config.go`
- `deploy/.env.example`
- `deploy/config.example.yaml`

### 4.2 运维监控/告警日志

会定期清理。

相关服务：

- `OpsCleanupService`

默认配置：

- `ops.cleanup.enabled = true`
- `ops.cleanup.schedule = 0 2 * * *`
- 错误日志/分钟指标/小时指标默认保留 `30` 天

会清理的表：

- `ops_error_logs`
- `ops_retry_attempts`
- `ops_alert_events`
- `ops_system_logs`
- `ops_system_log_cleanup_audits`
- `ops_system_metrics`
- `ops_metrics_hourly`
- `ops_metrics_daily`

相关位置：

- `backend/internal/service/ops_cleanup_service.go`
- `backend/internal/config/config.go`
- `backend/internal/service/ops_settings.go`
- `backend/internal/service/wire.go`

补充说明：

- 当前实现已经确认支持从 `ops_advanced_settings.data_retention` 动态覆盖 retention。
- UI 更新设置后会触发 `Reload`，不是只吃静态配置。
- 被自动清理的是告警事件 `ops_alert_events`，不是告警规则配置本身。

### 4.3 风控中心日志

会定期清理。

相关服务：

- `ContentModerationService.cleanupWorker`

执行方式：

- 启动后延迟 `5` 分钟第一次执行
- 之后每 `24` 小时执行一次

清理对象：

- `content_moderation_logs`

默认保留策略：

- 命中日志：`180` 天
- 未命中日志：`3` 天

相关位置：

- `backend/internal/service/content_moderation.go`
- `backend/internal/repository/content_moderation_repo.go`

补充说明：

- 风控配置里把 `retention_days` 设成 `0`，不会关闭清理。
- 当前代码会把 `<= 0` 归一化回默认值。

## 5. usage_logs / 其他审计表是否会持续增长

### 5.1 usage_logs

`usage_cleanup` 服务本身不是“全自动按天删表”，它只是执行任务表里的待办清理任务。

但项目里另有自动 retention 链路：

- `DashboardAggregationService` 默认开启
- 聚合后会定期执行 retention cleanup
- 会自动清理：
  - `usage_logs`
  - usage dashboard 聚合表
  - `usage_billing_dedup`

默认 retention：

- `usage_logs = 90` 天
- `usage_billing_dedup = 365` 天
- hourly aggregates = `180` 天
- daily aggregates = `730` 天

相关位置：

- `backend/internal/service/dashboard_aggregation_service.go`
- `backend/internal/repository/dashboard_aggregation_repo.go`
- `backend/internal/config/config.go`
- `backend/internal/service/wire.go`

结论：

- 如果 `dashboard_aggregation.enabled=true`，原始 `usage_logs` 不会无限增长。
- 如果把 dashboard aggregation 关掉，`usage_logs` 才可能持续长大。

### 5.2 payment_audit_logs

目前没有看到定时 retention/cleanup 服务。

相关位置：

- `backend/migrations/093_payment_audit_logs.sql`
- `backend/internal/service/payment_stats.go`

结论：

- `payment_audit_logs` 更像长期保留审计表，磁盘体积可能持续增长。
- 这主要影响数据库存储体积，不像“瞬时内存尖峰”。

## 6. PostgreSQL 内存会不会越来越大

结论：

- PostgreSQL 进程内存不应因为当前代码逻辑而无限增长。
- 更常见的是缓存、连接内存、查询工作内存、autovacuum 导致的波动。
- “内存突然上去再下来”不太像 PostgreSQL 经典慢性泄漏，更像应用侧大对象分配或瞬时工作集扩大。

需要区分两件事：

- 数据库进程内存：可能波动，但不等于泄漏
- 数据库磁盘占用：表是否清理决定是否持续增长

连接池相关：

- 代码默认：`database.max_open_conns=256`、`database.max_idle_conns=128`
- docker compose 示例：`50 / 10`

相关位置：

- `backend/internal/config/config.go`
- `deploy/docker-compose.yml`

补充说明：

- `.env.example` 里虽然写了 `POSTGRES_MAX_CONNECTIONS`、`POSTGRES_SHARED_BUFFERS` 等建议值，
- 但当前 `deploy/docker-compose.yml` 的 `postgres` 服务没有把这些参数显式传给 postgres 启动命令，
- 因此这些服务端调优变量在当前 compose 示例里不一定真正生效。

## 7. 有没有可能被攻击导致内存过大、服务器宕机

结论：有可能，主要是 DoS/资源耗尽风险，而不是传统“代码注入”。

### 7.1 大请求体并发攻击

成立条件：

- 攻击者拥有有效 API Key，或能访问到会完整读取 body 的接口

原因：

- 单请求体上限默认高达 `256MB`
- 读取逻辑是“整包读入内存”
- 后续还可能有解压、JSON 解析、错误日志采集的二次放大

结果：

- 少量并发大包就可能把 Go heap 顶上去
- 在容器/主机内存较小的情况下，可能直接 OOM 或触发严重 GC 抖动，表现为接口雪崩甚至进程退出

### 7.2 压缩请求放大攻击

成立条件：

- 攻击者能发送 `gzip/zstd/deflate` 的请求体

原因：

- 代码先读入压缩后的原文，再额外生成解压后的 buffer
- 即使解压后限制为 `64MB`，原始压缩体本身也可占到很大内存

结果：

- 单请求实际内存占用大于表面 payload 大小
- 多并发时比普通 JSON 包更容易拉高内存峰值

### 7.3 高基数会话键导致慢性内存增长

成立条件：

- 攻击者持续制造大量不同的兼容会话 key / prompt cache key

原因：

- `sync.Map` 只有访问时才删除过期项

结果：

- 这更像“慢慢涨”的内存消耗
- 未必立刻打挂，但长期运行可能抬高基线内存

### 7.4 错误风暴放大

成立条件：

- 攻击者用有效 key 故意制造大量失败请求

原因：

- 错误路径会额外记录请求体、错误体、上游错误信息

结果：

- 虽然队列是有界的，也会丢弃，但在打满之前仍会造成明显额外内存占用和 CPU 压力

## 8. 当前已有的保护

- 全局请求体限制：`http.MaxBytesHandler`
- 网关请求体限制：`RequestBodyLimit`
- 解压后 body 限制：`64MB`
- 应用文件日志轮转
- 运维错误日志队列是有界的，满了会丢
- 风控异步队列有逻辑上的队列长度控制
- auth 相关公开接口有 Redis rate limit

但需要注意：

- 网关大 body 上限仍然过大
- `ReadRequestBodyWithPrealloc` 依然是“整包读内存”
- gateway 主业务路径没有看到一个足够强的、面向大包 DoS 的前置限流/并发闸门
- 两个兼容会话 `sync.Map` 缺少主动过期清扫

## 9. 风险排序

### 高优先级

1. 默认 `256MB` 请求体上限过大
2. 请求体完整读入内存，且压缩体会先读 raw 再解压
3. 兼容会话 `sync.Map` 缺少后台主动清理

### 中优先级

1. 运维错误日志链路在失败风暴下会放大内存峰值
2. usage / 风控异步队列过大，容易积压对象

### 低优先级

1. `payment_audit_logs` 无自动清理，主要影响磁盘，不是当前尖峰内存主因

## 10. 建议

### 立即建议

1. 把 `server.max_request_body_size` 和 `gateway.max_body_size` 从 `256MB` 下调到更保守值
2. 对高风险接口增加更强的前置限流/并发限制
3. 给 `openaiCompatSessionResponses` 和 `openaiCompatAnthropicDigestSessions` 增加后台定时清理

### 排查建议

1. 观察暴涨时的：
   - Go heap
   - goroutine 数
   - GC pause / GC 次数
   - usage 记录队列长度
   - 风控队列长度
   - ops error 队列长度
2. 同时确认 PostgreSQL 当时的：
   - 活跃连接数
   - 大查询/排序
   - 进程 RSS

## 11. 最终判断

- 这次问题更像“资源耗尽型内存尖峰”，不是明确的持续泄漏。
- 代码里确实存在可以被放大的 DoS 面，尤其是大 body + 压缩 body + 并发请求。
- 如果攻击者有有效 API Key，或者内网/代理层没有更严格的限制，确实有机会把内存打高，严重时可能导致服务雪崩或 OOM。
