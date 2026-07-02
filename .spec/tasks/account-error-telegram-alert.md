---
status: completed
---

# Account Error Telegram Alert

新增独立后台功能：定时从线上 `ops_error_logs` 聚合最近窗口内异常账号，达到阈值后通过 Telegram 发送简短摘要；影响用户只展示邮箱，不展示用户 ID；不影响请求热路径。
