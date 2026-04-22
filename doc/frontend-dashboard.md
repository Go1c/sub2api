# frontend-dashboard 模块说明

Dragon Code 风格的独立前端，复刻 `https://dragoncode.codes/dashboard` 的视觉与结构。与上游同步的 `frontend/` 解耦，可离线运行。

## 位置与关系

```
repo-root/
├── frontend/              # ← 上游 Sub2API 前端（teal 配色），跟 upstream 同步，不要改
├── frontend-dashboard/    # ← 本模块（blue + serif 配色），独立工程
└── doc/                   # 本目录
```

## 技术栈

| 类别 | 选型 |
|------|------|
| 框架 | Vue 3.5 + TypeScript |
| 构建 | Vite 5 |
| 样式 | TailwindCSS 3 + CSS 变量令牌 |
| 状态 | Pinia |
| 路由 | Vue Router 4 |
| i18n | vue-i18n 9（zh-CN 默认，en-US 回落） |
| 图表 | Chart.js 4 + vue-chartjs 5 |
| HTTP | axios + 自定义 mock 拦截器 |

## 本地开发

```bash
cd frontend-dashboard
# 任选一种启 pnpm:
#   A) corepack enable && corepack prepare pnpm@9 --activate
#   B) npm install -g pnpm@9
#   C) 每次用 npx -y pnpm@9.15.0 <cmd>
pnpm install
cp .env.example .env      # 可选，默认已能跑
pnpm dev                  # http://localhost:5174
```

### 常用脚本

| 命令 | 作用 |
|------|------|
| `pnpm dev` | dev server, 默认端口 5174, HMR 启用 |
| `pnpm build` | 产物输出到 `dist/` |
| `pnpm preview` | 预览 `dist/`，默认端口 4174 |
| `pnpm typecheck` | vue-tsc 严格类型检查 |
| `pnpm lint` | ESLint 自动修复 |

## 目录

```
src/
├── api/http.ts              # axios 实例 + 401 拦截 + token 注入
├── assets/styles/tokens.css # 颜色/字体/阴影 CSS 变量
├── components/
│   ├── dashboard/           # UserDashboardStats/Charts/RecentUsage/QuickActions + RechargeModal
│   └── layout/              # Sidebar / TopBar / AppLayout / AuthLayout
├── composables/             # （占位）组合式函数
├── i18n/                    # zh-CN.ts / en-US.ts / index.ts
├── mock/                    # 离线拦截 + 静态 fixtures
├── router/index.ts          # 路由 + 未登录重定向到 /login
├── stores/auth.ts           # Pinia 认证 store, 持久化到 localStorage
├── views/
│   ├── LoginView.vue / RegisterView.vue
│   ├── DashboardView.vue
│   ├── KeysView.vue / UsageView.vue / GroupsView.vue / ProfileView.vue
│   └── NotFoundView.vue
├── App.vue / main.ts / style.css / env.d.ts
```

## Mock 机制

`src/mock/index.ts` 在启动时判断：

```ts
shouldMock = VITE_USE_MOCK !== 'false' && !VITE_API_BASE_URL
```

若启用，将挂接一个 axios 请求拦截器，按路径正则（见 `handlers` 表）匹配后通过 `config.adapter` 短路返回静态 fixture，HTTP 200，`{ code: 0, data, msg: 'ok' }`。

**切换真后端**（例如本地起了 `backend/`）：

```bash
# .env
VITE_API_BASE_URL=http://localhost:8080
VITE_USE_MOCK=false
```

## 设计令牌

| Token | 值 | 用途 |
|-------|------|------|
| `--color-brand` / `brand-500` | `#4f8cff` | 主色，按钮、链接、图表 primary line |
| `--color-brand-dark` / `brand-900` | `#1a2f5a` | 侧栏渐变终点、hero 背景 |
| `--font-serif` | `Source Serif 4, Noto Serif SC, …` | 标题、品牌文案 |
| `--font-sans` | system stack + `PingFang SC` 等 | 辅助 UI 文字 |

## 本地验证清单

在开 PR 前，至少跑通：

- [ ] `pnpm typecheck` 无 error
- [ ] `pnpm build` 成功，无 WARN
- [ ] `pnpm dev` 能打开并点完三条主路径：
  - `/login` → `/dashboard`（触发 mock 登录）
  - `/dashboard` 首屏加载完成、图表渲染、充值弹窗可开关
  - `/keys` `/usage` `/groups` `/profile` 各页能切换无红错
- [ ] Chrome DevTools 切到 375px 视宽，侧栏不遮罩内容

## 定价数据同步策略

**硬规则：前端绝不 hardcode 价格**。

首页 `ModelPricingSection` 通过 `fetchPublicPricing()` 调 `GET /api/v1/pricing/public`，后端负责：

1. 从 `channel_model_pricing` 表读取各 channel 的原始 input_price/output_price
2. 按 `groups.input_price/output_price` 或上层 multiplier 派生前端展示的倍率/折扣
3. 返回如下形状：

```ts
interface PublicPricing {
  currency: string     // 'CNY'
  unit: string         // '百万 tokens'
  rateNote: string     // 显示在表格上方的说明行（含汇率）
  rows: Array<{
    model: string           // 'Opus 4.6'
    group: string           // '企业稳定版' / 'kiro 逆向' / 'codex' …
    multiplier: string      // '2.5x'
    inputPrice: number      // CNY per 1M tokens
    outputPrice: number
    officialInput: number   // 官方 CNY 折算
    officialOutput: number
    discount: string        // '3.6折'
    openClaw: boolean
  }>
}
```

**后端未实现时**：`src/mock/fixtures.ts -> publicPricing` 是唯一数据源。**手动维护它与后端配置保持一致**，避免前端页面和实际计费脱节。

**后端实现上线后**：删除 mock 里对 `/pricing/public` 的拦截即可，生产环境 `VITE_API_BASE_URL` 指向后端自动生效。

**禁止**：
- 在 Vue 组件里直接写 `¥12.50` 之类的字面量（review 时会被打回）
- 把价格放进 i18n 文案（语言无关的数据不该混进文案层）

## 与 upstream 同步时的冲突规避

- 本模块**不引用** `frontend/` 的任何文件，upstream 对 `frontend/` 的改动不会影响这里。
- 如需抄 upstream 的组件逻辑，**复制**而非 symlink，避免双向耦合。
- 公共资源（图标、字体）优先放 `frontend-dashboard/public/` 或 `src/assets/`，不要引用 `frontend/` 的。
