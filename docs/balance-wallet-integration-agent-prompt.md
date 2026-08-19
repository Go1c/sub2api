# 外部站接入钱包的 Agent 提示词

下面内容可直接交给 cchaven-control（或其它外部站）的开发 Agent。替换方括号中的项目上下文后执行。

```text
你正在 [外部站仓库/工作区] 中实现 LumioAPI 站内余额扣款接入。请先读取该仓库的 AGENTS.md、现有认证、订单、HTTP client、secret 管理和测试约定，再按现有架构完成实现与验证，不要修改 LumioAPI 仓库。

目标

在用户确认购买后，由外部站服务端调用 LumioAPI 钱包扣款。用户 Bearer JWT 决定扣款用户；外部站的 X-Balance-Client-Key 决定消费方身份。purpose 固定为 cchaven_monthly（如本项目获准的是其它值，以部署配置为准）。不做汇率换算，不创建 LumioAPI 订阅/套餐/订单。

安全红线

1. BALANCE_CLIENT_SECRET 是 bcs_ 开头的高权限服务密钥，只能存在服务端 secret manager / 环境变量中。绝不能进入浏览器、桌面端、移动端、前端 bundle、localStorage、响应体、埋点或日志。
2. 不接收、构造或发送 user_id 代扣。只把当前登录用户的 LumioAPI JWT 放入 Authorization Header。
3. 不记录 Authorization、X-Balance-Client-Key 或完整扣款请求体。日志只允许订单号、purpose、金额摘要、HTTP 状态、reason 和成功 txn_id。
4. 生产只走 HTTPS。若 LumioAPI 开启 session binding，按双方受信代理配置透传用户原始 IP（例如 X-Forwarded-For）和原始 User-Agent；不要盲信终端自报的转发头。

配置

- LUMIO_API_BASE_URL，例如 https://api.lumio.games
- BALANCE_CLIENT_SECRET，由 LumioAPI 管理员创建/轮换，只读加载
- BALANCE_PURPOSE=cchaven_monthly
- 合理的连接、请求和总超时；重试必须有界

调用契约

POST {LUMIO_API_BASE_URL}/api/v1/user/balance/debit
Authorization: Bearer <当前用户 LumioAPI JWT>
X-Balance-Client-Key: <BALANCE_CLIENT_SECRET>
Idempotency-Key: <外部站稳定订单号>
Content-Type: application/json

body（amount 必须发 JSON number，不能发字符串）：
{
  "amount": 19.90,
  "currency": "CNY",
  "purpose": "cchaven_monthly",
  "ref": "<同一订单号>"
}

金额使用 decimal/定点类型，要求 > 0 且最多两位小数，禁止先经过 binary float 再拼 JSON。currency 固定 CNY，它只是站内余额单位标签，不做 USD/CNY 换算。订单号同时作为 Idempotency-Key 和 ref；同一订单的所有重试必须保持 key、amount、currency、purpose、ref 完全一致。

成功响应 code=0，持久化 data.txn_id，并把该订单标记为已扣款。网络在响应前断开时，必须用同 key 和同 body 重放确认，不能创建新订单号或再次扣款。

错误处理

- 400 IDEMPOTENCY_KEY_REQUIRED / IDEMPOTENCY_KEY_INVALID / INVALID_BALANCE_DEBIT_REQUEST：本地契约错误，停止自动重试并告警。
- 401 INVALID_TOKEN / TOKEN_EXPIRED：刷新或重新取得用户 JWT 后，用同订单号和同 body 重试。
- 401 INVALID_BALANCE_CLIENT：服务端密钥缺失、停用或已轮换；停止用户侧自动重试，触发配置告警，绝不把密钥状态细节返回终端。
- 403 INSUFFICIENT_BALANCE：向用户展示余额不足并允许充值；充值后使用原订单号和原 body 重试，因为该错误不占用幂等 key。
- 403 USER_INACTIVE / PURPOSE_NOT_ALLOWED / BACKEND_MODE_ACTIVE：不扣款，按原因阻断或稍后恢复；PURPOSE_NOT_ALLOWED 触发配置告警。
- 409 IDEMPOTENCY_KEY_CONFLICT：同订单号关联了不同请求，停止重试并人工排查。
- 409 BALANCE_DEBIT_BUSY：遵守 Retry-After，使用同 key/body 做有界退避重试。
- 503 BALANCE_STORE_UNAVAILABLE 或网络超时：使用同 key/body 做有界指数退避；不要把未知结果当失败后换新 key。

实现要求

1. 在现有订单 service 边界新增 Lumio wallet client/adapter，不把 bcs_ secret 暴露到 controller DTO。
2. 在发起扣款前确保订单号已稳定持久化；扣款成功后原子或可恢复地保存 txn_id 和扣款状态。
3. 对同一订单的并发确认做本地串行/幂等保护，但最终以 LumioAPI 的 txn_id 重放结果为准。
4. HTTP client 只解析专用信封 {code,message,reason,data}；不要只按 HTTP 2xx 判断业务成功。
5. 保持现有 UI/产品语义；purpose 只是账本标签，不解释为 LumioAPI 套餐、订阅或 Claude 权益。
6. 如需展示历史，服务端可用用户 JWT 或 uat_ 调 GET /api/v1/user/balance/transactions；不得用 client secret 或 user_id 查询他人流水。

测试与验收

- 成功：请求头/body 精确，保存 txn_id，只扣一次。
- 相同订单并发与响应丢失重放：最终同 txn_id，外部订单只进入一次已扣款状态。
- 余额不足：不标记已扣款；充值后同订单号成功。
- TOKEN_EXPIRED 刷新后同 key/body 重试。
- INVALID_BALANCE_CLIENT、PURPOSE_NOT_ALLOWED、IDEMPOTENCY_KEY_CONFLICT 不进入无界重试。
- BUSY/503/网络超时遵守有界退避，所有尝试保持同 key/body。
- secret 不出现在日志、快照、前端产物、错误响应和测试 golden file。
- amount 0、负数、三位小数、字符串金额在本地被拒绝。
- 回归现有支付、订单取消、重复回调和登录流程。

完成后给出：修改文件、配置项、状态机/幂等说明、测试命令与结果、部署顺序，以及需要 LumioAPI 管理员提供的 client_id/一次性 secret/purpose。不要提交真实 secret。
```

完整服务端契约见 [balance-wallet-api.md](./balance-wallet-api.md)。
