---
name: upstream-sync-20260720-v0162
description: main 同步至 v0.1.162 后，将 0.1.160→0.1.162 增量分主题并入 dev 的评估台账与进度
metadata:
  type: record
  date: 2026-07-20
  status: T1+高价值T2+#243–#246 已合；#247 调度冷却待合；大块 A/B/C/E 评估结论已记；F 不合
---

# Upstream Sync 台账 — 2026-07-20 (v0.1.162)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | 已与 `upstream/main` 对齐 `e625ce3b3`（`0.1.162`） |
| 同步方式 | GitHub `merge-upstream` API（fast-forward） |
| 增量基线 | 上一版 fork main `57914967c`（`0.1.160`）→ `e625ce3b3`（`0.1.162`） |
| first-parent | **65** 个落点（含 VERSION/sponsors） |
| 全量 commit | **176** |
| 触及文件 | **377**（+23k / −2.4k） |
| `dev` tip（评估时） | `1f0fa14e3`（PR #221 wave2-continue-21 已合；PR #222 回归修复进行中） |
| **禁止** | 整包 `git merge main` → `dev`（trial 约 **505** 处 both-modified） |

## 分类（first-parent，关键词粗分）

| 档 | 数量 | 说明 |
|---|---|---|
| T1 安全 | 4 | step-up 默认关、ingress reject 过滤、prompt-audit fail-closed、S3 临时密钥拒绝持久化 |
| T1 基础设施 | 9 | compose/Dockerfile、BindEnv、client-ip、github release token、image storage env 等 |
| T2 网关 | ~32 | Grok / OpenAI / Codex / Anthropic / WS / sticky / scheduler 等（与 wave2 重叠大，逐主题） |
| T2 前端 | 5 | i18n、dark palette、branding SVG、channels scroll 等 |
| T3 计费/订阅展示 | 5 | plan currency symbol、renew expired、validity label、antigravity plan_type、balance modal |
| CHORE / OTHER | 10 | VERSION、sponsors、杂项修复 |

## 关键新能力（dev 状态）

- ✅ `ingress_reject` + auth abuse + auth cache outbox（#224）
- ⚠️ `image_storage`：已有 S3 实现；**后台 `image_storage_settings` 服务/UI 全量 port 仍缺**；本批只补 env 空默认
- ⚠️ client-ip 自定义 CDN 头 / `ForwardedClientIPHeaders` 全量 port 仍缺（与 fork IP 栈深度交织）
- ⚠️ branding SVG / batchImage i18n 等前端：按 T2 另开
- ⚠️ T2 网关大量 first-parent 落点与 wave2 重叠，**禁止**整包 merge main→dev

## Migration 编号映射

upstream `183`/`184` 在 fork 编号空间已被占用（fork 用 `900+` 扩展）。并入时：

| upstream | fork 拟编号 | 说明 |
|---|---|---|
| `183_ops_ingress_reject_aggregates.sql` | `915_ops_ingress_reject_aggregates.sql` | 在 `914_prompt_audit_full_prompt` 之后 |
| `184_auth_cache_invalidation_outbox.sql` | `916_auth_cache_invalidation_outbox.sql` | 同上 |

## 执行顺序

1. **T1a** 小安全修复：prompt-audit fail-closed（#4565）、S3 ephemeral key（#4638）
2. **T1b** security switches default-off / step-up 可关（#4526）— 注意 fork settings 深度改造
3. **T1c** ingress reject + auth abuse + auth cache outbox（#4515 主体）— 含 migration 重编号与 wire
4. **T1d** 其余 infra（compose / client-ip / github token / image storage env）按需
5. **T2** 网关/前端：对照 wave2 已合内容，只补 **dev 缺且有价值** 的 PR
6. **T3** 计费/订阅：保留 fork 额度池与支付语义，只 port 不冲突的展示/小修复

## 进度

| 批次 | 分支 | 状态 |
|---|---|---|
| 评估 + main sync | — | ✅ `origin/main` = `upstream/main` = `e625ce3b3` |
| T1a 小安全 | `sync/t1-security-v0162` | ✅ 已合 PR #223 |
| T1b step-up 开关 | — | ✅ 已在 `origin/dev`（#4526 等价能力齐全，跳过） |
| T1c ingress reject | `sync/t1c-ingress-reject-v0162` | ✅ PR #224 已合 `dev`；migration 915/916 |
| T1c follow-up | `fix/usage-log-insert-placeholders` | ✅ PR #225 已合：usage_logs `$57/$58`、golangci baseline、integration suite 漂移 |
| T1d 安全小补丁 | `sync/t1d-infra-v0162` | ✅ PR #226 已合：image_storage 空默认 + compose env 透传 |
| T1d GitHub token | `sync/t1d-github-token-v0162` | ✅ PR #227 已合：UPDATE_GITHUB_TOKEN |
| T2 小修复批 | `sync/t2-small-fixes-v0162` | ✅ PR #228 已合 |
| T2 网关修复批 | `sync/t2-gateway-fixes-v0162` | ✅ PR #229 已合：Codex models 401 下线、Anthropic stop_reason null、WS HTTP bridge failover、OpenAI quota 错误形态、Responses SSE/image intent、Grok 手动测试调度门 |
| T2 杂项批 | `sync/t2-misc-v0162` | ✅ PR #230 已合：sticky 同账号重试不计缓存、Codex list envelope、Claude `[1m]` 后缀、channels 滚动、balance modal dark、system update detach（fork 形态）、redis compose 续行、i18n 硬编码 |
| T2 model_not_found | `sync/t2-model-not-found-4508` | ✅ PR #231 已合：#4508 临时账号耗尽保留 503，不误报 model_not_found |
| T2 plan-validity | `sync/t2-plan-validity-4528` | ✅ PR #232 已合：#4528 套餐有效期文案去写死「天」 |
| T2 renew-expired | `sync/t2-renew-expired-4541` | ✅ PR #233 已合：#4541 admin assign 过期订阅续期（fork 额度池字段适配；gofmt 跟进） |
| ops PG local/dev | `sync/ops-pg-tuning-local-dev` | 🟡 PR #234 待合：local/dev compose 接 `POSTGRES_*`（与 yml 同款） |
| T2 OAuth system 去重 | `sync/t2-oauth-dedupe-4606` | 🟡 PR #235 待合：#4606（讨论 4=A） |
| T2 流失败上报 | `sync/t2-stream-fail-4520` | 🟡 PR #236 待合：#4520 客户端码 + Ops `CountTowardsSLA`（讨论 2=A） |
| T2 content_part | `sync/t2-content-part-4468` | 🟡 PR #237 待合：#4468（讨论 3=A） |
| T2 Agent Identity Team | `sync/t2-agent-identity-team-4572` | 🟡 PR #238 待合：#4572 匹配键 + skip OAuth expiry（讨论 1=A） |
| T2 Anthropic monitor | `sync/t2-anthropic-monitor-4537` | 🟡 PR #239 待合：#4537（讨论 5=A） |
| T2 WS mode 文档 | `sync/t2-ws-mode-docs-4522` | 🟡 PR #240 待合：#4522 全包 i18n+config+README（讨论 6=A） |
| T2 Dockerfile 交叉编译 | `sync/t2-dockerfile-cross-4507` | 🟡 PR #241 待合：#4507 BUILDPLATFORM + cache；保留 fork VERSION 回退（讨论 7 代决=A） |
| T2/T3 | — | **不整包合**；大块专题与产品差异项见下表 |

### 产品 / 运维讨论结论（2026-07-21）

| # | 项 | 结论 |
|---|---|---|
| 1 | #4572 Agent Identity | **A** 整包：Team 匹配键 + 跳过 OAuth 过期策略（否则导入失败） |
| 2 | #4520 流失败上报 | **A** 整包：稳定错误码 + Ops 逻辑 502 / SLA |
| 3 | #4468 content_part | **A** 整包 |
| 4 | #4606 OAuth system 去重 | **A** 整包（json_object / 混合 content 仍保留双份） |
| 5 | #4537 Anthropic monitor | **A** 整包 |
| 6 | #4522 WS 文档 | **A** 全包（含 README） |
| 7 | #4507 Dockerfile 交叉编译 | **A**（用户委托代决）：构建期 only，同架构无行为变化 |
| 8 | `docker-deploy.sh` raw 源 | **不动** — 部署已稳定；继续默认 upstream `Wei-Shaw/.../main/deploy` |
| 9 | 生产 PG 参数 | **B** — 只记录与仓库 compose 脱钩、**未上机验证**；不上机、不改线上 |

### 已具备 / 跳过（避免重复讨论）

| 项 | 结论 |
|---|---|
| #4630 IP 部分更新不清空 | 已具备，无需再 port |
| #4629 套餐货币符号 `$` | 不合（产品） |
| #4588 本体 docker-compose.yml PG 调优 | 已具备；缺口在 local/dev → PR #234 |
| local/docker-deploy PG 接线 | #234（待合） |
| publish 线上 PG | 与仓库 compose 脱钩；讨论 9=B 仅文档/台账，未上机 |
| `docker-deploy.sh` 改指 fork raw | 讨论 8=**不动**（稳定优先） |
| #4590 Trae/Codex 工具缓存 | 默认跳过（大） |
| #4543 / #4626 受保护视频 | 默认跳过（大） |
| #4553 WS turn lifecycle | 默认跳过（大） |
| #4547 / #4496 调度冷却 model/池 | 默认跳过（中心路径） |
| #4604 + BindEnv 全量 client-ip | 默认跳过（与 fork IP 栈深绑） |
| 上游 branding SVG | 不合（Lumio 品牌） |
| Grok Free/媒体、#4575/#4573/#4517 前端 | 中等，未做；按产品另开 |
| #4583 ops 报表邮件、#4558 antigravity plan_type | 中等，未做 |

### T2 misc 冲突 / 适配纪要

| 项 | 决策 |
|---|---|
| system update detach | 只 port `systemUpdateContext` + PerformUpdate 解耦 + 前端 update 15min timeout；**不**引入 main 的 `RollbackToVersion` / already-up-to-date / interface 抽象（fork 无该能力） |
| system_handler_test.go | fork 无此测试文件 → 不恢复 |
| redis local volume | 保留 fork `./redis_data:/data`（无 `:Z`）；只合 command 行续 `\` |
| plan currency / Trae / grok video / branding SVG / dark palette 全量 | 仍跳过（深度冲突或 fork 产品差异） |
| #4572 | 匹配键仅 `account:` + `resolveCodexImportExpiry` 对 Agent Identity 早退；`NewAccountHandler` 多一个 fork 参数 |
| #4507 | 合 BUILDPLATFORM/TARGET*/cache；**保留** fork `resolve-version.sh` 缺失时 VERSION 文件回退 |
| #4520 | `OpsStreamError.Code` + `CountTowardsSLA`；与 fork 已有 stream error 标记路径并集 |



### 大块专题决策（2026-07-21）

| 代号 | 上游 | 用户决定 | 评估结论 |
|---|---|---|---|
| A | #4553 WS turn lifecycle | **先评估** | **缺口大（~+1.3k）**：缺 `openai_ws_v2_passthrough_lifecycle_test.go`；main 相对 dev 改 `passthrough_relay` / `openai_ws_client_read` / passthrough adapter。fork 已有 WS HTTP bridge failover（#229）。**建议：无明确 WS turn 线上故障则不合**；要合需独立专题 + 全量 WS 单测。 |
| B | #4590 Trae/Codex/Claude Desktop 工具缓存 | **先评估** | **缺口中大（cache+chat_bridge ~+600 行 + EditAccountModal）**：main 有账号级 `grok_client_tool_cache_enabled`、Claude Desktop 指纹 opt-in、Trae 跨 turn 路由；dev 在 #4489 后仍是较简 Free 注入逻辑。**与刚合 Grok 四件套叠加**。**建议：仅当 Codex/Trae/Claude Desktop Free 缓存有真实投诉再合**；合则在 #243 之后单独 PR。 |
| C | #4543/#4626 受保护视频代理 | **先评估** | **缺口大（media content ~+750）**：main 有 `GrokMediaEndpointVideoContent`、安全代理/改写 URL、chained content proxy 测试；dev 路由已有 `/videos/:request_id/content` 入口痕迹，**内容代理实现与测试不齐**。**建议：若产品要代理 xAI 受保护视频内容则开专题合；否则跳过**（#243 媒体 eligibility/mapping 已够调度侧）。 |
| D | #4496+#4547 调度冷却 | **合** | ✅ 已开 PR #247：池模式尊崇 temp 规则 + 已知模型 `SetModelRateLimit` 隔离；未知模型仍账号级兜底。 |
| E | #4604 client-ip + BindEnv | **先评估** | **缺口大且深绑 fork**：main 有 `ForwardedClientIPSettings`、自定义 CDN 头列表、settings 热更新、多语言 README/EDGE_SECURITY；dev `pkg/ip` 以 Gin trusted proxy 链为主，合规 geo 路径敏感。**建议：不合默认**；若要 CDN 真实 IP，需对照 fork geo/限流/会话绑定单开专题，禁止整包。 |
| F | 上游 branding SVG | **不合** | 保持：Lumio 品牌，不直合上游 SVG。 |


### CI 基线债（全 PR 共有，不在本台账批次阻塞）
- `backend-security` govulncheck：S3 SDK / crypto/tls 调用图（#222/#223/#224/#225 均红）
- CLA 对上游作者 commit 的噪音（常 admin merge）


## 验收门槛（每批 PR）

- `go build ./cmd/server`
- `go test -tags=unit`（相关包全绿；改 repository 时至少跑触及包）
- 涉及 integration 源码：`go vet -tags integration ./...`
- 涉及 frontend：`cd frontend && pnpm typecheck && pnpm build`
- PR：`gh pr create --repo Go1c/sub2api --base dev`


## T1c 冲突处理纪要

### 已按「并集」处理（非漏合历史）

| 区域 | 决策 |
|---|---|
| API key auth | 保留 fork 中文余额不足文案 + 上游 header 尺寸/ingress Mark |
| API key errors | 保留 fork `ErrFallbackKeyInvalid` + 上游 `ErrAPIKeyAuthOverloaded` |
| Ops port/models/service | 保留 fork `LookupDeletedKeyAudit` / retry / request-body queue 路径；并入上游 runtime settings / queue body bounds / ingress aggregator 依赖 |
| Ops cleanup | 同时清理 `ops_retry_attempts` 与 `ops_ingress_reject_aggregates` |
| wire / wire_gen | 保留 fork 用户请求监控等 cleanup；并入 ingress/outbox/runtime refresh 生命周期 |
| migration | `183/184` → `915/916`（fork 编号空间） |

### 产品确认（已定）

| 项 | 结论 |
|---|---|
| `IgnoreInvalidApiKeyErrors` 强制 true | 接受（跟上游） |
| Ops 详情 `api_key_prefix` | 保留 |
| 恢复 3 个上游测试文件 | 已恢复并合入本 PR |
| `deploy/README.md` | 不恢复；保留 `EDGE_SECURITY.md` |
| 合并顺序 | #222 → #223 → #224（用户授权全部合入） |
