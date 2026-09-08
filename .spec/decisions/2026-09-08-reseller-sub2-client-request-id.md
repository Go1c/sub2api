---
name: reseller-sub2-header-must-be-client-request-id
description: 网关 X-Sub2-Request-ID 必须是下游 Client Request ID；异步计费必须拷贝该键
---

# 分销对账头必须是 Client Request ID

- 状态: 生效
- 取代: 部分取代 `.spec/decisions/2026-09-04-reseller-usage-correlation.md` 中「头非法则忽略、不失败请求」——未传 / 过长仍忽略；网关推理请求若带头且不是 UUID 则 400。

## 背景

下游出站已带 `X-Sub2-Request-ID`，但常见值是他们的内部 request id，不是 `ClientRequestID`。同时我们异步计费克隆 context 时只拷了 `ClientRequestID` / `RequestID`，`correlation_id` 落库为空。

## 决策

1. **异步计费拷贝 `Sub2RequestID`**，否则 export 对账键恒为 null。不改其他用户的 `GET /usage`、dashboard、计费 `request_id`。
2. **未传或过长仍忽略**，不 4xx。普通用户不带头，体验不变。
3. **网关推理请求若带头且不是 UUID**：HTTP 400，`INVALID_SUB2_REQUEST_ID`，提示改用 Client Request ID。`GET /v1/models`、`GET /v1/usage` 与控制面 `/api/v1` 不拦。
4. **不**把该头与我们自己生成的 `ClientRequestID` 做相等比较（我们进网关会另生成 UUID，对上只会误杀）。
5. **不改** `ClientRequestID` 中间件的生成逻辑。

## 替代方案（未采用）

- **带头但格式不对仍忽略**：对账会继续对空，下游不知道要换 ID。
- **比较 X-Sub2-Request-ID 与本进程 ClientRequestID**：几乎永远不相等，会拦掉正确的下游 UUID。
- **控制面 / 模型列表也 400**：会伤到未对账的额度查询和模型发现。
