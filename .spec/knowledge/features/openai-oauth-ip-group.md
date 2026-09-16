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
- 一次请求内的 transient 重试复用现有 `tried` 集合。同一轮不重复已失败出口，整轮尝试完才清空集合进入下一轮；最多两轮，耗尽后切换账号。
- 保留原有账号配额限流、会话绑定及并发逻辑。已有会话绑定的出口满并发时仍保持绑定；新绑定没有可用槽位时按现有逻辑失败关闭。
- 此处的轮次作用于单次请求的重试，不在账号所有成功请求之间建立全局轮次。

## JSON 导入默认值

- 两个入口分别接线：新建账号向导的 Codex auth.json / session JSON 导入，以及账号列表的 JSON 文件「导入数据」（`/admin/accounts/data`）。后者仅对 OpenAI OAuth 账号补齐这些默认值。
- 未手动指定时取列表中第一个 OpenAI / composite 分组和第一个 IP 组；没有可选项时不强制绑定。
- 保留用户手动选择的分组和代理。IP 组握手沿用既有逻辑，使用组内第一个代理。
- 白名单模式导入时合并最新内置 OpenAI 模型清单，等同「同步最新支持模型」；保留自定义模型与映射模式。
- 文件导入通过 `group_ids` / `proxy_ip_group_id` 传递本地绑定；导出不写这些实例内 ID。文件已有代理绑定与自定义模型映射保留；加载默认分组或 IP 组失败时中止并提示错误。
- 导入请求写入 `extra.error_alert.enabled: false`，默认关闭 Telegram 报错监控。

相关实现：`backend/internal/service/openai_ip_group_proxy.go`；回归测试：`openai_ip_group_proxy_test.go`、`openai_ip_group_rate_limit_test.go`（`unit` build tag）。
