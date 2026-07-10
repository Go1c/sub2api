---
status: completed
---

# Fix OpenAI Version Header for GPT-5.6

## Task

修复普通 OpenAI `/v1/responses` 请求丢失 Codex `Version` 头的问题，使 Sol / Luna 请求不会被上游误判为旧版 Codex。

## Acceptance

- normal 与 passthrough builder 均透传客户端 `Version`。
- Codex 官方客户端缺少 `Version` 时补 `codexCLIVersion`。
- 非 Codex 请求不无条件补 Version。
- 不改变账号 custom User-Agent 语义。
- 聚焦测试、`gofmt`、`git diff --check` 通过。

## Verification Results

- 红：whitelist 断言失败，normal / passthrough 普通请求均未保留或补充 Version。
- 绿：Version whitelist、normal / passthrough 的客户端值保留、Codex 缺失回退、非 Codex 不补及既有 compact 行为测试全部通过。
- `gofmt` 与 `git diff --check` 通过。
