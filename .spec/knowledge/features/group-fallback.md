---
name: group-fallback
description: 分组级兜底: A 分组账号不可用时切到管理员配置的 B 分组重试
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 分组级兜底

简介：管理员可为 Anthropic / Antigravity 分组配置账号不可用时的兜底分组。请求先走 A 分组；若 A 当前账号报错且响应尚未开始写出,网关优先切到 B 分组重新调度。

## 背景 / 目标

- 典型场景是 A 分组只有一个账号,B 分组有多个账号；A 账号不可用时应尽快兜到底到 B,避免在 A 内继续退避重试。
- 兜底分组是管理员显式配置的运行时逃生口,不是用户主动绑定 B 分组。

## 设计

- 配置字段沿用 `groups.fallback_group_id_on_exhausted`。
- 运行时只在 Claude Messages 和 Responses 链路消费该字段。
- 触发条件:当前账号返回 `UpstreamFailoverError`,且响应尚未开始写出,且本次请求还没有使用过分组兜底。
- 切换后克隆当前 API Key 到 B 分组,重新解析订阅 / 额度池,再用 B 分组账号池正常选择账号。
- B 分组可以是专属分组；专属授权守卫只用于用户/API Key 直接绑定分组,不阻断管理员配置的运行时兜底。
- 响应已经开始写出时不兜底,避免把 A 和 B 的流式响应拼接给客户端。

## 已决策

- 分组兜底优先于同组 failover: A 当前账号已经报错且可安全重试时,先尝试 B；B 不可用再回到原有 failover / 密钥级兜底。
- 禁止链式分组兜底: B 不能再配置 `fallback_group_id_on_exhausted`,避免 A -> B -> C 的不可控链路。
- B 必须是同平台 active 分组,否则运行时跳过该兜底配置。

## 待解决

- 暂无。

## 相关

- `backend/internal/handler/group_fallback.go`
- `backend/internal/handler/gateway_handler.go`
- `backend/internal/handler/gateway_handler_responses.go`
- `frontend/src/views/admin/GroupsView.vue`
