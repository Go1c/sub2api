---
name: upstream-sync-v0179-newest-gateway
description: origin/main 快进到 0.1.179 后，按「最近半个月、从最新窗口」把可落地的网关/鉴权修复带入 dev 的台账；整包 merge 已证实不可行。
metadata:
  type: record
  date: 2026-08-22
  status: 已实现，待 Review 合入 dev
---

# Upstream Sync 台账 — 2026-08-22 (v0.1.179 最新窗口)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | merge-upstream API 快进到 `67380eafd`（`0.1.179`），与 `upstream/main` 一致 |
| 同步方式 | 快进 `origin/main`；**不**整包 merge 进 `dev`；从最新窗口 cherry-pick / 适配移植 |
| 增量基线 | 上一版 fork main `baeac1f3d`（`0.1.177`，2026-08-15）→ `67380eafd`（`0.1.179`） |
| first-parent | 88 个落点（2026-08-17…08-21） |
| 全量 commit | 271 |
| 触及文件 | 582（+48053 / −4033） |
| `dev` tip（开工时） | `1f13ec9f7` |
| **禁止** | 整包 `git merge main` → `dev`；直推 `main` / `publish` |

用户约束：优先合最新提交；历史分叉（merge-base 仍是 2026-05 的 `a466e80ed`）不整包带入；外沿是最近半个月。半个月里 8-07…8-15 已部分经 v0176 / v0177 进过 `dev`，本轮从 **0.1.178/179 最新窗口** 下手。

## 整包 merge 试探（不做）

`git merge-tree origin/dev origin/main`：

- `dev` 独有 1216 提交，`main` 独有 2774
- 冲突 **829**（content 473 / add-add 345 / modify-delete 11）
- `backend/internal/service` 312 个冲突文件

结论与 2026-05-29 评估一致：不能把 `main` 整包合进 `dev`。

## 已纳入（T1 最新窗口）

干净 cherry-pick：

| 上游 | 内容 |
|---|---|
| `#5581` | passthrough 模型发现 |
| `#5759` | Codex usage probe 模型名清洗 |
| `#5801` | Chat Completions 缓冲读错误转 failover（fork 适配：failover 错误带上上游响应头，不跟 main 的变参账号副作用链） |
| `#5625` | Antigravity 官方 daily endpoint |
| `#5612` | Antigravity paid-tier endpoint |
| `#5632` | apicompat 流式 tool name 空串 |
| `#6049` | OpenAI sticky prefix system |
| `#6053` | 前端 token refresh 锁 CPU |
| `#5549` | OpenAI capabilities 空集合 |

手工适配：

| 上游 | 内容 |
|---|---|
| `#6016` | Prompt Audit `config_loaded` 只在真正变化 / 恢复失败时打日志 |
| `#5720` | 邀请码占用与建用户原子化（TOCTOU）；`user_repo.Create` 优先加入 `TxFromContext` |

交付分支：`sync/v0179-newest-gateway` → `--base dev`。

## 明确排除（不要当遗漏）

- 整包 merge upstream / 直推 `main` / `publish`。
- 国内部署链 `#5666` 及后续 CN quota / DeepSeek / header-override（fork 无 `PlatformComposite` / CN provider 面）。
- `#5888` OpenAI Responses 兼容大改（164 文件）。
- `#5925` Grok compatibility（51 文件）。
- 渠道分时价 / 档位乘数 / channel-monitor quota mode。
- `#5815` grok-4.6 `xhigh`：fork 没有 main 那套 `normalizeGrokReasoningEffortValue`，不能直接摘补丁。
- `#5708` 首页 model plaza、`#5838` 用户角色 Select（fork 编辑用户弹窗没有 role 下拉）。
- `VERSION` / sponsors / gitignore / star-history / Dockerfile Go 镜像钉。

## 验证

| 门禁 | 结果 |
|---|---|
| `go test -tags=unit ./internal/securityaudit`（含 config_loaded） | 通过 |
| `go test -tags=unit ./internal/service -run 'TestChatCompletionsBufferedResponsesReadError\|InvitationCode\|Register_Invitation'` | 通过 |
| `go test -tags=unit ./internal/pkg/apicompat ./internal/pkg/antigravity ./internal/pkg/openai` | 通过 |
| `go vet -tags integration`（改动包） | 通过 |
| 前端 `tokenRefresh.spec.ts` | 7/7 通过 |
| 全量 `go test ./...` | 未跑（既有时长问题） |

## 相关

- 工作流：[`../standards/workflow.md`](../standards/workflow.md)
- 上一轮：[`upstream-sync-v0177-codex-compact-group-usage.md`](upstream-sync-v0177-codex-compact-group-usage.md)
