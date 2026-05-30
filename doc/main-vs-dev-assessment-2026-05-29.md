# Main 合并到 Dev 的深度评测

日期：2026-05-29

## 1. 结论先行

先给结论，不绕弯：

1. 不建议直接把 `origin/main` 整体 merge 到当前 `dev` 并尝试一次性收口。
2. 值得吸收的是 `origin/main` 里的 bug fix、协议兼容、计费正确性、调度稳定性和部分性能优化。
3. 不值得在第一轮硬并入的是支付新能力、多币种、与现有 `dev` 产品线正面冲突的上游功能。
4. 最稳妥的路径不是“分支级合并”，而是“按主题回收 upstream 能力”。

如果一定要做一次 merge 试验，也应当只在临时集成分支上做，用来暴露冲突，不应直接在日常 `dev` 上做。

## 2. 本次评估的基线

评估对象：

- `origin/main`
- `origin/dev`

当前 refs：

- `dev`: `b40007a8` (`fix: harden subscription payment edge cases`)
- `origin/dev`: `eb270105` (`fix: localize subscription and balance prompts`)
- `origin/main`: `f18451e5`
- `upstream/main`: `f18451e5`

直接含义：

- 你的 `origin/main` 已经与 `upstream/main` 对齐。
- 你本地 `dev` 还落后 `origin/dev` 6 个提交。
- 真正应该比较的是 `origin/main` 对 `origin/dev`，不是当前本地 `dev` 对一个不存在的本地 `main`。

merge-base：

- `a466e80ed6e9f498108286203a8a9e3a2d75e58a`

## 3. 定量结果

分叉规模：

- `origin/main` 独有提交：`361`
- `origin/dev` 独有提交：`272`

只看 `main` 自分叉点以来带来的内容，即：

- `git diff origin/dev...origin/main --shortstat`

结果：

- `682 files changed`
- `72,144 insertions`
- `8,466 deletions`

只看 `dev` 自分叉点以来带来的内容，即：

- `git diff origin/main...origin/dev --shortstat`

结果：

- `578 files changed`
- `106,246 insertions`
- `10,610 deletions`

说明：

- 这不是“`main` 比 `dev` 多几个修复”的量级。
- 这是两条都持续演进了一段时间的分支。

`main` 侧热点目录（按 `git diff --dirstat=files,5 origin/dev...origin/main`）：

- `backend/internal/service/` 36.0%
- `backend/internal/handler/` 13.1%
- `backend/internal/` 9.9%
- `frontend/src/components/` 8.2%
- `backend/ent/` 8.0%
- `frontend/src/views/` 7.3%
- `backend/internal/repository/` 5.2%

一个好消息：

- `frontend-dashboard/` 在 `origin/dev...origin/main` 中没有差异。

也就是说，LumioAPI 独立前端不是这次合并的主要冲突源。

## 4. `main` 带来的东西，按优先级划分

下面只按你的目标排序：先 bug 修复、性能优化、正确性；新功能如果冲突大，可以放。

### 4.1 第一优先级：应该优先吸收

这些内容即使不是“新功能”，也会直接影响运行正确性、协议兼容或调度稳定性。

#### 网关与协议正确性

代表提交：

- `d7bed40d` 修复 OpenAI WS 兼容性与 usage 统计
- `32ea9cfe` API Key Responses 回退 SSE body
- `20f53407` Responses -> Chat 转换补齐 `completion_tokens_details`
- `89dffdd2` Anthropic -> Responses token 口径修正
- `0a521f09` Gemini Messages 流式 `tool_use` / `text` 块闭合修复
- `27600b1d` 过滤 `count_tokens` generation-only 字段
- `ed1b57c5` 端点能力 gate routing
- `37044b83` 端点能力 UI/文案纠偏

判断：

- 这类改动基本都偏 correctness，不是花哨功能。
- 如果不吸收，后续最容易出现的是“某些模型/端点偶发不兼容、计费偏差、usage 统计错位”。

#### 调度与可用性

代表提交：

- `1e406fed` 优化 OpenAI 账号冷却调度
- `08061717` OpenAI WS rate limit failover
- `a31b5074` 模型 404 仅冷却账号-模型组合
- `ead471d6` 按 5h/7d 用量阈值自动暂停账号调度
- `56e96fdd` 并发获取失败分类修正

判断：

- 这是上游最有工程价值的一批改动。
- 它们能提升账号池稳定性，降低错误冷却和错误封禁范围。

#### 计费正确性与 usage 口径

代表提交：

- `b9509e82` `cache_read` 长上下文倍率
- `ed2aac25` `cache_creation` 长上下文倍率
- `f7ac5e59` 保留 chat responses usage billing
- `2bd3125d` Preserve usage request context
- `bb4c1abe` image billing size normalization

判断：

- 这批改动不一定会在页面上立刻显眼，但会直接影响账单和对账可信度。

#### Auth / 数据完整性

代表提交：

- `6aec5050` OAuth 401 时不覆盖 `credentials JSONB`
- `11fe7de9` 重新授权保留 `Extra`
- `0af44ce4` 反代部署下客户端 IP 识别修复

判断：

- 这是偏安全/数据保全的修复。
- 吸收优先级高。

### 4.2 第二优先级：有价值，但要看你是否要产品化

这些内容不是不能要，而是会带来 schema、Ent、管理端 UI 一起变。

#### 用户 x 平台配额体系

代表提交：

- `6b39b344` 用户 x 平台 USD 配额
- `f7f5e338` 配额 DB 写聚合 flusher
- `06fca662` 无配额用户 preflight sentinel 回填

判断：

- 这是“功能 + 性能优化”绑在一起的一组改动。
- 如果你最终不打算采用平台配额，`flusher` 和 `sentinel` 本身也就没有意义。
- 如果要采用，就必须成组吸收，不能只拿优化补丁。

#### 分组自定义 `/v1/models`

代表提交：

- `f597c158` 支持自定义 `/v1/models` 模型列表

判断：

- 有产品价值，但会碰 `group` schema、handler、前端页面和 DTO。

#### Channel Monitor 的 OpenAI API mode / 模板

代表提交：

- `3eff5f51` OpenAI API mode migration
- `b685fe69` 内置 OpenAI 检测模板

判断：

- 功能合理，但会跟你自己的监控扩展、页面和 migration 正面冲突。
- 建议放到第二轮，不要和核心 bugfix 混在一起。

#### Auth/注册体验增强

代表提交：

- `a5b9b68b` 邮箱白名单后缀通配符
- `b19da9c7` DingTalk OAuth
- `a613a587` 订阅到期邮件提醒开关

判断：

- 都是可用能力，但不是第一轮必须项。

#### OpenAI embeddings gateway

代表提交：

- `ccace69d` Add OpenAI embeddings gateway

判断：

- 技术上有价值，但属于能力面扩展，不属于“先修 correctness”那一层。

### 4.3 第三优先级：第一轮建议直接放弃

如果目标是先拿 bugfix/perf，而不是全量追平 upstream，这些东西建议先不碰：

- Airwallex / 多币种支付链路
- 与支付 UI 深度耦合的新页面/测试
- Sponsor README / workflow / 文档噪音
- 纯展示性后台增强，例如某些管理页列展示小改动

原因很简单：

- 支付是当前 `dev` 冲突最密集的区域之一。
- 你的 `dev` 已经在支付、订阅、余额、信用池方向做了大量 fork 级开发。
- 在第一轮把上游支付栈并进来，收益不高，冲突成本极高。

## 5. `dev` 分支必须保护的东西

`dev` 不是单纯“落后于 upstream 的分支”，它已经是你自己的产品线。

`dev` 独有重点主题包括：

- 订阅信用池 / balance 混合支付
- site messages
- support chat
- invoice request / export
- lottery
- Mapay 支付通道
- upstream balance 相关 UI
- external login handoff / public pages
- OpenAI image routing 策略
- `frontend-dashboard/`

这意味着：

- 任何 merge 决策都要默认保护这些能力。
- 在冲突区域里，不能用“上游较新”作为默认取舍标准。

## 6. 结构性阻塞点

### 6.1 migration 编号已经正面撞车

这是最大的硬阻塞。

`main` 的 136-144：

- `136_add_dingtalk_provider_type.sql`
- `136_remove_ops_retry_replay.sql`
- `136_usage_log_image_size_metadata.sql`
- `137_redeem_code_expires_at.sql`
- `138_channel_monitor_openai_api_mode.sql`
- `139_seed_openai_monitor_templates.sql`
- `140_extend_user_provider_default_grants_check.sql`
- `141_subscription_expiry_notify_enabled.sql`
- `142_user_platform_quotas.sql`
- `143_group_models_list_config.sql`
- `144_add_opus48_to_model_mapping.sql`

`dev` 的 136-143：

- `136_ops_user_request_monitoring.sql`
- `137_add_site_messages.sql`
- `138_add_invoice_requests.sql`
- `139_add_group_expose_upstream_model.sql`
- `140_add_lottery.sql`
- `141_subscription_credit_pool.sql`
- `142_scheduler_outbox_subscription_notify_index.sql`
- `143_add_subscription_plan_purchase_notice.sql`

直接含义：

- 不能直接 merge migration 文件。
- 也不能简单 cherry-pick 上游 migration 文件名。
- 后续若要移植 `main` 的 schema 变更，应该新建 fork 自己的 migration 文件，而不是复用 upstream 已占用文件名。

建议：

- 从现在开始为 fork 预留单独 migration 命名空间，例如 `900_*.sql`。
- 原因是当前迁移执行器按文件名排序，并按文件名 + checksum 记录；继续共用 14x 区间，只会反复撞车。

### 6.2 Ent / schema 漂移不是局部修补能解决的

`main` 侧涉及：

- `auth_identity`
- `channel_monitor`
- `group`
- `redeem_code`
- `usage_log`
- `user`
- 新增 `user_platform_quota`

`dev` 侧涉及：

- `api_key`
- `channel_monitor`
- `group`
- `payment_order`
- `subscription_plan`
- `usage_log`
- `user`
- `user_subscription`
- 新增 `lottery_*`
- 新增 `site_message`
- 新增 `subscription_credit_ledger`

判断：

- 这已经不是“改几个 Ent 生成文件”。
- 正确顺序必须是：定目标 schema -> 新 migration -> 重生 Ent -> 补 repository/service/handler。

### 6.3 真实 merge 冲突数量已经足够说明问题

我做了一次无 checkout 的 merge simulation：

- `git merge-tree --write-tree --name-only origin/dev origin/main`

显式冲突文件数：`95`

按区域看：

- `backend/internal/`: 49
- `frontend/src/`: 32
- `backend/ent/`: 6
- `backend/cmd/`: 2
- 其余包括 `backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/vitest.config.ts`、`Dockerfile`

最危险的冲突簇：

#### 后端

- `backend/internal/service/gateway_service.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/payment_*`
- `backend/internal/service/setting_service.go`
- `backend/internal/service/channel_monitor_*`
- `backend/internal/service/auth_service.go`
- `backend/internal/handler/payment_handler.go`
- `backend/internal/handler/admin/setting_handler.go`
- `backend/internal/server/http.go`
- `backend/internal/server/routes/admin.go`

#### 前端

- `frontend/src/components/payment/*`
- `frontend/src/views/user/PaymentView.vue`
- `frontend/src/views/user/SubscriptionsView.vue`
- `frontend/src/views/user/UsageView.vue`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/views/admin/RiskControlView.vue`
- `frontend/src/views/admin/GroupsView.vue`
- `frontend/src/router/index.ts`
- `frontend/src/types/payment.ts`
- `frontend/src/i18n/locales/{en,zh}.ts`

一个关键观察：

- 冲突最密的地方，恰好就是你现在最不想一次性重写的地方：支付、订阅、设置、风控、网关、调度。

## 7. 对“直接 merge main 到 dev”的判断

我的判断非常明确：

- 可以做试验性 merge。
- 不应该把它当作正式集成路径。

原因：

1. 你当前本地 `dev` 还落后 `origin/dev` 6 个提交。
2. 工作区还有未提交改动。
3. 95 个显式冲突已经足够说明这不是“人工收几个文件”能搞定的事情。
4. migration 与 Ent 的问题，即使把文本冲突都解开，也不代表分支真的可运行。

所以，“直接 merge”最多用来回答一个问题：

- 哪些主题重叠最深，应该改成按专题吸收。

## 8. 推荐策略：按主题吸收，而不是按分支吞并

### 8.1 推荐执行顺序

第一批，只做高价值 bugfix / correctness / perf：

- OpenAI / Responses / WS 兼容修复
- usage / billing 口径修复
- scheduler / cooldown / failover 修复
- OAuth / credentials / Extra 保全修复

第二批，再看中等价值但有 schema 成本的能力：

- 风控阈值
- `/v1/models` 自定义
- Channel Monitor API mode
- 注册/Auth 增强

第三批，最后才讨论真正的新能力：

- 用户 x 平台配额
- embeddings gateway
- 其它 upstream UI/后台增强

### 8.2 第一轮建议明确放弃的上游内容

如果你的目标真的是“先修稳”，第一轮就明确不收：

- Airwallex / 多币种支付
- 与上游支付栈绑定的 UI/测试
- sponsor / workflow / README 类改动

这样能明显缩小冲突面。

### 8.3 如果你坚持先做一次 merge 试验

那也建议这样做：

1. 先把本地 `dev` 同步到 `origin/dev`
2. 建一个临时分支，例如 `integration/main-into-dev-20260529`
3. 在这个临时分支上 merge `origin/main`
4. 冲突处理时遵循下面的默认规则：

- 默认保 `ours`：
  - 支付、订阅、信用池、site messages、lottery、invoice、Mapay、public pages
- 默认偏向 `theirs`：
  - gateway 协议修复、scheduler、OpenAI WS/Responses 兼容、usage 口径、auth 数据保全
- 必须人工重构，不能简单选边：
  - migrations
  - `backend/ent/*`
  - `backend/internal/handler/dto/settings.go`
  - `frontend/src/types/payment.ts`
  - `frontend/src/views/admin/SettingsView.vue`
  - `frontend/src/views/user/UsageView.vue`

但我仍然不建议把这条路径作为主路线。

## 9. 我认为最合理的下一步

如果你同意我的优先级，下一步应该不是“直接 merge”，而是：

1. 先同步本地 `dev` 到 `origin/dev`
2. 做一份“第一轮吸收清单”，只纳入 bugfix / perf / correctness
3. 我按这份清单逐组回收 upstream 变更
4. 每组改完后分别做 migration、Ent、后端编译/测试、前端验证

这样做的好处是：

- 风险可控
- 每组都能独立回滚
- 不会把支付、订阅、风控、配额一次性搅在一起

## 10. 已确认决策

你已经确认或授权我决定如下：

1. 不收上游“用户 x 平台配额”整套能力
2. 不收上游支付新能力，尤其多币种 / Airwallex
3. 接受为 fork 预留单独 migration 命名空间，采用 `900_*.sql`

第 3 点的实际含义：

- 以后 fork 自己新增的数据库结构变更，不再和 upstream 共用 `14x` 这类编号区间
- upstream 继续走它自己的迁移序列
- fork 侧新增迁移统一走 `900_*.sql`、`901_*.sql` 这类编号
- 这样能避免再次出现 `136`~`143` 这种“同编号、不同语义”的正面冲突

基于这个决策，第一轮上游吸收范围可以进一步收紧为：

- 网关协议兼容修复
- usage / billing 正确性修复
- scheduler / cooldown / failover 稳定性修复
- OAuth / auth 数据保全修复

明确排除：

- 平台配额相关 schema / UI / 预检链路
- Airwallex / 多币种支付链路
- 与上述两类能力强绑定的前端页面、DTO 和测试

## 11. 线上稳定优先的同步原则

你的要求可以压缩成两条：

1. 当前线上不能因为同步而出问题
2. 以后重复同步 upstream，也不能越来越难、越来越危险

基于这个要求，后续策略固定为：

### 11.1 不做整分支 merge，只做主题级吸收

默认禁止：

- 直接把 `origin/main` 整体 merge 进 `dev`
- 为了“省事”一次性收口支付、配额、风控、网关、迁移

默认允许：

- 按主题逐组回收 upstream 的 bugfix / correctness / perf 改动
- 对需要 schema 变化的上游修复，转写为 fork 自己的迁移

原因：

- 整分支 merge 的冲突面太大，真实冲突文件已经有 `95` 个
- 真正会炸线上的一般不是文本冲突，而是 migration、计费口径和协议行为错位

### 11.2 以后所有 fork 自有 migration 统一走 `900_*.sql`

这条作为硬规则保留。

作用：

- 把 fork 和 upstream 的数据库演进轨道拆开
- 避免后续再次与 upstream 的 `14x`、`15x` 编号冲突
- 让未来同步时可以一眼分清“这是 upstream 迁移”还是“这是 fork 迁移”

### 11.3 每次同步都走同一个筛选流程

每次从 upstream 同步时，把变更分成三类：

#### A 类：直接值得吸收

满足任一条件即可优先考虑：

- 修协议兼容
- 修计费正确性
- 修调度稳定性
- 修 OAuth / auth 数据保全
- 修明显性能问题

这类改动通常优先进入同步候选清单。

#### B 类：有价值，但必须改写后再吸收

典型特征：

- 需要新 migration
- 需要改 Ent/schema
- 会碰现有 `dev` 的支付、订阅、风控、自定义产品能力

这类改动不能直接搬，必须按 fork 现状重写。

#### C 类：明确不吸收

当前已经确定排除：

- 平台配额体系
- Airwallex / 多币种支付
- 与这两类能力强绑定的页面、DTO、测试和 Webhook 逻辑

### 11.4 每次同步必须做到“小批次、可回滚、可验证”

后续同步不做“大包更新”，而是按下面的粒度推进：

1. 一次只处理一个主题
2. 一个主题里同时包含：
   - 代码改动
   - 必要 migration
   - 对应测试
   - 验证记录
3. 主题验证通过后再进入下一组

这样即使出问题，也能迅速定位到是哪一组引入的。

### 11.5 进入发布前的最低验证标准

任何准备进入 `publish` 或线上部署的同步改动，至少满足：

- 后端编译通过
- 前端构建通过
- 受影响测试通过
- migration 在本地可执行
- 受影响核心链路做 smoke test

对这类同步，重点 smoke test 的链路固定为：

- 登录 / 注册 / OAuth 回调
- OpenAI / Responses / WS 网关请求
- usage / billing 展示与统计
- 账号调度 / failover / cooldown
- 现有支付与订阅流程不得回归

### 11.6 对“不会出问题”的工程化解释

没人能诚实承诺“绝对零风险”，但可以把风险控制在可管理范围内。

这里的实现方式不是追求一次性合并成功，而是：

- 不引入大范围未知变更
- 不破坏现有 migration 轨道
- 不让上游新功能挤进现有支付/订阅核心链路
- 每次同步都可拆、可查、可回滚

这正是后续同步仍然能持续做下去、而不是越同步越危险的关键。

## 12. 第一批可安全吸收的 upstream 提交清单

这一批的筛选标准很保守：

- 不引入你已明确拒绝的能力（平台配额、Airwallex、多币种支付）
- 不改 migration / Ent / 支付主链路
- 尽量限定在局部 service / repository / apicompat / test
- 单个提交的文件面越小越优先

这批不是“最重要的全部改动”，而是“最适合先落地、最不容易伤到线上”的那部分。

### 12.1 建议直接进入第一批

#### A. 账户状态与数据保全

1. `6aec5050` `fix(oauth): don't overwrite credentials JSONB in 401 handler`

- 价值：防止 refresh token 被旧快照写回数据库，导致账号被误禁用
- 范围：`backend/internal/service/ratelimit_service.go`
- 风险：低
- 结论：第一批必收

2. `202aab8e` `fix(accounts): unschedule errored accounts`

- 价值：账号进入错误状态后及时从调度中移除
- 范围：`backend/internal/repository/account_repo.go` + integration test
- 风险：低
- 结论：第一批必收

3. `60f6602b` `fix: clear scheduler cache when deleting accounts`

- 价值：删除账号后清掉 scheduler 缓存，避免残留脏状态
- 范围：`backend/internal/repository/account_repo.go` + integration test
- 风险：低
- 结论：第一批必收

#### B. 协议兼容与 usage 正确性

4. `0a521f09` `fix(gemini): close tool_use block before text in messages streaming`

- 价值：修 Gemini -> Anthropic SSE block 生命周期错误，避免客户端误解析
- 范围：`gemini_messages_compat_service` + regression test
- 风险：低
- 结论：第一批必收

5. `20f53407` `fix(apicompat): Responses→Chat completion_tokens_details 透传`

- 价值：补齐 reasoning/audio/prediction token 细项，方便 usage 对账
- 范围：`backend/internal/pkg/apicompat/*`
- 风险：低
- 结论：第一批必收

6. `89dffdd2` `fix(apicompat): emit OpenAI-semantic input_tokens when converting Anthropic to Responses`

- 价值：修 cached tokens 被漏算的问题，避免 prompt/input token 偏低
- 范围：`backend/internal/pkg/apicompat/*`
- 风险：低
- 结论：第一批必收

7. `32ea9cfe` `fix: fallback to SSE body for API key responses`

- 价值：修 API Key Responses 流式 body 回退
- 范围：`backend/internal/service/openai_gateway_service.go` + test
- 风险：低到中
- 结论：第一批可收

8. `8211aa70` `fix: retry on "thinking block must contain thinking" upstream error`

- 价值：对 Claude thinking block 的上游签名错误自动走现有重试逻辑
- 范围：`gateway_service.go` + test
- 风险：低
- 结论：第一批可收

#### C. 计费与统计正确性

9. `b9509e82` `fix(billing): apply long-context multiplier to cache_read price`

- 价值：修长上下文场景下 `cache_read` 少计费
- 范围：`billing_service.go` + tests
- 风险：低
- 结论：第一批必收

10. `ed2aac25` `fix(billing): apply long-context multiplier to cache_creation price`

- 价值：修长上下文场景下 `cache_creation` 少计费
- 范围：`billing_service.go` + tests
- 风险：低
- 结论：第一批必收

11. `1e6d0b60` `fix(antigravity): capture message_start input_tokens in streaming passthrough`

- 价值：修流式消息的 input-side usage 丢失
- 范围：`antigravity_gateway_service.go` + tests
- 风险：低
- 结论：第一批可收

12. `8a999f43` `fix(ws): exclude terminal events from first-token detection`

- 价值：修 observability 指标，避免把总耗时误记成首字延迟
- 范围：`openai_ws_forwarder.go` + tests
- 风险：很低
- 结论：第一批可收

#### D. 条件吸收

13. `a9c7a3a0` `fix(bedrock): strip context_management when beta is removed`

- 价值：只影响 Bedrock 兼容
- 前提：你线上确实在跑 Bedrock Claude Code 相关链路
- 风险：低
- 结论：若线上使用 Bedrock，则第一批可收；否则先跳过

14. `0af44ce4` `fix: 修复反代部署下拒绝日志客户端 IP 不准确`

- 价值：修拒绝诊断日志中的客户端 IP 记录
- 前提：更偏可观测性，不影响主业务逻辑
- 风险：很低
- 结论：可顺手带上，但优先级低于前 12 个

### 12.2 虽然有价值，但不进入第一批

这些改动不是不要，而是现在先不收，原因是 blast radius 明显更大。

1. `11fe7de9` `fix(account): 重新授权不再清空 Extra 配置`

- 价值高
- 但它新增管理端接口、改后端路由、改前端重新授权流程
- 结论：放第二批

2. `56e96fdd` `fix: classify concurrency acquire failures`

- 范围不算大
- 但会碰 gateway handler 的错误响应语义
- 结论：放第二批

3. `f7ac5e59` `fix(openai): preserve chat responses usage billing`

- 价值高
- 但同时碰 `openai_gateway_chat_completions` / `messages` / `apicompat`
- 结论：放第二批

4. `1e406fed` `fix: optimize OpenAI account cooldown scheduling`

- 价值很高
- 但改了 29 个文件，是明显的系统级调度改造
- 结论：不进入第一批

5. `08061717` `fix: enable account failover for OpenAI WS rate limits`

- 价值高
- 但改动 7 个文件、600+ 行，属于中高风险网关行为变更
- 结论：不进入第一批

6. `a31b5074` `fix(scheduler): 模型404仅冷却账号模型组合`

- 价值高
- 但横跨 gateway/service/repository/test 多层
- 结论：不进入第一批

7. `33ac8eb2` `fix openai http2 response header timeout`

- 价值高
- 但涉及 `http_upstream`、配置、部署样例和底层传输行为
- 结论：不进入第一批

8. `fc66cd70` `fix: recognize codex tool outputs in ws continuation`

- 价值明确
- 但逻辑面较深，先不和第一批混做
- 结论：不进入第一批

9. `6381f9e3` `fix(openai): 识别上游静默拒绝并触发 failover`

- 价值明确
- 但会改变 failover 判定语义
- 结论：不进入第一批

### 12.3 建议的落地顺序

按线上风险从低到高，第一批建议分四小组推进：

1. 账户保全组
   - `6aec5050`
   - `202aab8e`
   - `60f6602b`

2. 协议兼容组
   - `0a521f09`
   - `20f53407`
   - `89dffdd2`
   - `32ea9cfe`
   - `8211aa70`

3. 计费正确性组
   - `b9509e82`
   - `ed2aac25`
   - `1e6d0b60`

4. 可观测性 / 条件修复组
   - `8a999f43`
   - `a9c7a3a0`（仅 Bedrock 在用时）
   - `0af44ce4`

### 12.4 第一批为什么这样切

核心原则只有一句：

- 先修“明显会让线上账不对、usage 不对、账号状态错、协议输出错”的问题
- 暂时不碰“虽然收益高，但会重写调度/路由/前端流程”的问题

这能最大化降低第一轮同步把线上打坏的概率。

## 13. 第一批落地执行方案

这一节把上面的判断收敛成可以执行的工程步骤。

### 13.1 当前本地工作区状态

对当前本地工作区的观察：

- 本地 `dev` 仍然落后 `origin/dev` 6 个提交
- 工作区已经有一批未提交后端改动
- 这些改动看起来已经吸收了第一组和第二组的大部分内容
- 计费正确性组、Antigravity、WS 指标、Bedrock/IP 日志修复尚未看到对应文件改动
- `doc/branching.md` 也有未提交改动，提交前需要确认是否属于同一批同步说明

当前已在工作区出现的第一批候选：

- `6aec5050` OAuth 401 不回写旧 credentials
- `202aab8e` errored accounts 不再可调度
- `60f6602b` 删除账号时清理 scheduler cache
- `0a521f09` Gemini Messages 流式 block 闭合
- `20f53407` Responses -> Chat token details
- `89dffdd2` Anthropic -> Responses input token 语义修正
- `32ea9cfe` API Key Responses SSE body fallback
- `8211aa70` thinking block missing content 错误进入重试

当前仍待处理的第一批候选：

- `b9509e82` `cache_read` 长上下文倍率
- `ed2aac25` `cache_creation` 长上下文倍率
- `1e6d0b60` Antigravity streaming input usage
- `8a999f43` WS terminal events 不计入 first-token
- `a9c7a3a0` Bedrock context management 兼容（仅 Bedrock 在用时）
- `0af44ce4` 反代下拒绝日志客户端 IP（可选）

所以后续不是“从零执行第一批”，而是：

1. 先保护当前未提交 diff
2. 验证已经吸收的两组
3. 再决定是否继续吸收第三组和第四组

### 13.2 执行前置条件

继续动代码前，先满足这些条件：

1. 当前未提交 diff 必须先保护起来
2. 不要在 dirty `dev` 上继续叠 cherry-pick
3. 不在第一批里引入 migration / Ent 改动
4. 每个小组独立提交，不能把所有上游修复揉成一个大提交
5. 与 `origin/dev` 的 6 个提交要在当前吸收工作被保护后再处理

如果当前改动确认就是第一批同步工作，推荐补救顺序：

```bash
git fetch origin upstream
git checkout dev
git checkout -b sync/upstream-core-fixes-20260529
git status --short
```

然后先跑已吸收组的测试。测试通过后，按组提交当前改动，再处理 `origin/dev` 的 6 个提交。

如果当前未提交改动里混有其它工作，不要直接提交。先把非本次同步的改动拆出去，或者 stash 到单独记录后再继续。

### 13.3 不建议无脑 `cherry-pick` 的原因

第一批候选提交都比较小，但仍然不建议机械执行：

```bash
git cherry-pick 6aec5050 202aab8e 60f6602b ...
```

原因：

- `dev` 已经改过订阅、余额、网关、调度等关键路径
- 有些上游测试需要按 fork 现状调整 fixture
- 有些修复所在文件已经有本地未提交改动
- cherry-pick 成功不等于行为已经适配 fork

更稳妥的方式是按小组执行：

1. 先 `git show --stat <commit>` 看范围
2. 再 `git show <commit> -- <files>` 看具体 diff
3. 可以 cherry-pick，但每个冲突都按 fork 现状人工审查
4. 每组完成后立即跑对应测试
5. 只在测试通过后提交该组

### 13.4 第一组：账户保全

目标：

- 修 OAuth 401 覆盖 credentials 的问题
- 修账号错误状态和删除后的 scheduler 脏缓存问题

提交：

- `6aec5050`
- `202aab8e`
- `60f6602b`

重点文件：

- `backend/internal/service/ratelimit_service.go`
- `backend/internal/repository/account_repo.go`
- `backend/internal/repository/account_repo_integration_test.go`

验证建议：

```bash
cd backend
go test ./internal/service -run 'Test.*RateLimit|Test.*OAuth|Test.*Account' -count=1
go test ./internal/repository -run 'TestAccount' -count=1
go test ./cmd/server -count=1
```

验收标准：

- OAuth 401 不会把旧 credentials 快照写回数据库
- errored account 不再留在调度候选里
- 删除 account 后 scheduler cache 被清理

当前状态：

- 已出现在本地未提交 diff 中
- 尚需跑测试确认

### 13.5 第二组：协议兼容

目标：

- 修 Gemini Messages 流式 block 闭合
- 修 Responses / Chat / Anthropic usage 互转
- 修 API Key Responses SSE body 回退
- 修 thinking block 上游错误重试

提交：

- `0a521f09`
- `20f53407`
- `89dffdd2`
- `32ea9cfe`
- `8211aa70`

重点文件：

- `backend/internal/service/gemini_messages_compat_service.go`
- `backend/internal/pkg/apicompat/*`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/gateway_service.go`

验证建议：

```bash
cd backend
go test ./internal/pkg/apicompat -count=1
go test ./internal/service -run 'TestGeminiMessages|TestOpenAI.*Responses|TestGateway.*Thinking|Test.*SSE' -count=1
go test ./internal/handler -run 'Test.*Gateway|Test.*Responses' -count=1
```

验收标准：

- Gemini 流式输出中 `tool_use` / `text` block 顺序合法
- Responses 转 Chat 时保留 token details
- Anthropic 转 Responses 时 input token 语义符合 OpenAI usage
- API Key Responses 流式场景有 SSE body fallback
- thinking block 特定上游错误进入重试逻辑

当前状态：

- 已出现在本地未提交 diff 中
- 尚需跑测试确认

### 13.6 第三组：计费正确性

目标：

- 修长上下文 `cache_read` / `cache_creation` 倍率
- 修 Antigravity streaming input usage 丢失

提交：

- `b9509e82`
- `ed2aac25`
- `1e6d0b60`

重点文件：

- `backend/internal/service/billing_service.go`
- `backend/internal/service/antigravity_gateway_service.go`

验证建议：

```bash
cd backend
go test ./internal/service -run 'Test.*Billing|Test.*Antigravity|Test.*Usage' -count=1
go test ./internal/repository -run 'TestUsage' -count=1
```

验收标准：

- 长上下文 cache read / cache creation 价格倍率正确
- Antigravity streaming 的 input-side usage 不再丢失
- 现有订阅信用池 / balance 计费测试不回归

当前状态：

- 尚未出现在本地未提交 diff 中
- 建议在前两组验证并提交后，再单独处理

### 13.7 第四组：可观测性与条件修复

目标：

- 修 WS 首字延迟指标
- 可选修 Bedrock context management 兼容
- 可选修反代下拒绝日志 IP

提交：

- `8a999f43`
- `a9c7a3a0`
- `0af44ce4`

其中：

- `a9c7a3a0` 只有线上确实使用 Bedrock Claude Code 链路时才收
- `0af44ce4` 可顺手带上，但不应该为了它扩大冲突面

验证建议：

```bash
cd backend
go test ./internal/service -run 'Test.*WS|Test.*FirstToken|Test.*Bedrock|Test.*ClientIP' -count=1
```

验收标准：

- WS terminal events 不再污染首字指标
- Bedrock 场景按实际启用情况决定是否纳入
- 反代下日志 IP 修复不改变请求鉴权和限流行为

当前状态：

- 尚未出现在本地未提交 diff 中
- `a9c7a3a0` 和 `0af44ce4` 都可以继续保持可选，不必为了“第一批完整”强行纳入

## 14. 合并过程中的取舍规则

遇到冲突时，按下面规则处理。

### 14.1 默认保留 fork 现状

这些区域默认保留 `dev`：

- 订阅信用池
- balance 混合支付
- Mapay 支付
- invoice request / export
- site messages
- support chat
- lottery
- `frontend-dashboard/`
- 当前 fork 已经定制过的支付、订阅、余额 UI

原因：

- 这些是 fork 的产品差异，不是 upstream 的落后代码
- 上游的支付/配额变更与这些能力重叠很深

### 14.2 默认优先吸收 upstream 修复

这些区域在确认不引入新产品能力时，默认偏向吸收 upstream：

- apicompat token / usage 转换
- gateway 协议兼容
- OAuth credentials / Extra 数据保全
- scheduler cache 清理
- billing multiplier 修复
- WS 指标修复

### 14.3 必须人工重构

这些文件或主题不能简单选 `ours` / `theirs`：

- migration 文件
- `backend/ent/*`
- `backend/internal/handler/dto/settings.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/payment_*`
- `frontend/src/types/payment.ts`
- `frontend/src/views/user/PaymentView.vue`
- `frontend/src/views/user/SubscriptionsView.vue`
- `frontend/src/views/admin/SettingsView.vue`

第一批原则上应避开这些高风险合并面；如果某个小修必须碰这些文件，就把它单独拆出来，不和其它提交混做。

## 15. 发布前验证矩阵

第一批全部完成后，不要只跑单点测试。最低验证矩阵如下。

### 15.1 后端

```bash
cd backend
go test ./internal/pkg/apicompat -count=1
go test ./internal/service -count=1
go test ./internal/repository -run 'TestAccount|TestUsage|TestBilling' -count=1
go test ./internal/handler -run 'Test.*Gateway|Test.*Responses' -count=1
go test ./cmd/server -count=1
make build
```

如果本轮改到 Ent / migration，虽然第一批不应该发生，还必须额外跑：

```bash
cd backend
go generate ./ent
go test ./internal/repository -run 'TestMigrations' -count=1
```

### 15.2 上游原 Vue 前端

第一批理论上不碰 `frontend/`。如果实际执行中没有修改它，可以不跑完整前端构建，只记录：

- `frontend/` 无 diff

如果有任何 `frontend/` diff，至少跑：

```bash
cd frontend
pnpm install
pnpm typecheck
pnpm build
```

### 15.3 LumioAPI 前端

第一批理论上不碰 `frontend-dashboard/`。如果没有 diff，只记录：

- `frontend-dashboard/` 无 diff

如果有任何影响 API DTO、用户 usage、订阅或支付展示的后端改动，建议仍跑：

```bash
cd frontend-dashboard
pnpm typecheck
pnpm build
```

并启动 dev server 做一次 smoke：

```bash
cd frontend-dashboard
pnpm dev
```

重点看：

- 登录态页面能打开
- 订阅页不报错
- usage / billing 展示不报错
- 支付入口不回归

## 16. 回滚边界

第一批必须按组提交，原因是回滚边界清晰。

推荐提交结构：

1. `fix: absorb upstream account state safeguards`
2. `fix: absorb upstream gateway protocol compatibility fixes`
3. `fix: absorb upstream usage billing correctness fixes`
4. `fix: absorb upstream observability compatibility fixes`

如果发布后发现问题：

- 账号状态问题，只回滚第 1 组
- 协议输出问题，只回滚第 2 组
- 计费问题，只回滚第 3 组
- 指标或 Bedrock 问题，只回滚第 4 组

不要把四组压成一个提交，否则线上排障时只能整体回滚，风险反而变大。

## 17. 最终建议

最终建议仍然是“不做整分支 merge”，但结合当前本地状态，实际下一步应改成：

1. 不 merge `origin/main` 到 `dev`
2. 先从当前 dirty `dev` 切出同步分支，保护已经吸收的第一组/第二组改动
3. 跑第一组和第二组的测试，确认当前未提交 diff 不是半截状态
4. 测试通过后，按账户保全、协议兼容两个主题提交当前改动
5. 提交后再把同步分支对齐 `origin/dev` 的 6 个提交
6. 再决定是否继续吸收第三组计费正确性和第四组可观测性修复
7. 坚持不收平台配额和 Airwallex / 多币种支付
8. 任何 fork 自有 schema 变化统一使用 `900_*.sql` 命名空间

这条路径不能保证“绝对不会出问题”，但它把风险限制在小范围、可测试、可回滚的变更组里。

从线上稳定和后续持续同步的角度看，这是当前最合理的主路线。

## 18. 实际合并记录（2026-05-30）

本节记录本次 selective sync 的实际状态。这里的“已合并”指对应同步主题分支已吸收并验证，随分支合入 `dev` 后进入 `origin/dev`。

### 18.1 已合并到 `dev` 的同步批次

| 主题 | upstream 提交 | 本地提交 | 状态 | 风险 |
|------|---------------|----------|------|------|
| 账户状态与 OAuth 数据保全 | `6aec5050`, `202aab8e`, `60f6602b` | `a8ba443b` | 已合并 | 低 |
| Gateway / Responses / Anthropic / Gemini 协议兼容 | `0a521f09`, `20f53407`, `89dffdd2`, `32ea9cfe`, `8211aa70` | `82e93797` | 已合并 | 低到中 |
| usage / billing 正确性 | `b9509e82`, `ed2aac25`, `1e6d0b60` | `45fc61d0` | 已合并 | 低 |
| WS 指标、Bedrock 兼容、反代 IP 日志 | `8a999f43`, `a9c7a3a0`, `0af44ce4` | `2820fc76`, `a6781dad` | 已合并 | 低到中 |
| 并发获取失败分类 | `56e96fdd` | `a89ad299` | 已合并 | 中 |
| 重新授权保留 `Extra` | `11fe7de9` | `3049896e` | 已合并 | 中 |
| chat responses usage billing / request context 保留 | `2bd3125d`, `f7ac5e59` | `613842c8`, `0fd7c29d` | 已合并 | 中到高 |
| OpenAI / Codex tool continuation 兼容 | `16a31557`, `87d73236`, `348a4877`, `fc66cd70`, `a729752d` | `84139594`, `245b08c3`, `d30b35e4`, `c15fdadc`, `bd7c0b38` | 已合并 | 中 |
| admin 账号 credentials 脱敏 | `0f8e2d09` | `69c26b94` | 已合并 | 中 |
| service correctness 小修复 | `2a17c0b2`, `f788e6bd`, `6d69ae87`, `f97b8534`, `c3a14717`, `297b54d0` | `9396e620`, `e7368e43`, `d6f92c77`, `2e08ca1a`, `ebd16099`, `f787ed02` | 已合并 | 中 |
| OpenAI / Responses 兼容小修复 | `679c0865`, `df82a3bc`, `1d47fd63`, `e9a25e7b`, `a6117429`, `be15a3e6` | `0b2a6d37`, `e48af74a`, `37e04bb6`, `4c920673`, `0400f5f4`, `1959c6d1` | 已合并 | 中 |
| 上游 Vue 前端 auth / 编辑兼容小修复 | `44679221`, `224e9fc6`, `4d51e53d`, `3ca232ad` | `2a4d1fad`, `21d84467`, `0bdb6784`, `f842049c` | 已合并 | 中 |
| Gateway models / stream keepalive / OpenAI usage parsing | `2ec1d331`, `164e2f61`, `0393bd7c` | `4e689527`, `2fd34f1b`, `09af7387` | 已合并 | 中 |
| setup 初始化完成后的路由保护 | `a9a357e9` | `b756ac63` | 已合并 | 低到中 |
| apicompat 参数清洗 / count_tokens 过滤 | `fe3283a1`, `276b5c77`, `27600b1d` | `dbffb71c`, `da287da3`, `a2cc7dfe` | 已合并 | 中 |
| frontend / deploy / dependency 小修复 | `18790386`, `a5acefcc`, `44995404`, `e46d2c21`, `ffd53343` | `4dcbeba8`, `e9e674f7`, `50c4647e`, `d26594ee`, `45590f24` | 已合并 | 低到中 |
| subscription repo 测试适配 | 本地 API 适配 | `074740e1` | 已合并 | 低 |
| 本评估文档与分支说明 | 本地文档 | `715c75b2`, `098bc053` 起持续更新 | 已合并 | 低 |

说明：

- 没有整分支 merge `origin/main`。
- 没有引入 upstream 平台配额。
- 没有引入 Airwallex / 多币种支付。
- 没有新增或改写 migration / Ent schema。
- Bedrock 冲突只吸收 `context_management` 按最终 beta tokens 清理这一项，没有顺手带入 upstream 侧其它 Bedrock CC 兼容扩展。
- `56e96fdd` 只吸收 handler 层并发 acquire 错误分类：真实并发限制仍返回 429，客户端取消返回 499，Redis / deadline 等 acquire 失败返回 503；未引入调度系统重构。
- `11fe7de9` 新增专用重新授权落库接口，只做 credentials 更新与 `Extra` key 级合并，并让前端重新授权流程改走该接口；未引入 migration / Ent / 支付链路变更。
- `2bd3125d` / `f7ac5e59` 只吸收 usage request context 透传与 Responses 顶层 `usage` 计费修复；冲突处理中未引入 upstream 平台配额字段、未恢复 `openai_embeddings.go`、未带入额外 WS failover 逻辑和不属于本主题的 raw chat 大块测试扩展。
- `16a31557` / `87d73236` / `348a4877` / `fc66cd70` / `a729752d` 只吸收 OpenAI/Codex continuation、tool output 识别与 `call_*` ID 保留相关修复；未引入 WS rate-limit failover、调度冷却、平台配额或 schema 变更。`a729752d` 是 `348a4877` 对应测试断言修正，行为变更来自 `348a4877`。
- `0f8e2d09` 只吸收 admin 账号响应 credentials 脱敏和全对象编辑时敏感 credentials 保留语义：新增 `credentials_status` 暴露存在性，前端留空敏感字段时由后端保留旧 token；未引入 schema / migration / 支付 / 平台配额变更。
- `2a17c0b2` / `f788e6bd` / `6d69ae87` / `f97b8534` / `c3a14717` / `297b54d0` 作为 service correctness 小批次合入：包含 Vertex token exchange 走账号代理、未知默认 transport 类型保护、未定价模型零成本 usage 记录、mimic tool_use 名称同步改写、OpenAI usage-limit plan type 同步，以及相关测试补强。`f788e6bd` 对 `account_codex_import.go` 的改动被明确排除，因为该文件属于未合入的 OAuth 导入功能；本批只吸收 `vertex_service_account.go` 的 transport 检查。
- `679c0865` / `df82a3bc` / `1d47fd63` / `e9a25e7b` / `a6117429` / `be15a3e6` 只吸收 OpenAI/Responses 小范围兼容修复：versioned compatible base URL、chat completions 转 Responses 时避免 `null` content、DeepSeek `reasoning_content` 透传、空 thinking block 保留、图片生成 upstream context detach、WS passthrough 首字时间修正。未带入 `cc5328c4` 这类更大 SSE 终止事件语义改造。
- `44679221` / `224e9fc6` / `4d51e53d` / `3ca232ad` 只吸收上游 Vue 前端小兼容修复：TOTP 自动填充、OIDC pending flow 使用 compat email、兑换码批量复制兼容、编辑账号弹窗在旧后端未返回 `credentials_status` 时回退旧 credentials 结构。`3ca232ad` 的测试冲突中保留了本地 upstream-balance 覆盖测试。`65493df9` 没有机械合入，因为本地已有 `frontend/src/utils/ccswitch.ts`，并且已覆盖 OpenAI Codex 默认模型、defaultModels、`/v1` suffix 和 usageBaseUrl，比 upstream 新增的 `ccswitchImport.ts` 更完整。
- `2ec1d331` / `164e2f61` / `0393bd7c` 只吸收三项网关兼容小修复：Gemini 分组 `/v1/models` 按平台过滤并回退 Gemini 默认模型、Anthropic API key passthrough 流式空闲 keepalive、OpenAI-compatible usage 兼容 `prompt_tokens` / `completion_tokens` 形状。未引入 upstream 自定义 `/v1/models` schema 功能、WS rate-limit failover 或调度冷却重构。
- `a9a357e9` 只吸收 setup 初始化完成后阻止继续访问 `/setup` 的路由保护。冲突点在 `frontend/src/router/index.ts` 与 `frontend/src/router/__tests__/guards.spec.ts` 的 import / mock state 并列新增，解决时保留本地 external auth handoff、feature flags、backend mode 路由限制，并加入上游 `getSetupStatus()` 检查。
- `fe3283a1` / `276b5c77` / `27600b1d` 只吸收 apicompat 与 Anthropic passthrough 小范围请求清洗：reasoning content 转换 errcheck、Responses 转换时对 reasoning 模型剥离 `temperature` / `top_p`、`count_tokens` 透传前过滤生成字段。`276b5c77` 的测试冲突只发生在 `anthropic_responses_test.go`，解决时保留本地 cache token 语义测试并加入 upstream 参数剥离测试。`c4d7edba` 已尝试但跳过，因为它修改的 `chatcompletions_responses_bridge.go` 在当前 `dev` 不存在，实际依赖未合入的 Responses/Chat fallback bridge 大功能。
- `18790386` / `a5acefcc` / `44995404` / `e46d2c21` / `ffd53343` 只吸收低耦合维护修复：Docker Compose 不再暴露 PostgreSQL/Redis 到宿主机、安装脚本提前检查 Bash 4+、Docker frontend builder 固定 pnpm v9、Ops deep link 状态初始化顺序修正、`js-cookie` 通过 pnpm override 升级到 `3.0.7`。`44995404` 的 Dockerfile 冲突中，本地原为 `pnpm@10.30.3`，按 upstream 与仓库 pnpm v9 构建约束改为 `pnpm@9`；`ffd53343` 的 `package.json` 冲突中保留本地 `onlyBuiltDependencies` 并新增 `overrides.js-cookie`。

### 18.2 本次已验证命令

本次同步分支合入 `origin/dev` 前已跑：

```bash
cd backend
go test ./internal/pkg/apicompat -count=1
go test ./internal/service -count=1
go test ./cmd/server -count=1
go test ./internal/repository -count=1
go test ./internal/handler ./internal/handler/admin -count=1
go test ./internal/server ./internal/server/middleware -count=1
make build

cd ../frontend
pnpm typecheck
pnpm build
```

结果：

- 后端测试通过
- 后端 build 通过
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仅有既有 Vite chunk / dynamic import warning
- `frontend-dashboard/` 本次无改动，未运行构建

第二批并发错误分类同步追加验证：

```bash
cd backend
go test ./internal/handler -run 'TestConcurrencyErrorResponse|TestWaitForSlotWithPingTimeout_ParentContextCanceled|TestWaitForSlotWithPingTimeout_AcquireError' -count=1
go test ./internal/handler -count=1
```

结果：

- `backend/internal/handler` 针对性并发错误分类测试通过
- `backend/internal/handler` 全包测试通过

第二批重新授权保留 `Extra` 同步追加验证：

```bash
cd backend
go test ./internal/handler/admin ./internal/service ./internal/server -count=1

cd ../frontend
pnpm typecheck
pnpm build
```

结果：

- `backend/internal/handler/admin`、`backend/internal/service`、`backend/internal/server` 测试通过
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仍仅有既有 Vite dynamic import / chunk size warning

第二批 chat responses usage billing / request context 同步追加验证：

```bash
cd backend
go test ./internal/handler ./internal/server/middleware -run 'TestSubmitUsageRecordTaskCopiesRequestContext|TestOpenAISubmitUsageRecordTaskCopiesRequestContext|TestGatewayHandlerSubmitUsageRecordTask|TestOpenAIGatewayHandlerSubmit|TestClientRequestID' -count=1
go test ./internal/handler ./internal/server/middleware -count=1
go test ./internal/pkg/apicompat -run 'TestResponsesEventToAnthropicEvents_TopLevelTerminalUsage|TestResponsesEventToChatChunks_TopLevelTerminalUsage' -count=1
go test ./internal/service -run 'TestForwardAsChatCompletions_StreamsUsageWithoutClientStreamOptions|TestForwardAsChatCompletions_StreamsTopLevelTerminalUsage|TestForwardAsChatCompletions_BufferedTopLevelTerminalUsage' -count=1
go test ./internal/pkg/apicompat ./internal/service -count=1
```

结果：

- `backend/internal/handler`、`backend/internal/server/middleware` usage context 相关测试通过
- `backend/internal/pkg/apicompat` 顶层 terminal usage 转换测试通过
- `backend/internal/service` chat responses usage billing 回归测试通过
- `backend/internal/pkg/apicompat`、`backend/internal/service` 全包测试通过

第三批 OpenAI / Codex tool continuation 同步追加验证：

```bash
cd backend
go test ./internal/service -run 'Test(NeedsToolContinuationSignals|HasFunctionCallOutput|HasToolCallContext|FunctionCallOutputCallIDs|HasFunctionCallOutputMissingCallID|HasItemReferenceForCallIDs|ForwardAsAnthropic_PreviousResponseIDKeepsMultiToolCallContext|OpenAIGatewayService_ProxyResponsesWebSocketFromClient_StoreDisabled(FunctionCallOutput|ToolSearchOutput|.*FunctionCallOutput)|ApplyCodexOAuthTransform_(ToolContinuation|ToolSearchOutput|CustomAndMCPToolOutputs))' -count=1
go test ./internal/service -count=1
```

结果：

- 第一次聚焦测试发现 `348a4877` 需要同步上游测试断言修正 `a729752d`，补入后聚焦测试通过
- `backend/internal/service` 全包测试通过

第四批 admin credentials redaction 同步追加验证：

```bash
cd backend
go test ./internal/handler/dto ./internal/service -count=1

cd ../frontend
pnpm typecheck
pnpm build
```

结果：

- `backend/internal/handler/dto`、`backend/internal/service` 测试通过
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仍仅有既有 Vite dynamic import / chunk size warning

第五批 service correctness 小修复同步追加验证：

```bash
cd backend
go test ./internal/service ./internal/pkg/tlsfingerprint -count=1
```

结果：

- `backend/internal/service` 全包测试通过
- `backend/internal/pkg/tlsfingerprint` 测试通过；外部 capture 测试在未设置 `TLSFINGERPRINT_CAPTURE_URL` 时 skip

第六批 OpenAI / Responses 兼容小修复同步追加验证：

```bash
cd backend
go test ./internal/pkg/apicompat ./internal/service ./internal/service/openai_ws_v2 -count=1
```

结果：

- `backend/internal/pkg/apicompat` 测试通过
- `backend/internal/service` 全包测试通过
- `backend/internal/service/openai_ws_v2` 测试通过

第七批上游 Vue 前端小兼容修复同步追加验证：

```bash
cd frontend
pnpm test:run -- src/components/account/__tests__/EditAccountModal.spec.ts
pnpm typecheck
pnpm build
```

结果：

- Vitest 实际跑完全量前端测试：125 个 test files / 685 个 tests 全部通过
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仍仅有既有 Vite dynamic import / chunk size warning

第八批 Gateway / setup 兼容小修复同步追加验证：

```bash
cd backend
go test ./internal/handler ./internal/service -run 'TestGatewayModels|TestGatewayService_AnthropicAPIKeyPassthrough_Streaming(SendsKeepaliveDuringIdle|KeepaliveDoesNotInterleavePartialEvent)|TestParseSSEUsage_SelectiveParsing|TestExtractOpenAIUsageFromJSONBytes_AcceptsResponseAndChatUsageShapes|TestParseOpenAIWSResponseUsageFromCompletedEvent|TestForwardAsAnthropic_MappedClaudeModelAcceptsChatUsageShape' -count=1
go test ./internal/handler ./internal/service ./internal/pkg/apicompat -count=1

cd ../frontend
pnpm test:run -- src/router/__tests__/guards.spec.ts
pnpm typecheck
pnpm build
```

结果：

- `backend/internal/handler`、`backend/internal/service` 聚焦回归测试通过
- `backend/internal/handler`、`backend/internal/service`、`backend/internal/pkg/apicompat` 全包测试通过
- Vitest 实际跑完全量前端测试：125 个 test files / 687 个 tests 全部通过
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仍仅有既有 Vite dynamic import / chunk size warning

第九批 apicompat 参数清洗 / count_tokens 过滤同步追加验证：

```bash
cd backend
go test ./internal/pkg/apicompat ./internal/service -run 'Test(AnthropicToResponses_TemperatureStripped|ChatCompletionsToResponses_Temperature|ChatCompletionsToResponses_AssistantReasoningContentPreserved|GatewayService_AnthropicAPIKeyPassthrough_CountTokensFiltersGenerationFields)' -count=1
go test ./internal/pkg/apicompat ./internal/service -count=1
```

结果：

- `backend/internal/pkg/apicompat`、`backend/internal/service` 聚焦回归测试通过
- `backend/internal/pkg/apicompat`、`backend/internal/service` 全包测试通过

第十批 frontend / deploy / dependency 小修复同步追加验证：

```bash
bash -n deploy/install.sh
POSTGRES_PASSWORD=test docker compose -f deploy/docker-compose.yml config

cd frontend
CI=true pnpm install --frozen-lockfile
pnpm typecheck
pnpm build
```

结果：

- `deploy/install.sh` Bash 语法检查通过
- `CI=true pnpm install --frozen-lockfile` 通过；第一次沙箱内执行因 registry DNS 受限失败，联网重跑后完成并恢复 `node_modules`
- 上游 Vue 前端 typecheck / build 通过
- `pnpm build` 仍仅有既有 Vite dynamic import / chunk size warning
- `docker compose config` 未能执行：本机没有 `docker` 命令

### 18.3 明确未合并

| 主题 | 代表提交 / 范围 | 状态 | 原因 / 风险 |
|------|-----------------|------|-------------|
| 用户 x 平台配额体系 | `6b39b344`, `f7f5e338`, `06fca662` | 不合并 | 已决策排除；会引入 schema、Ent、preflight 和 UI 变更 |
| Airwallex / 多币种支付 | 支付 provider、webhook、支付 UI、DTO、测试 | 不合并 | 与 fork 现有 Mapay、订阅信用池、balance 混合支付冲突密集 |
| 与支付新能力强绑定的前端页面 | `frontend/src/components/payment/*`, `PaymentView.vue`, `SubscriptionsView.vue` 等 | 不合并 | 第一轮目标是 correctness，不重写支付/订阅产品线 |
| sponsor / workflow / README 噪音 | sponsor 文案、CI/workflow、上游 README 变化 | 不合并 | 对线上正确性无直接收益 |
| upstream migration `136`~`144` 原文件 | `backend/migrations/136_*.sql` ~ `144_*.sql` | 不直接合并 | 已与 fork migration 编号撞车；后续 schema 变更必须转写到 `900_*.sql` 命名空间 |

### 18.4 未合并但后续可评估

这些不是“不要”，而是本次不和第一批混做。

| 主题 | 代表提交 | 风险 | 后续建议 |
|------|----------|------|----------|
| OpenAI 账号冷却调度优化 | `1e406fed` | 高 | 29 个文件级别改动，属于调度系统改造 |
| OpenAI WS rate-limit failover | `08061717` | 高 | 改变 WS failover 行为，需压测或至少较完整 smoke |
| 模型 404 仅冷却账号模型组合 | `a31b5074` | 高 | 横跨 gateway/service/repository/test，需要单独主题分支 |
| OpenAI HTTP/2 response header timeout | `33ac8eb2` | 中到高 | 涉及底层传输、配置和部署样例 |
| 上游静默拒绝 failover | `6381f9e3` | 中到高 | 会改变 failover 判定语义，需先定义线上期望 |
| embeddings gateway | `ccace69d` | 中 | 能力扩展，不属于 correctness 第一批 |
| `/v1/models` 自定义模型列表 | `f597c158` | 中 | 会碰 group schema、handler、DTO、前端页面 |
| Channel Monitor OpenAI API mode / 模板 | `3eff5f51`, `b685fe69` | 中到高 | 会与已有监控页面、migration、设置项冲突 |
| Responses/Chat bridge developer role 修正 | `c4d7edba` | 中 | 当前 `dev` 没有 `chatcompletions_responses_bridge.go`；依赖未合入的 Responses/Chat fallback bridge 功能链 |
| Settings 暗色 tab shell 修复 | `b0c77233` | 低到中 | 当前 `dev` 的 `SettingsView.vue` 已是另一套 tab 结构，没有 upstream 的 `.settings-tabs-shell`，不适合机械套用 |

### 18.5 后续同步规则

后续继续按这条规则推进：

1. 继续禁止直接 merge `origin/main` 到 `dev`
2. 每次只吸收一个主题
3. 每个主题都更新本节状态
4. 涉及 schema 时不复用 upstream migration 文件名，统一走 `900_*.sql`
5. 只要碰支付、订阅、余额、风控、调度或 gateway 行为语义，就必须单独验证并记录风险
