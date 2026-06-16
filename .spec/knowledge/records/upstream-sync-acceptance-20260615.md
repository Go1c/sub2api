---
name: upstream-sync-acceptance-20260615
description: upstream v0.1.136 同步的验收清单——上线/回归时核对「合了什么、风险在哪、怎么验」。
metadata:
  type: record
  date: 2026-06-15
  status: 归档
---

# Upstream v0.1.136 同步 — 验收清单（2026-06-16）

> 本文件面向**验收**：合了什么、风险在哪、怎么验。过程细节见 [`upstream-sync-20260615.md`](upstream-sync-20260615.md)。

## 概览

- **同步对象**：upstream `Wei-Shaw/sub2api` v0.1.136（`e34ad2b1`），fork `main` 已对齐。
- **并入 dev 的 PR**：13 个（#69/#72/#73/#74/#75/#76/#77/#78/#79/#80/#81/#82）+ 用户 #70。
- **dev CI**：全绿（`backend-security`、`frontend-security` 均已修复）。
- **原则**：只并「高风险（bug/安全/正确性/性能）」与「自包含低风险」；可有可无的功能不并。

---

## 🔴 高风险 — 必须重点验收

### 1. Bedrock 请求兼容（#80）— ⭐最需验
**改了什么**：Bedrock 账号发请求时，自动清理 Claude Code / opus-4.x / fable-5 带的 Bedrock 不支持字段（`thinking.type=enabled`→`adaptive`、删 `service_tier`/`interface_geo`、清洗非法 `tool_use` ID、过滤不支持的 `anthropic_beta`）。
**风险**：① 对**所有** Bedrock 请求无条件生效（dev 没有渠道级开关）；② `sanitizeBedrockCCFields` 是改造版，**不删 `context_management`**（仍由 fork 原有逻辑按 beta token 决定）。
**怎么验**：
- [ ] 用 **Bedrock 账号** + Claude Code（或 opus-4-7/4-8/fable-5）发请求，确认**不再报 ValidationException**、能正常返回。
- [ ] 确认带 `context-management` beta token 的请求，`context_management` 字段**仍被保留**（fork 精细逻辑没被破坏）。
- [ ] 确认普通 Bedrock 请求（非 CC）行为无回归。

### 2. API Key 独占分组访问守卫（#81）— ⭐安全
**改了什么**：API key 绑定独占分组时，若用户被移出该分组授权 / 分组被停用或删除，鉴权直接 403 拒绝（之前会越权放行）。
**风险**：触碰 API key 鉴权热路径 + 鉴权缓存（snapshot 版本 9→10，上线后旧缓存会刷新一次）。范围决策：**未覆盖**「分组已删除但 key 仍绑定」（dev 删分组时已清空 key 的 group_id，不会出现该场景）。
**怎么验**：
- [ ] 把某用户从一个**独占分组**的 allowed_groups 移除 → 该用户用绑定该分组的 key 请求应被拒（403 `GROUP_NOT_ALLOWED`）。
- [ ] **停用**一个分组 → 绑定它的 key 请求应被拒（403 `GROUP_DISABLED`）。
- [ ] **订阅型分组**不受影响（仍正常）。
- [ ] 改完用户 allowed_groups **立即生效**（鉴权缓存已失效，不用等 TTL）。
- [ ] 非独占分组、未绑定分组的 key 行为无回归。

### 3. 余额改为指针类型（#72 / `0560340b`）
**改了什么**：用户 balance 从 `float64` 改为指针（可区分「未设置 nil」与「0」）。
**风险**：所有读余额的地方（展示、扣费、余额闸门、通知）若把 nil 当 0 会出错。
**怎么验**：
- [ ] 余额展示、扣费、余额不足拦截、余额通知，确认正常；尤其新用户/无余额记录时不报错、不误判为 0。

### 4. Claude Code 客户端识别改用 billing block（#72 / `d626ccce`）
**改了什么**：识别请求是否来自 Claude Code，从「只看 prompt」改为「看 billing attribution block」。
**风险**：影响哪些请求被判为 CC（关系到计费/路由）。
**怎么验**：
- [ ] 确认 Claude Code 流量仍被正确识别与计费；非 CC 流量没被误判为 CC。

### 5. OpenAI 5h/7d 用量窗口（#73 `16bc8769` + #72 `bf1a2d6d` + #82 `12cecca7`）
**改了什么**：OpenAI 账号的 5h ResetsAt 对齐会话窗口结束、过期窗口清零、Codex 用量统计对齐 reset 窗口、5h 用量百分比语义修正。
**风险**：用量记账口径调整，需与 fork 的订阅/用量窗口逻辑一致。
**怎么验**：
- [ ] OpenAI 账号的 **5h/7d 用量窗口**显示、自动重置、百分比正确；与实际用量吻合。

---

## 🟡 中风险 — 抽查

### 6. OpenAI 网关 8 个正确性修复（#82）
**改了什么**：非流式响应强制 `Content-Type: application/json`、stream 字段类型校验、tool_use/tool_result 配对修复（Responses→Anthropic）、传 prompt cache key、防错误帧双写、ws 用量去重、responses 流终止输出归一化。
**怎么验**：
- [ ] OpenAI / Anthropic 兼容网关请求正常；非流式响应 Content-Type 正确；工具调用（tool_use）链路正常。

### 7. Go 1.26.4 工具链（#75）
**改了什么**：`go.mod` + 3 个 Dockerfile + 3 个 CI workflow 升到 1.26.4（修掉 stdlib 漏洞）。`backend/Dockerfile` 之前停在 1.25.7，一并升到 1.26.4。
**怎么验**：
- [ ] 本地/CI Docker 镜像用 `golang:1.26.4` 能正常构建、部署、启动。

### 8. 账号额度窗口归一化（#72 / `fb0195f3`）
- [ ] 编辑账号的固定额度窗口后，窗口边界正确。

---

## 🟢 低风险 — 看一眼即可

| 项 | PR | 验收 |
|---|---|---|
| 新增模型 `claude-opus-4-8` / `claude-fable-5` 可选 | #79 | 前端模型白名单/映射能选这两个；Antigravity 平台映射生效 |
| /admin/usage 查看已删除用户历史用量 | #73 | 管理端能查 |
| /admin/usage 提速 | #73 | 打开/刷新更快 |
| 用量明细表可空字段崩溃修复 | #77 | 含 null 字段的用量明细不再整表空白 |
| Select 下拉高度、用量窗口 tooltip、缓存命中文案 | #73/#77 | UI 正常 |
| /admin/users 按 API Key 分组过滤、ops 监控指标、proxy 质量分类、分组描述清空持久化 | #72 | 各自正常 |

---

## ✅ 安全修复（已并，CI 已转绿）

- CWE-79 存储型 XSS（API key 名转义）、CWE-204 ID-oracle（未授权返回 404）— #69
- go1.26.3 stdlib 漏洞 GO-2026-5039/5037 — #75（toolchain 升级）
- `form-data` CRLF 注入 GHSA-hmw2-7cc7-3qxx — #76（override ≥4.0.6）

---

## 🚫 明确未并入（本次不做）

| 项 | 原因 |
|---|---|
| ~20 个 OpenAI 网关 fix（failover/错误透传/ws 桥接/跨组鉴权等） | 困在 fork 深度改造的 `gateway_service.go` 热路径 + 缺失子系统（`MarkResponseCommitted` 防 double-write、codex 路由耦合），需专门的 gateway↔upstream 大对齐 |
| 删除审计 / 失败请求观测子系统 | 可有可无的运营功能 |
| auth LinuxDO 登录修复（`aea2950b`） | 依赖 `LoginOrRegisterOAuthWithTokenPair` 6 参签名前置，dev 是 5 参 |
| CLI 指纹版本 bump、codex 模拟保真度、纯优化/refactor | 非高风险功能 |
| Bedrock region 模型 ID 映射 | 非必核（运营可手配） |

> 如需把「~20 个 OpenAI fix」或某个子系统补上，需另立一次专门的 gateway↔upstream 大对齐评估（高风险、需逐处 Review）。

---

## 一句话给验收人

**重点验前 5 项（🔴）**：Bedrock 兼容、独占分组越权、余额指针、CC 识别、OpenAI 用量窗口 —— 这几处是行为/安全/计费相关，最可能影响线上。其余抽查即可。dev CI 已全绿。
