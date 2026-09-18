---
name: openai-oauth-ip-group
description: OpenAI OAuth IP 组选路：会话粘性、随机选择未尝试出口与两轮重试
metadata:
  type: doc
  level: L2
  status: 已实现
---

# OpenAI OAuth IP 组选路

- 已绑定且仍存活、未在本轮失败的会话继续使用原出口。
- 新绑定或失败后换出口时，从存活且本轮未尝试的 IP 中随机选择；候选随机排序后逐个检查并发槽位，跳过无法占用的出口，不按 IP ID 顺序请求。
- 一次请求内的 transient 重试复用现有 `tried` 集合。同一轮不重复已失败出口，整轮尝试完才清空集合进入下一轮；最多两轮，耗尽后切换账号。SOCKS / 运输层连不上（`authentication failed`、connection refused 等）同样只跳过该 IP、不把账号临时停调度；单出口账号的持久运输层失败仍停 10 分钟。
- 保留原有账号配额限流、会话绑定及并发逻辑。已有会话绑定的出口满并发时仍保持绑定；新绑定没有可用槽位时按现有逻辑失败关闭。
- 此处的轮次作用于单次请求的重试，不在账号所有成功请求之间建立全局轮次。

## JSON 导入默认值

- 两个入口分别接线：新建账号向导的 Codex auth.json / session JSON 导入，以及账号列表的 JSON 文件「导入数据」（`/admin/accounts/data`）。后者仅对 OpenAI OAuth 账号补齐这些默认值。
- 未手动指定时取列表中第一个 OpenAI / composite 分组和第一个 IP 组；没有可选项时不强制绑定。
- 保留用户手动选择的分组和代理。IP 组握手沿用既有逻辑，使用组内第一个代理。
- 白名单模式导入时合并最新内置 OpenAI 模型清单，等同「同步最新支持模型」；保留自定义模型与映射模式。
- `/admin/accounts/data` 在后端补齐 OpenAI OAuth 账号缺失的默认分组、IP 组、模型和监控配置；旧客户端直接提交原始 JSON 也适用，不依赖前端加工。显式配置优先，`proxy_ip_group_id: 0` 保留为不绑定。
- 文件导入通过 `group_ids` / `proxy_ip_group_id` 传递本地绑定；导出不写这些实例内 ID。文件已有代理绑定与自定义模型映射保留；加载默认分组或 IP 组失败时中止并提示错误。
- OpenAI OAuth JSON 文件导入统一使用并发 `8` 和 `extra.codex_fingerprint_mode: device`（仅设备），覆盖源文件中的这两项值；创建向导默认并发也为 `8`，允许手动调整。
- 导入请求覆盖写入：`extra.error_alert.enabled: false`（关 Telegram 报错监控）、`extra.turn_state_probe.enabled: true`、`extra.openai_oauth_responses_websockets_v2_mode: passthrough`（WS 透传）、`extra.account_traffic_control` 两开关开（严格 RPM + 自适应并发 observe，建议数字 60/5/1）。存量账号缺这些 extra 键时运行时行为不变。

相关实现：`backend/internal/service/openai_ip_group_proxy.go`、`openai_upstream_transport_error.go`；回归测试：`openai_ip_group_proxy_test.go`、`openai_ip_group_rate_limit_test.go`、`openai_upstream_transport_error_handle_test.go`（`unit` build tag）。

OAuth 新建 / PAT / Codex session 导入未指定并发时，服务端默认同为 `8`；显式并发值保留。JSON 文件导入仍按本地导入策略覆盖源实例并发值。
