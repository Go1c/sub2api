# sub2api / LumioAPI — 中心文档

`Wei-Shaw/sub2api` 的 fork(线上品牌 **LumioAPI**)。Go 后端(Gin + Ent + PostgreSQL + Redis,来自 upstream)+ Vue 3 主前端(`frontend/`,当前开发焦点)。本仓库按 **LumioAgent** 规范组织:**主 Agent 调度,子 Agent 执行,Skill 是方法,.md 是规则。**

主 Agent(= 宿主主 loop)理解目标、拆任务、调度、收口,自己不写代码;把活分给职能化的子 Agent,子 Agent 用 Skill 执行可复用流程,所有人共享 `.spec/` 下的 .md 作为规则与知识。

> 项目知识(`knowledge/README.md` 导航)、硬性禁令(`rules/system.md`)都经 `CLAUDE.md` 的 `@import` 每次 init 强制载入,本文件不再复述。子 Agent 规范在 `agents/`,技能在 `skills/`;沉淀 / 同步任何能力 → 用 `spec-steward` 技能。

## 项目结构(核心)

| 路径 | 内容 | 维护原则 |
|------|------|---------|
| `backend/` | Go 后端 — 来自 upstream | **不主动改**,与 upstream 同步 |
| `frontend/` | Vue 3 主前端(当前线上入口) | **本仓库当前前端开发焦点** |
| `.spec/` | 规范 / 知识 / 子 Agent / 技能(本分支专属) | 任何改动都同步索引 |
| `deploy/` | Docker / 安装脚本 — 来自 upstream | 不主动改 |

**黄金规则**:对 `backend/` 和 `frontend/` 的改动都要控制范围,优先做必要改动并保持可验证,避免无关重构增加后续 upstream 同步负担。

## 调度核心

**子 Agent 名册**(便利镜像;权威是各 `.agent.md`):

| 名称 | 职责 | 何时调度 |
|------|------|----------|
| `planner` | 把模糊目标转成清晰、可执行、带验收标准的任务卡 | 需求不清时 |
| `coder` | 根据任务卡编写 / 修改代码 | 有明确任务卡时 |

- **默认流程:** `planner`(拆解)→ `coder`(实现)→ 主 loop 收口。需求清晰且简单 → 跳过 planner,直接派 coder。
- **谁来调度:** 只有主 loop 调度子 Agent;被调用的子 Agent 只执行、不再派活。
- **失败处理:** coder 不达标退回重做;同一问题三次仍不过,停下质疑方案本身。
- **上下文隔离:** 每个子 Agent 在自己的上下文里只拿任务卡 + 相关文件。

## 宿主差异

| 能力 | Claude Code | Codex |
|------|-------------|-------|
| 任务持久化 | `TaskCreate` / `TaskUpdate` / `TaskList` | `.spec/tasks/<slug>.md`(frontmatter `status`)|
| 子 Agent 发现 | `.claude/agents/` 自动发现 `.agent.md` | 主 loop 手动读 `.spec/agents/*.agent.md` |
| 技能加载 | `.claude/skills/` 自动发现 | `.agents/skills/` 索引,手动调用 |

Codex 主 loop 本地执行角色规范:需求不清时读 `planner.agent.md` 并用 `task-breakdown`;任务明确时读 `coder.agent.md` 并按需用 `test-driven-development` / `spec-steward`。只有用户明确要求并行时,才用 Codex 多代理工具。

## 视觉识别(LumioAPI)

改样式时以此为北极星:

- 主色 `#4f8cff`(brand-500)配 navy `#1a2f5a`(brand-900)渐变
- 标题 / 品牌文案用衬线 `Source Serif 4` + `Noto Serif SC`
- 辅助 UI 文字用 system sans(`.ui-sans` 类),数字用 `.ui-mono`(便于对齐)

详见 [`knowledge/standards/code-style.md`](knowledge/standards/code-style.md)。

> 调度 / 协作的**硬性禁令**(不得再派生子 Agent、frontmatter 限制、调度变更须同步、fork 红线)在 [`rules/system.md`](rules/system.md)。
