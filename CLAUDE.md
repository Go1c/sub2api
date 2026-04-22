# CLAUDE.md — sub2api fork 指引

本仓库是 [`Wei-Shaw/sub2api`](https://github.com/Wei-Shaw/sub2api) 的 fork，在此基础上增加了 **LumioAPI 品牌的独立前端（视觉参考 dragoncode.codes）** (`frontend-dashboard/`)，最终部署目标是 **Zeabur**。

## 项目结构（核心）

| 路径 | 内容 | 维护原则 |
|------|------|---------|
| `backend/` | Go 后端（Gin + Ent + PostgreSQL + Redis）— 来自 upstream | **不主动改**，与 upstream 同步 |
| `frontend/` | 上游 Sub2API 的 Vue 3 前端（teal 配色）| **不主动改**，与 upstream 同步 |
| `frontend-dashboard/` | LumioAPI 前端（blue 主色，视觉参考 dragoncode.codes）| **本仓库开发焦点**，在 `dev` 分支迭代 |
| `doc/` | 项目文档（本分支专属）| 任何改动都同步更新 |
| `deploy/` | Docker / 安装脚本 — 来自 upstream | 不主动改 |

**黄金规则**：对 `backend/` 和 `frontend/` 的任何改动都会在下次 upstream 同步时增加冲突负担。除非修 upstream 的 bug 并准备提交 PR 回去，否则**不要碰这两个目录**。所有业务迭代在 `frontend-dashboard/`。

## 分支策略

| 分支 | 用途 |
|------|------|
| `main` | 只跟 `upstream/main` 同步，不做开发 |
| `dev`  | 日常开发，所有 PR 合入点 |
| `publish` | 源码 tag 语义的发布分支，Zeabur 从这里部署 |

完整工作流见 [`doc/branching.md`](doc/branching.md)。

**Remote 配置必须**：
```bash
git remote -v
# origin    https://github.com/Go1c/sub2api
# upstream  https://github.com/Wei-Shaw/sub2api.git    ← 必须存在
```

## 常用命令

### frontend-dashboard（本仓库重点）

```bash
cd frontend-dashboard
pnpm install          # 首次，或改 package.json 后
pnpm dev              # http://localhost:5174，支持 HMR
pnpm typecheck        # 严格 TS 检查
pnpm build            # 产物到 dist/
pnpm preview          # 预览 dist/，http://localhost:4174
```

若系统没 pnpm：`corepack enable && corepack prepare pnpm@9 --activate`，或 `npx -y pnpm@9.15.0 <cmd>`。

### 同步上游

```bash
git checkout main && git fetch upstream && git merge --ff-only upstream/main && git push origin main
git checkout dev && git merge main && git push origin dev
```

### 发布

```bash
git checkout publish && git merge --no-ff dev -m "release: vX.Y.Z"
git tag -a vX.Y.Z -m "release X.Y.Z" && git push origin publish --tags
```

## 工作约定（给 Claude 的硬规则）

1. **本地验证优先**：任何代码改动，commit 前至少跑 `pnpm typecheck` + `pnpm build`。涉及 UI 的改动，dev server 起来 curl 过、或请求用户浏览器确认；不要只凭类型检查就报"完成"。
2. **不直推 `main` / `publish`**：这两个分支只接受合并 /tag 操作。日常 commit 全部走 `dev`。
3. **不改 `frontend/` 与 `backend/`**（除非用户明确要求或准备提 upstream PR）。
4. **Zeabur 约束**：部署相关改动按 PaaS 约束——环境变量走 `VITE_*` 前缀、端口走 `PORT`、不引入需要宿主 volume 的依赖。详见 [`doc/deploy-zeabur.md`](doc/deploy-zeabur.md)。
5. **文档同步**：对 `frontend-dashboard/` 结构、环境变量、脚本等的改动，同步更新 [`doc/frontend-dashboard.md`](doc/frontend-dashboard.md)。

## 关键文档索引

- [`doc/plan/frontend-dashboard-replica.md`](doc/plan/frontend-dashboard-replica.md) — 初始复刻计划（历史记录）
- [`doc/branching.md`](doc/branching.md) — 分支与同步工作流
- [`doc/frontend-dashboard.md`](doc/frontend-dashboard.md) — 新前端模块说明
- [`doc/deploy-zeabur.md`](doc/deploy-zeabur.md) — Zeabur 部署指南
- 根 `README.md` — upstream 原有说明（注意：许多安装/部署步骤针对上游整包，不完全适用本仓库新增前端）

## 视觉识别

LumioAPI 的关键视觉点（沿用 dragoncode.codes 的观感），改样式时以此为北极星：

- 主色 `#4f8cff`（brand-500）配 navy `#1a2f5a`（brand-900）渐变
- 标题/品牌文案用衬线 `Source Serif 4` + `Noto Serif SC`（Google Fonts）
- 辅助 UI 文字才用 system sans（`.ui-sans` 类）
- 数字用 `ui-mono`（`.ui-mono` 类），便于对齐
