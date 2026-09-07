---
name: openai-capacity-shed-default-retries
description: 容量降载同账号重试恢复为 pool 默认三次额外尝试，不再单独收成 1 次
---

# OpenAI 容量降载恢复默认同账号重试

- 状态: 生效

## 背景

2026-09-03 把 `server_is_overloaded` 的同账号静默重试从 pool 默认 3 次额外尝试收成 1 次，避免连打把 ChatGPT/Codex RPM 打满。副作用是整池过载时更快把 `Our servers are currently overloaded. Please try again later.` 回给用户。

## 决策

1. 容量降载不再设置 `SameAccountRetryMax`（0 = 沿用账号 `pool_mode`，默认 3 次额外尝试）。
2. 其余保持：请求级瞬时、不冷却账号、允许换号、耗尽回 `code=server_error` 并保留 overloaded 原文。

## 替代方案（未采用）

- **只把静默间隔从 500ms 拉长、次数仍为 1**：用户要求恢复原先的三次额外尝试。
- **整段 revert 容量降载处理**：会丢掉错误码改写；Codex 对 `server_is_overloaded` 判致命。
