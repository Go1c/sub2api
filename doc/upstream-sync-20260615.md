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
### Phase 2b — T2 前端/用量低风险簇（分支 `upstream-t2-frontend-20260616`）

**已 cherry-pick（5 commit + 1 测试适配），验证：`go build` ✅ · `go vet -tags integration` exit 0 ✅ · `pnpm typecheck` ✅ · `pnpm build` ✅ · `go test`（service/handler）全绿。**

| upstream | 说明 | 风险 |
|---|---|---|
| `b60d8bb4` | /admin/usage 支持查看已删除用户历史用量（含测试适配 NewUserHandler 2 参） | 低：管理端只读 |
| `bf24b611` | /admin/usage 打开/刷新提速 | 低：前端性能 |
| `0760cda9` | i18n 缓存命中/创建/命中率文案 | 低：纯文案 |
| `f5cecea5` | Select 下拉高度放开，避免选项截断 | 低：UI |
| `16bc8769` | 5h ResetsAt 对齐 SessionWindowEnd、过期窗口清零 | 低-中：用量窗口边界，确认与 fork 订阅窗口一致 |

**Deferred（cherry-pick 冲突，按子特性分组，需逐组仔细解）：**
- **失败请求/错误日志可观测性子特性**（互相关联，撞 fork 的 settings/ops 定制）：`cfb195c7`（14 文件大特性）+ `ddf06335` ops key 归因 + `fe895273` 管理端错误请求页 + `b8c89c34` 契约测试字段 —— 建议作为一个子特性一起解
- **`d662c973` claude-fable-5**（8 文件，撞 bedrock/antigravity/模型白名单）—— 模型新增，单独解
- **i18n / 单视图冲突**（撞 fork 定制，机械但需逐个核对）：`c256a544` 用量窗口 tooltip · ⭐`cf12bc52` 用量明细虚拟表可空字段崩溃修复（valuable bugfix）· `72c11216` bedrock_cc_compat 开关持久化 · `aea2950b` LinuxDO 登录修复 · `57d9e15e` 添加账号同步上游模型
- `af19d443` 代理有效期与失败回退（wire_gen 冲突）

### ⚠️ 关键发现：剩余"看似独立"的提交多是缺失子系统的尾巴

fork 早期分叉、只选择性同步过上游，导致 dev **整段缺失若干上游子系统**。剩下的高价值提交很多是这些子系统的"尾部 commit"，单独 cherry-pick 会因缺前置而失败/不可编译：

| 想要的 commit | 实际依赖的、dev 缺失的子系统 |
|---|---|
| `705fe7d8` 删用户带 API key | `ddf06335` 的删除审计链：`deleted_api_key_audits` 表 + `DeleteWithAudit`（dev 无此表/无此方法） |
| `1a86c6ce` 独占分组访问（安全） | API Key 分组可用性守卫子系统：`abortIfAPIKeyGroupUnavailable` / `validateAPIKeyGroupAvailable`（dev 完全没有） |

**含义**：clean cherry-pick 阶段基本到头了。剩余工作性质变了 —— 不是"挑 commit"，而是**按子系统整体搬迁 + 热路径三方合并**，冲突大、需 Review，属于要先决策范围的大件。

### 剩余大件（全部为「整段子系统搬迁」，每件需用户决策范围 + Review）

> **总结论**：clean cherry-pick / 机械冲突阶段**已彻底到头**。dev 落后 upstream 多代，以下子系统**整段缺失**，剩余有价值的上游 commit 几乎都是它们的尾部，无法单独并入——必须按子系统整体搬迁（含 migration / 签名 / 模型代际），属大工程。

- [ ] **OpenAI 网关/调度/图像大簇**（~33 commit，含 T1 deferred 的 4 个）—— 同一组热路径文件一次性三方合并；单实例可只取正确性/安全、跳过纯优化
- [ ] **失败请求/错误日志观测子系统**（`ddf06335`+`cfb195c7`+`fe895273`+`b8c89c34`+`705fe7d8`）—— 整段搬迁（含 `deleted_api_key_audits` 审计表 migration + `DeleteWithAudit`），撞 fork settings/ops
- [x] **API Key 分组可用性/独占守卫子系统**（`1a86c6ce` 安全）—— **已做（见 Phase 4c）**：移植 22ff1acd+1a86c6ce 合并效果，越权防护
- [ ] **auth OAuth token-pair 子系统**（`aea2950b` LinuxDO）—— dev 的 `LoginOrRegisterOAuthWithTokenPair` 是 5 参，上游 6 参（带 authSource）；需先搬签名变更
- [x] **模型代际 + Bedrock 兼容子系统**（`d662c973`）—— 数据层 Phase 4a + **Bedrock 执行层 Phase 4b 已做**（修 ValidationException bug）。仅剩非必核项不做：Bedrock region 模型 ID 映射、CLI 指纹版本 bump

> **关键耦合发现**：选项 2/3 不可分割 —— Bedrock sanitize 函数的调用点在 `gateway_service.go`（fork 深度改造 codex 路由的同一热路径文件）。模型「数据层」可独立同步（已做），但「执行层」（Bedrock 请求处理 + 调用点）必须与 OpenAI 热路径一起做 fork↔upstream 大对齐。
- [x] **i18n / 单视图机械冲突** —— 见 Phase 2c（2 并 / 1 跳过 / 3 转后续）
- [ ] **T3 payment**（fork 命脉，挑着手工 port）
- [x] **T3 go1.26.4 toolchain bump**（`13468778`）—— 见下

### Phase 3a — go1.26.4 toolchain bump（分支 `upstream-t3-go1264-toolchain-20260616`）

cherry-pick `13468778`：go.mod `go 1.26.4` + 3 个 Dockerfile + 3 个 CI workflow 全部统一到 1.26.4。
- **冲突**：`backend/Dockerfile` 之前 fork 停在 `golang:1.25.7-alpine`，解为 `1.26.4`（go.mod 已要求 1.26.4，构建镜像必须 ≥ 它）。
- **验证**：`go build ./...`（go1.26.4 自动拉取）✅；`govulncheck ./...` → **No vulnerabilities found**，GO-2026-5039/5037 消除。
- **效果**：一直红的 `backend-security` 应转绿（首次）。单 docker 部署，无 PaaS 顾虑。

### Phase 2c — i18n / 单视图机械冲突（分支 `upstream-t2-views-i18n-20260616`）

**已并（2 commit，手解冲突），验证：`go build` ✅ · `go vet -tags integration` exit 0 ✅ · `pnpm typecheck` ✅ · `pnpm build` ✅。**

| upstream | 冲突解法 |
|---|---|
| ⭐`cf12bc52` 用量明细可空字段崩溃 | 核心修复 `DataTable.vue`（scrollRect→空白根因）+ `types/index.ts` **clean 应用**；`UsageView.vue` 只解模板：保留 dev 无 tab 结构（丢弃 dev 没有的 `errorViewEnabled`/`activeTab` 错误页 tab），保留 `?? 0` 空值防御 + 虚拟化 props |
| `c256a544` 用量窗口 tooltip | i18n 上下文漂移：dev `upstreamBalance` 块与上游 `usageWindowsHint` 同级 key，**keep-both** |

**跳过 / 转后续：**
- `72c11216` bedrock_cc_compat 开关 —— dev 完全没有该开关特性，加进来是未使用变量，**跳过**
- ⚠️ `aea2950b` LinuxDO 登录 —— **原以为前端独立，实则也动 `auth_linuxdo_oauth.go`**：调用 `LoginOrRegisterOAuthWithTokenPair` 用了 6 参（带 authSource），dev 签名只有 5 参 → 又是缺失上游签名的尾巴。**撤出本批**，转「auth OAuth 子系统」单独评估
- `57d9e15e` 添加账号同步上游模型 · `af19d443` 代理有效期/失败回退 —— 动 backend，转后续

> **教训**：cherry-pick 即使「看起来是前端」也可能携带 clean 应用的 backend 文件（引用了 dev 没有的签名）。**每批都必须跑 `go build` + `go vet -tags integration`，不能只跑 pnpm**（aea2950b 因此一度让 #77 的后端 test/golangci 红，已撤出）。

### Phase 4c — API Key 独占分组访问守卫（分支 `upstream-apikey-group-guard-20260616`）

**必核(安全)**：API key 绑定的独占分组在用户被移出 allowed_groups / 分组停用删除后仍可用 → 越权。移植 `22ff1acd`+`1a86c6ce` 合并效果。

- 中间件：`abortIfAPIKeyGroupUnavailable`（分组删/停用）+ `abortIfAPIKeyGroupNotAllowed`（用户已无独占分组授权），插在 USER_INACTIVE 检查后；订阅型分组豁免
- auth snapshot：加 `User.AllowedGroups` + `Group.IsExclusive`，版本 9→10（强制旧缓存刷新）
- repo：`GetByKey`/`GetByKeyForAuth` 加载 AllowedGroups 边 + IsExclusive，`apiKeyEntityToService` 填充 AllowedGroups
- admin：用户 allowed_groups 变化时失效 auth 缓存（`sameInt64Set`），与 fork 的 `InvoiceEnabled` keep-both
- **改造**：丢弃 `MarkOpsClientBusinessLimited`（dev 按响应码自动分类 business-limited）；**不移植** 22ff1acd 的 group_repo DeleteCascade（dev 删分组时清空 key 的 group_id，守卫仍覆盖"停用"和"移出授权"两核心场景）
- 验证：`go build` ✅ · `go vet -tags integration` exit 0 ✅ · **`go test -tags unit`**（中间件含新越权测试 + service + repository）✅ · 普通 `go test` ✅
- **踩坑**：dev 的 middleware 测试带 `//go:build unit`，本地默认 `go test` 不编译 → 验证必须带 `-tags unit`（与 `-tags integration` 同理）

### Phase 4b — Bedrock 执行层（分支 `upstream-bedrock-cc-compat-20260616`）

**必核(bug)**：fork 用 AWS Bedrock，缺 thinking/CC sanitize → Bedrock 账号跑 Claude Code/opus-4.x/fable-5 报 ValidationException。

- 移植 `sanitizeBedrockThinking`/`sanitizeBedrockToolUseIDs`/`sanitizeBedrockCCFields`/`sanitizeBedrockCCBetaTokens` + `isBedrockOpus47OrNewer`/`isBedrockFable5` 到 `bedrock_request.go`（依赖的 autoInject/filter/claudeVersionRe dev 已有；新增 `logger` import）
- 在 `forwardBedrock` 的 `PrepareBedrockRequestBodyWithTokens` 前**对所有 Bedrock 请求无条件调用**（dev 无渠道级 `bedrock_cc_compat` 开关，且这是修报错，默认应开）
- **协调点**：`sanitizeBedrockCCFields` 改造版**不删 context_management**（交给 fork 既有的 `sanitizeBedrockFieldsForBetaTokens` 按 beta token 精细决定）、**不重设 anthropic_version**（Prepare 已设），避免双删/重复
- 验证：`go build` ✅ · `go vet -tags integration` exit 0 ✅ · `go test`（含移植的 thinking/toolUseID/CCFields 测试 + fork 既有 context_management 测试）全绿

### Phase 4a — 模型代际「数据层」可独立部分（分支 `upstream-models-opus48-fable5-20260616`）

**手工提取（非 cherry-pick），给 dev 加 `claude-opus-4-8` + `claude-fable-5` 两个模型到数据层；验证：`go build` ✅ · `go vet -tags integration` exit 0 ✅ · `go test`（antigravity/domain/service 映射）✅ · `pnpm typecheck` ✅ · `pnpm build` ✅。**

| 文件 | 改动 |
|---|---|
| `domain/constants.go` | `DefaultAntigravityModelMapping` 加 2 模型 |
| `pkg/antigravity/claude_types.go` | `claudeModels` 加 2 模型 |
| `pkg/antigravity/request_transformer.go` | `modelInfoMap` 加 fable-5/opus-4-8/opus-4-7；`isAntigravityOpusHighTierModel` 加 opus-4-8 |
| `pkg/claude/constants.go` | `DefaultModels` 加 2 模型（**未动 CLI 指纹版本**） |
| `useModelWhitelist.ts` | claude/antigravity 白名单数组加 2 模型（**未带 `splitModelMappingObject` 功能代码**） |

为什么手工提取而非 cherry-pick：upstream 这些文件的 diff 夹带了 fork 不需要/缺前置的功能代码（`splitModelMappingObject`、Bedrock 函数体、CLI 指纹 bump、UseKeyModal 的 codex 配置模板），整体并入会污染或编译失败。

### 已落地汇总（截至 2026-06-16，dev CI 全绿）

| PR | 内容 | commit 数 |
|---|---|---|
| #70 | 订阅闸门回归修复（用户本人） | 1 |
| #69 | T1 安全/正确性 | 9 |
| #72 | T2 后端零散修复 | 12+1 |
| #73 | T2 前端/用量低风险 | 5+1 |
| #75 | T3 go1.26.4 toolchain（修绿 backend-security） | 1 |
| #76 | form-data ≥4.0.6（修绿 frontend-security） | 1 |
| #77 | T2c i18n/单视图（手解冲突） | 2 |

clean-cherry-pick / 机械冲突阶段共并 **~30 个上游 commit + 适配/修复**，全部经 `go build`/`go test`/`go vet -tags integration`/`pnpm typecheck`/`pnpm build` 门禁。**dev CI 现已全绿**（backend-security、frontend-security 均已修）。

**到此为止**：upstream v0.1.136 中所有「自包含、低风险」的修复已全部并入。剩余都是上方列出的「整段子系统搬迁」大件，每件需用户决策是否要该特性、并接受较大改动范围与 Review 成本。
