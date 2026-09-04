# 决策索引

`.spec/decisions/` 下每条 ADR 必须在此登记一行，否则 spec-lint 会判索引漂移。

| ADR | 一句话 |
|-----|--------|
| [`2026-09-04-lottery-sparse-wheel-and-wechat-promo.md`](2026-09-04-lottery-sparse-wheel-and-wechat-promo.md) | 抽奖转盘固定 8 格；公众号引导与中奖广告共用备份 S3 公开 HTTPS 海报 |
| [`2026-09-04-reseller-usage-correlation.md`](2026-09-04-reseller-usage-correlation.md) | 分销对账用独立 correlation 头/列 + 专用增量 export，不用计费 request_id |
