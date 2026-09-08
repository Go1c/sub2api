# 决策索引

`.spec/decisions/` 下每条 ADR 必须在此登记一行，否则 spec-lint 会判索引漂移。

| ADR | 一句话 |
|-----|--------|
| [`2026-09-04-lottery-sparse-wheel-and-wechat-promo.md`](2026-09-04-lottery-sparse-wheel-and-wechat-promo.md) | 抽奖转盘固定 8 格；公众号引导与中奖广告共用备份 S3 公开 HTTPS 海报 |
| [`2026-09-04-reseller-usage-correlation.md`](2026-09-04-reseller-usage-correlation.md) | 分销对账用独立 correlation 头/列 + 专用增量 export，不用计费 request_id |
| [`2026-09-07-openai-capacity-shed-default-retries.md`](2026-09-07-openai-capacity-shed-default-retries.md) | 容量降载同账号重试恢复为 pool 默认三次额外尝试，不再单独收成 1 次 |
| [`2026-09-08-reseller-sub2-client-request-id.md`](2026-09-08-reseller-sub2-client-request-id.md) | 网关 X-Sub2-Request-ID 必须是 Client Request ID；异步计费必须拷贝该键 |
