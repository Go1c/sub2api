# Upstream Sync 评估台账 — 2026-06-15

## 状态

| 项 | 值 |
|---|---|
| 本次同步对象 | fork 远端 `main` |
| 同步方式 | GitHub `merge-upstream` API（即网页 "Sync fork"，因 `main` 是 protected/locked 分支，普通 push 被拒）|
| 结果 | `origin/main` fast-forward 到 `upstream/main` = `e34ad2b1`（v0.1.136），无历史改写；本地 `main` 已对齐 |
| 本批次 delta | `f18451e5..e34ad2b1`，共 **180 commit**（111 非 merge），388 文件 / ~25k 行 |
| 触及面 | backend 156 commit · frontend 27 · deploy 3 |
| dev 现状 | **未改动**。本文件仅为评估，不含任何代码合并 |

## 核心判断

`dev` 自 fork 点起已自定义 **670 个文件**。把 `main` 整体 merge 进 `dev`，trial merge 实测产生 **135 个冲突文件**，且全部砸在 fork 改造最重的区域（payment / subscription / billing / settings / openai-gateway / 前端 i18n+视图）。

**结论：不能用一次 `git merge main` 同步。** 建议沿用本仓库既有惯例 —— 按类别开 `upstream-<category>-20260615` 主题分支，从 `dev` 切出，分批 cherry-pick / 子集 merge，每批跑验证后单独 PR 进 `dev`。顺序 T1 → T2 → T3。

非合并提交按区域粗分（关键字归类，仅供规模参考）：
gateway 46 · frontend 11 · admin 11 · security 8 · payment 7 · infra 6 · 杂项/版本/文档 22。

---

## T1 — 安全 / 纯后端正确性修复（最该先拿，冲突低）

upstream 独占逻辑为主，与 fork 自定义重叠小，价值高。

| commit | 说明 | 备注 |
|---|---|---|
| `0ae33296` | CWE-79：`html.EscapeString` 净化 API key 名，防存储型 XSS | 安全 |
| `11b60171` | CWE-204：未授权访问 key 返回 404 而非 403，堵 ID oracle | 安全 |
| `c10598df` | 修复 idempotency 响应 UTF-8 截断 | 正确性 |
| `1b6a15b4` | db-pool 强制连接生命周期下限 | 稳定性 |
| `8a56c9fa` | 用 maintenance db 引导 postgres 连接 | 部署健壮性 |
| `585ff094` | Redis 3.2–4.x 下 Lua TIME 脚本复制兼容 | 兼容性 |
| `30c00a91` | 账号分组调度索引优化 | 性能 |
| `362f9e77` | 多实例后台任务 leader lock | 多实例正确性 |
| `415d08f2` / `b6c0706e` / `86d9b6bf` / `a4362963` | scheduler sticky health escape / 快照同步 / Codex used% 自愈 / sticky session group 校验 | 调度正确性 |
| `bba86f97` | `userRepo.Delete` 复用调用方事务 | 正确性 |

> 注意：`16bc8769` / `b65dde63` / `a8ffb052`（OpenAI 5h 用量百分比语义）这组在 upstream 内部有一次 revert 反复，且与 fork 的订阅/用量窗口逻辑相关 —— 归入 T2 一起评估，别单独拿。

## T2 — 有价值但与 fork 自定义重叠（逐主题分支 + 验证，需逐处确认）

冲突中等，必须人工核对是否覆盖 fork 改造。

- **OpenAI gateway / codex / responses / images / ws / bedrock / gemini（46 commit，最大块）**
  与 fork 的 codex 隐藏图像路由（`2e5c9938`）+ image 计费改造直接相撞。冲突文件：`openai_gateway_*.go`、`image_generation_intent.go`、`openai_images*.go`、`openai_ws_forwarder*.go`、`gateway_handler_*.go`。需要逐 commit 判断 fork 是否已自带等价逻辑。
- **settings / admin（11 commit）**：与 fork 的 idempotent settings upsert 热修（`hotfix/publish-settings-*`）相撞，冲突 `setting_service.go` / `setting_handler.go` / `admin_service.go` / `settings_view.go`。
- **ent schema**：`backend/ent/*`（`user.go`、`mutation.go`、`migrate/schema.go`、`client.go`）。**用 `go generate ./...` 重新生成，不要手解冲突。**
- **前端（11 commit）**：admin usage / error-requests 表、proxies 视图、`UserErrorDetailModal`、合规弹窗等。`en.ts`/`zh.ts` 冲突是机械的（两边各加 key），但量大。
- **5h 用量百分比语义那组**（见 T1 注）。

## T3 — 最高风险 / fork 命脉（建议单独专项，必要时手工 port）

- **payment / subscription / billing**：fork 在订阅额度池、余额闸（`45a81c78`）、Stripe 配置缓存（`1a497354`）、订阅支付计入余额（`e0dfe758`）上深度改造。upstream 的相关变更会正面冲突：
  - `04deb819` EasyPay 用 `trade_status` 查单
  - `1e2e8b1d` channel pricing 完全覆盖 image output token 价
  - `ef5ad0fb` 前端 usage 页展示 `image_output_tokens` 拆分
  - `eba20463` OpenAI OAuth token refresh enrich
  - `0acf00c4` admin 合规确认 gate
  逐个判断对 fork 是否需要；部分属 upstream/newapi 路线，不一定要并入。
- **CI / Docker（Zeabur 定制）**：`Dockerfile`、`deploy/Dockerfile`、`.github/workflows/*`。保留 fork 版本，仅按需取 upstream（如 Go toolchain bump `13468778`）。

---

## 建议的执行机制（落地步骤）

1. 每类开 `upstream-t1-safe-20260615` 等主题分支，从 `dev` 切出。
2. `git cherry-pick <commit>...` 或对一组文件做子集 merge；ent 改动改用重新生成。
3. 验证：前端 `pnpm typecheck && pnpm build`；后端 `cd backend && go build ./... && go test ./...`。
4. 每批单独 PR 进 `dev`（`--repo Go1c/sub2api`），互不阻塞。
5. 合并后在本文件追加"已处理"勾选，维持台账。

## 处理进度

### Phase 1 — T1 安全/正确性（分支 `upstream-t1-safe-20260615`，已验证）

已 cherry-pick 进 dev（`go build ./...` + 受影响包 `go test` 全绿）：

| upstream | 说明 |
|---|---|
| `1b6a15b4` | db-pool 连接生命周期下限 |
| `0ae33296` | CWE-79：净化 API key 名防存储型 XSS |
| `11b60171` | CWE-204：未授权 key 返回 404 防 ID oracle |
| `585ff094` | Redis 3.2–4.x Lua TIME 脚本复制兼容 |
| `b6c0706e` | 账号状态清除后同步 scheduler 快照 |
| `8a56c9fa` | 用 maintenance db 引导 postgres 连接 |
| `bba86f97` | `userRepo.Delete` 复用调用方事务 |
| `c10598df` | idempotency 响应 UTF-8 截断修复 |
| `30c00a91` | 账号分组调度索引（新 migration） |

**改放 T2 处理**（cherry-pick 冲突，均动 fork 自定义的 scheduler/gateway/wire）：
`415d08f2` sticky health escape · `86d9b6bf` Codex used% 自愈 · `a4362963` sticky session group 校验 · `362f9e77` 多实例 leader lock（动 subscription/payment expiry + wire）。

**bba86f97 的处理**：保留 `user_repo.go` 生产改动，**丢弃 `user_repo_delete_atomicity_integration_test.go`** —— 该集成测试调用 `APIKeyRepository.DeleteWithAudit`，dev 尚未同步该方法（属另一个上游 feature，待 T2）。注意：集成测试带 `//go:build integration`，本地 `go build`/`go test` 不会编译，必须用 `go vet -tags integration ./...` 做门禁（这是本次踩的坑）。

### 依赖说明（CI 红的两个既有项，非本批引入）

- `test`：#67（订阅额度池闸门）合入 dev 时引入回归，已由 **PR #70**（`fix/subscription-gate-tests`，新增 `UserSubscription.IsCreditPoolExhausted()`）修复并入 dev；T1 分支 rebase 后转绿。
- `backend-security`：dev 上 go1.26.3 标准库漏洞 GO-2026-5039 / GO-2026-5037，每个 dev commit 都红。修复需 go1.26.4 toolchain bump（upstream `13468778`，动 CI/Docker），归 **T3**，本批不处理。

### Phase 2a — T2 后端零散修复（非热路径）（分支 `upstream-t2-backend-fixes-20260615`）

**已 cherry-pick 进分支（12 commit + 1 处测试适配），验证：`go build ./...` ✅ · `go vet -tags integration ./...` exit 0 ✅ · `go test`（handler/service/repository/middleware）全绿。**

| upstream | 说明 | 风险标记（Review 重点） |
|---|---|---|
| `0560340b` | balance 改为指针类型 | ⚠️ **类型语义变更**：余额 nil ↔ 0 的区分，确认所有读取点对 nil 的处理 |
| `d626ccce` | 通过 billing block 识别 Claude Code 客户端（不只看 prompt） | ⚠️ **客户端识别逻辑变更**：影响哪些请求被判为 CC（计费/路由），需确认对现有流量分类无回归 |
| `bf1a2d6d` | Codex 用量统计对齐 reset 窗口 | ⚠️ 用量记账口径调整，确认与 fork 订阅/窗口统计一致 |
| `bf3787de` | 网关放行 Claude Code count_tokens | 低：放开一个原被拦的只读端点 |
| `fb0195f3` | 编辑账号时归一化固定额度窗口 | 低-中：账号额度窗口边界 |
| `32ef4711` | 允许的代理质量状态计为 pass 而非 warn | 低：代理健康分类 |
| `bc7ce185` | 管理员清空分组描述时正确持久化 | 低：bugfix |
| `69b46545` | ops TTFT 分位按流式样本量加权 | 低：监控指标 |
| `029b6d61` / `7386f38c` | 用量聚合拆分缓存创建/命中 token + 契约测试 | 低：统计维度新增 |
| `f20e6bf7` | 新增 `account_temp_unscheduled_count` 告警指标 | 低：新增监控 |
| `329414ea` | /admin/users 按 API Key 所在分组过滤（含测试适配） | 低：新增过滤；测试 `NewUserHandler` 调用适配为 dev 的 2 参签名 |

**本批 deferred（8 个，cherry-pick 冲突，转后续）：**
- `1a86c6ce` enforce exclusive group access for api keys —— ⭐ **安全相关**（api_key_auth 中间件，4 文件冲突），优先在后续单独处理
- `705fe7d8` delete user api keys with user —— ⭐ 引入 `APIKeyRepository.DeleteWithAudit`，并入后可补回 T1 丢弃的 bba86f97 集成测试
- `57d9e15e` 添加账号同步上游模型 · `ddf06335` ops 错误日志 key 归因 · `fe895273` 管理端错误请求页 · `bf24b611` /admin/usage 提速 · `aea2950b` LinuxDO 登录修复 · `b8c89c34` 契约测试字段 —— 多为 frontend / ops / 契约测试冲突，归 T2e 前端簇或后续

- [x] **T1 安全/正确性** — 9/13 已并（bba86f97 去掉集成测试），4 个转 T2
- [x] **T2 后端零散修复** — 12+1 已并（见上表），8 个 deferred
- [ ] T2 OpenAI 网关/调度/图像大簇（~33 commit，含 T1 deferred 的 4 个）—— 一次性专项
- [ ] T2 前端可观测性簇 + 上面 8 个 deferred
- [ ] T3 payment / CI（含 go1.26.4 toolchain bump 修 backend-security；专项评估，可能部分不并入）
