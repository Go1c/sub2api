# 复刻 dragoncode.codes/dashboard 前端 — 实施计划

**创建日期**: 2026-04-22
**作者**: Go1c
**状态**: 已实施（本文保留作为历史记录）

> **品牌更名**：本计划以参考站名称 "Dragon Code" 作为工作名起草，产品最终品牌定为 **LumioAPI**。实施过程中的所有品牌字样以 LumioAPI 为准（见 `CLAUDE.md`、`doc/frontend-dashboard.md`）。

---

## 1. 背景与关键发现

目标站点 `https://dragoncode.codes/dashboard` 经抓包分析确认：

- **站点名**: "Dragon Code" / 页面标题 "Dragon - 企业级中转"
- **代码血缘**: 与本仓库 (`Wei-Shaw/sub2api` fork) **同源**，属于 Sub2API 的 UI 定制分支
- **技术栈**:
  - Vue 3.4 + Vite 5 + TypeScript
  - TailwindCSS + vue-i18n + Chart.js + Pinia + vue-router
- **视觉差异点（与本仓库现有 `frontend/` 的主要区别）**:
  - 品牌主色 `#4f8cff`（蓝），辅助色 `#1a2f5a` / `#1e3a8a`（navy）
  - 字体使用 Google Fonts 的 **Noto Serif SC** + **Source Serif 4**（衬线字体，辨识度高）
  - 站点名/Logo/文案为 "Dragon" 品牌

- **抓取到的用户侧组件命名**:
  `UserDashboardStats`、`UserDashboardCharts`、`UserDashboardRecentUsage`、`UserDashboardQuickActions`、`RechargeModal` 等。

**结论**: 这不是「从零复刻一个陌生站点」，而是「在已有同源代码基础上做主题/品牌/布局定制」。

---

## 2. 目标与非目标

### 目标
1. 新建独立目录 `frontend-dashboard/`，不动现有 `frontend/`。
2. 本地 `pnpm dev` 可预览，**前端独立运行**（使用 mock 数据，不强依赖后端）。
3. 视觉风格对齐 dragoncode.codes（主色、字体、布局、文案）。
4. 落地仓库分支策略：`main` 只同步上游，`dev` 开发，`publish` 源码 tag 发布。
5. 完成后更新 `doc/` 与 `CLAUDE.md`。

### 非目标
- 不动后端 Go 代码。
- 不替换 `frontend/`（上游同步需要）。
- 不复制 dragoncode.codes 的私有资源/付费系统密钥。
- 不追求像素级完全一致 —— 以"可识别为同款风格"为验收标准。

---

## 3. 分支与 Remote 策略

```
upstream (Wei-Shaw/sub2api)
    │
    │  (fetch only, 周期性合并)
    ▼
main  ──────────────────────────────┐
    │                                │
    │  (main 不再提交业务代码)          │
    ▼                                │
dev  ── 日常开发/PR 合入点 ───────────┤
    │                                │
    │  (dev 稳定后合并，或打 tag)       │
    ▼                                │
publish ── 源码 tag 语义的发布分支    ─┘
```

**具体操作**:
| 分支 | 来源 | 允许的操作 |
|------|------|-----------|
| `main` | 跟 `upstream/main` | 只做 `git fetch upstream && git merge upstream/main`，然后推 `origin/main` |
| `dev` | 从 `main` 切出 | 所有日常 commit、功能 PR 合入 |
| `publish` | 从 `dev` 合并 | 仅接收 tag-worthy 的稳定版本（源码语义，不含构建产物） |

**Remote 配置**:
```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
git fetch upstream
```

**保护**: 在 GitHub 上给 `main` 和 `publish` 加分支保护（禁止直推，强制 PR），留给日常 `dev`。

---

## 4. 目录结构设计

```
.
├── frontend/                      # 原 Sub2API 前端，保持与上游同步，不动
├── frontend-dashboard/            # 新建：dragoncode 风格的独立前端
│   ├── src/
│   │   ├── api/                   # 封装 axios，默认对 mock，可切真后端
│   │   ├── assets/
│   │   │   └── styles/
│   │   │       ├── tokens.css     # 颜色/字体/间距 CSS 变量
│   │   │       └── fonts.css      # Google Fonts 或本地衬线字体
│   │   ├── components/
│   │   │   ├── dashboard/         # UserDashboardStats/Charts/RecentUsage/QuickActions
│   │   │   ├── layout/            # Sidebar/Header/RechargeModal
│   │   │   └── common/
│   │   ├── composables/
│   │   ├── i18n/                  # zh-CN 默认 + en 占位
│   │   ├── mock/                  # msw 或本地 json，覆盖 dashboard 所需接口
│   │   ├── router/
│   │   ├── stores/                # Pinia
│   │   ├── views/                 # HomeView/LoginView/DashboardView/KeysView/UsageView/ProfileView …
│   │   ├── App.vue
│   │   └── main.ts
│   ├── public/
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── README.md
├── doc/                           # 新建：项目文档根
│   ├── plan/
│   │   ├── frontend-dashboard-replica.md    # 本文件
│   │   └── assets/                # 抓包截图/原型图
│   ├── branching.md               # 分支策略速查
│   └── frontend-dashboard.md      # 模块说明（完成后写）
└── CLAUDE.md                      # 新建/更新：给 Claude 的项目指引
```

---

## 5. 实施步骤（详细）

### 阶段 0 — 仓库准备
1. 确认 `upstream` remote：`git remote add upstream https://github.com/Wei-Shaw/sub2api.git && git fetch upstream`。
2. 切出分支：
   ```bash
   git checkout main
   git checkout -b dev
   git push -u origin dev
   git checkout -b publish
   git push -u origin publish
   git checkout dev
   ```
3. GitHub 侧为 `main` 与 `publish` 加分支保护规则。
4. 创建 `doc/branching.md` 写明工作流（见 §7）。

### 阶段 1 — 脚手架
5. 在 `frontend-dashboard/` 初始化 Vite + Vue 3 + TS 工程。
   - 依赖同版本对齐现有 `frontend/package.json` 的关键库（vue 3.4、vite 5、tailwind 3.4、vue-i18n 9、pinia 2、vue-router 4、chart.js 4、axios、dompurify、marked）。
6. 配置 `vite.config.ts`：`@` 指向 `src/`；端口 `5174`（避开现 `frontend/` 可能占用的 5173）。
7. 配置 `tailwind.config.js`：引入 `#4f8cff` 主色与 navy 辅色作为 `brand.500` / `brand.900`。
8. `index.html` 引入 Google Fonts：
   ```html
   <link href="https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@400;500;700&family=Source+Serif+4:opsz,wght@8..60,400;8..60,500;8..60,700&display=swap" rel="stylesheet" />
   ```
9. `src/assets/styles/tokens.css` 定义 CSS 变量：
   ```css
   :root {
     --color-brand: #4f8cff;
     --color-brand-dark: #1a2f5a;
     --color-bg: #f9fafb;
     --color-surface: #ffffff;
     --color-text: #0f172a;
     --font-serif: "Source Serif 4", "Noto Serif SC", ui-serif, Georgia, serif;
   }
   ```
10. `pnpm dev` 能打开空白 Vue 页。

### 阶段 2 — 基础框架（登录外布局）
11. 路由骨架：`/login` `/register` `/dashboard` `/keys` `/usage` `/profile` `/groups`（admin 路由先不做）。
12. `AuthLayout` 与 `AppLayout`（带 Sidebar + Header）。
13. i18n 最小集：`zh-CN.json` 填充导航/按钮文案（可直接抄本仓库 `frontend/src/i18n/`，品牌名替换为 "Dragon Code"）。
14. Pinia `authStore` 存 token（mock 登录即可通过）。

### 阶段 3 — Dashboard 核心页
15. 实现 `DashboardView.vue`，组合：
    - `UserDashboardStats`：4 个指标卡（API Keys / 今日用量 / 余额 / 请求数）。
    - `UserDashboardCharts`：Chart.js 折线图（近 7/30 天用量）。
    - `UserDashboardRecentUsage`：最近请求列表（简化表格）。
    - `UserDashboardQuickActions`：入口卡（新建 Key / 充值 / 查看文档）。
    - `RechargeModal`：占位弹窗，按钮不对接真支付。
16. `src/mock/` 目录用简单 `axios` interceptor 拦截以 `/api/v1/user/dashboard/*` 开头的请求，返回静态 JSON。
   - 优先方案：自写 interceptor（零依赖）；若后续复杂再引入 msw。

### 阶段 4 — 其他主要页面（骨架）
17. `KeysView`、`UsageView`、`ProfileView`、`GroupsView` 先做**列表骨架 + mock 数据**，不追求功能完整，仅保证视觉/交互可跑通。

### 阶段 5 — 视觉细节
18. 核对 dragoncode 首页/登录页布局（参考 `doc/plan/assets/` 下截图），调整 Sidebar 宽度、Header 阴影、卡片圆角与阴影。
19. 暗色主题：**后置**，作为 stretch goal。

### 阶段 6 — 本地预览与 QA
20. 手动点测 3 条主路径：登录 → Dashboard、Keys 列表、Usage 图表。
21. `pnpm typecheck` 通过。
22. Chrome DevTools 移动端视窗 (375px / 768px / 1280px) 巡检布局。

### 阶段 7 — 文档与 CLAUDE.md 收尾
23. 写 `doc/frontend-dashboard.md`：模块说明、mock 机制、如何切真后端。
24. 写/更新 `CLAUDE.md`（仓库根）：
    - 项目结构（`frontend/` vs `frontend-dashboard/` 的分工）
    - 分支策略 (`main` / `dev` / `publish`)
    - 常用命令（pnpm dev、typecheck、upstream sync）
    - 不做什么（不改 `frontend/`、不直推 `main`/`publish`）
25. 更新根 `README.md` 增加一节「Dragon Code Dashboard 变体」指向 `frontend-dashboard/`。

---

## 6. 验收标准

- [ ] `cd frontend-dashboard && pnpm install && pnpm dev` 可在 `http://localhost:5174` 打开。
- [ ] 无后端运行时，Dashboard/Keys/Usage 三页都有 mock 数据显示，不报红错。
- [ ] 视觉特征对齐：蓝色主题 + 衬线字体 + navy 渐变 hero 区。
- [ ] `pnpm typecheck` 与 `pnpm build` 均通过。
- [ ] 分支 `main` `dev` `publish` 均已创建并推送到 origin。
- [ ] `git remote -v` 能看到 `upstream Wei-Shaw/sub2api`。
- [ ] `doc/frontend-dashboard.md`、`doc/branching.md`、`CLAUDE.md` 均已写。

---

## 7. 分支工作流速查（将抽为 doc/branching.md）

**同步上游**（定期执行，仅在 `main` 上）:
```bash
git checkout main
git fetch upstream
git merge upstream/main      # 如无冲突
git push origin main
# 再把 main 的变更带进 dev：
git checkout dev
git merge main
git push origin dev
```

**日常开发**:
```bash
git checkout dev
git checkout -b feat/xxx
# ... commits ...
git push -u origin feat/xxx
# 开 PR 合入 dev
```

**发布**:
```bash
git checkout publish
git merge --no-ff dev
git tag -a vX.Y.Z -m "release X.Y.Z"
git push origin publish --tags
```

---

## 8. 风险与权衡

| 风险 | 应对 |
|------|------|
| upstream 前端升级时，`frontend-dashboard/` 可能漏同步新 API | 接受 —— 两个前端解耦是主动选择；关键接口变化在 doc 里手工追踪 |
| Google Fonts 在部分网络环境加载慢 | 准备本地字体 fallback，`@font-face` 指向 `public/fonts/` |
| mock 与真后端 schema 漂移 | `src/api/` 保留类型定义，mock 只在 dev 环境注入，`VITE_USE_MOCK=false` 可关闭 |
| 用户后续需要像素级还原 | 先做 80%，留 `doc/plan/assets/` 截图对照清单作为后续迭代依据 |

---

## 9. 工时估算

| 阶段 | 预估 |
|------|------|
| 阶段 0 仓库准备 | 0.5h |
| 阶段 1 脚手架 | 1h |
| 阶段 2 基础框架 | 2h |
| 阶段 3 Dashboard 核心 | 3h |
| 阶段 4 其他页面骨架 | 2h |
| 阶段 5 视觉细节 | 2h |
| 阶段 6 QA | 1h |
| 阶段 7 文档 | 1h |
| **合计** | **~12.5h** |

---

## 10. 开工前待确认

1. 是否允许使用 Google Fonts CDN（否则预下载到 `public/fonts/`）？
2. Sidebar 导航项以 dragoncode 的 `Dashboard / Keys / Usage / Groups / Profile` 为准，admin 相关路由是否本期也做？
3. `publish` 是否需要打首个 tag `v0.1.0` 作为"空基线"？
