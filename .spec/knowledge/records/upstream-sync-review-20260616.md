---
name: upstream-sync-review-20260616
description: upstream v0.1.136 同步上线前的独立代码复审记录与 go/no-go 结论——回看「为什么判定可上线、复审基线在哪」时查。
metadata:
  type: record
  date: 2026-06-16
  status: 归档
---

# Upstream v0.1.136 同步 — 上线前代码复审记录（2026-06-16）

> 本文件是对 [`upstream-sync-acceptance-20260615.md`](upstream-sync-acceptance-20260615.md) 所列改动的**独立代码复审**，
> 面向"下周推线上"的 go/no-go 决策。复审范围 = `dev` 分支上本次同步引入的全部非文档 commit
> （`631fce80~1..HEAD`，约 35 个功能 commit）。

## 结论

**未发现 bug。合并 / cherry-pick 解决正确（已做保真度对比）。可按计划上线。**

最高风险的核心计费路径（fork 自写）已逐行确认正确且有回归测试覆盖；DB 迁移为零停机在线设计。
对当前部署环境（**无独占分组 / 无 Bedrock / 无 OpenAI 账号**），实际行为变化面接近于零。

## 自动化验证（均真实执行，全绿）

| 验证 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./...` | exit 0 |
| 后端全量测试 | `go test ./...` | 40 包 ok，0 FAIL，0 panic |
| 集成 tag 检查 | `go vet -tags integration ./...` | exit 0，无诊断 |
| 前端类型检查 | `pnpm typecheck` | 通过 |
| 前端构建 | `pnpm build` | 通过 |
| 迁移编号 | — | 145 / 150 唯一，无碰撞 |

> 集成 tag 单独跑过，因为本仓库集成测试用 build tag 隔离，普通 build/test 抓不到其编译问题。

## 逐行复审项（按生产影响排序）

### 1. 核心计费路径 ✅（最高影响 + fork 自写，最该盯）
- `45a81c78` / `2034a5eb` 订阅额度池耗尽闸门 + 余额扣减下限。
- `deductUsageBillingBalance` 以 `min(amount, max(current,0))` 兜底——**余额绝不会被扣成负数**。
- 耗尽判定用 `IsCreditPoolExhausted()`，**正确区分"未配总池（QuotaLimitUSD==0）"与"池耗尽"**；
  `2034a5eb` 修掉了初版把纯窗口订阅误判耗尽的 over-gating bug。
- 回落语义已追到两个调用点：`CheckBillingEligibility` 显式回落余额闸门（耗尽+有余额→放行，
  耗尽+无余额→拒绝）；中间件 `ErrSubscriptionInvalid` 已在 `isFallbackableSubscriptionAuthError` 名单。
  两条命名回归测试在绿色套件内通过，端到端成立。

### 2. 合并 / cherry-pick 保真度 ✅
- CC 识别 `398a0c5b` 对上游 `d626ccce` **字节级一致**。
- 余额指针 `634e2c67` 对上游 `0560340b` 仅 hunk 行号偏移不同，改动代码一致。
- Bedrock `c47bdda4`、独占分组守卫 `8a78ce9c` 为 fork 自写：集成点 + 鉴权缓存快照填充已核，逻辑自洽。

### 3. 共享基础设施 ✅（与 provider 无关，人人经过）
- `81faaf40` postgres bootstrap 改用 maintenance DB；`6ccabce9` 连接池上限钳制；
  `402ad277` Redis Lua `replicate_commands()` 位置正确（脚本首行）；`23f2e750` 调度快照同步；
  `9bc0496b` `userRepo.Delete` 复用调用方事务；`8694c694` 幂等响应 UTF-8 截断回退。

### 4. 数据库迁移 ✅（零停机在线设计）
- `150_..._notx.sql`：`CREATE INDEX CONCURRENTLY`，runner（`migrations_runner.go:53,204`）确认识别 `_notx`
  后缀并**在事务外执行**，带 pg advisory lock、幂等。
- `145_ops_metrics_ttft_sample_count.sql`：元数据级加列（`ADD COLUMN IF NOT EXISTS DEFAULT 0` + `lock_timeout`）。

### 5. 安全修复 ✅
- `169184f8` CWE-204（越权返 404 防 ID oracle）；`61a02976` CWE-79（key 名 `html.EscapeString`）；
  `3784ec84` count_tokens 放行。

## 抽样 / 委托复核（未逐行，已交代）
- OpenAI #82 八个 commit + 用量窗口：子代理读 diff + 跑 build/test；当前环境不用 OpenAI，全量测试 + 集成 vet 绿。
- 纯 UI / 用量分析类（tooltip、下拉高度、用量明细 null 崩溃修复、admin 过滤、ops 指标、perf）：
  `--stat` 确认小范围新增 + 前端构建通过，爆炸半径小。

## 非阻断注意点（均非 bug）
1. **CWE-79**：含 `&`/`<` 的 key 名存为 HTML 转义，叠加 Vue 渲染转义可能双重转义（纯外观，上游行为）。
2. **连接池钳制**：`ConnMaxLifetimeMinutes=0`（原"永不过期"）会被改成默认 30 分钟——有意为之
   （永不过期正是它要治的云代理僵尸连接问题）。
3. `CREATE INDEX CONCURRENTLY` 通用注意：并发建索引中途失败会留 invalid 索引、`IF NOT EXISTS` 会跳过——
   标准运维常识，非本次引入。

## 复审者备注
- 验收清单标的 🔴 前五项前置条件，当前环境均不具备（无独占分组 / Bedrock / OpenAI），故对线上几乎无行为变化。
- 复审基线 commit：`74e9ed25`（dev / origin/dev，已同步，无待推送内容）。
