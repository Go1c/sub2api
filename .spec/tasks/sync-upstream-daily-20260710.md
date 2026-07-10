---
status: completed
---

# Sync Upstream Daily Changes — 2026-07-10

## Task

将 `upstream/main` 在 2026-07-10（Asia/Singapore）从 v0.1.149 基线
`12d811bd76572836d6df6e1fa8aa5ff91be3b12e` 之后落入主线的全部 18 个提交，
按主题同步到 fork 的 `dev`，保留本仓库已有 GPT-5.6、Codex headers、image tool、
订阅与前端定制。

## T1 — OpenAI Transport and Compatibility

- 涉及范围：compact SSE、GPT-5.6 max reasoning、parallel tool calls、reasoning effort、
  WebSocket passthrough、image namespace、Grok、Codex identity、Codex client version。
- 验收标准：对应主线落点的净变更全部存在，已有 fork OpenAI 定制不回退，聚焦测试通过。
- 依赖：无。

## T2 — Billing and Recovery

- 涉及范围：GPT-5.6 cache billing、usage integrity、payment/redeem/subscription recovery、
  setup-token refresh 与相关 repository/service 测试。
- 验收标准：计费三桶、恢复幂等和订阅语义与上游当天主线一致，现有 fork 订阅逻辑保留。
- 依赖：T1。

## T3 — Admin, Frontend, Settings, and Version

- 涉及范围：英文 admin i18n、UserBreakdown request type、用户级 Fast/Flex 策略、
  admin settings 前端以及 `VERSION`。
- 验收标准：前后端契约一致，设置项与界面可构建，版本更新为当天上游版本。
- 依赖：T2。

## Final Acceptance

- 18 个 upstream first-parent 落点逐项审计，无遗漏；等价或已适配补丁明确记录。
- `git diff --check` 通过。
- 后端 unit / integration / vet / lint 与前端 typecheck / build 按仓库政策执行，
  环境阻断需准确记录。
- 同步结果经 PR 合入远端 `dev`，再按 `dev → release/* → PR → publish` 完成 promotion。
- 用户已有未跟踪或未提交文件不被修改或纳入提交。
