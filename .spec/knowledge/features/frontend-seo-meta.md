---
name: frontend-seo-meta
description: 前端 GEO/SEO 头部元数据与爬虫可见简介——静态 index.html 默认值、injectSEOMeta 替换、#app 内可见 H1；改首页爬虫可见性或 injectSiteTitle 链路时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 前端 GEO/SEO 头部元数据

简介：`frontend/index.html` 自带 description / canonical / OG / Twitter / JSON-LD，以及 `#app` 内一段无隐藏样式的爬虫简介。Go embed 在配置了 `site_name` 时用 `injectSEOMeta` 替换 `<!--seo-meta-->` 块，不追加第二份标签。

## 背景 / 目标

- 不执行 JS 的搜索引擎和 AI 爬虫（含 Googlebot / GPTBot / ClaudeBot / PerplexityBot）必须从**第一次 HTML 解析**读到站点简介。
- Google 把 `display:none` / `hidden` H1 当作不存在；简介必须在 `#app` 内且不可隐藏。
- 生产默认文案面向 `https://api.lumio.games/`；`injectSiteTitle` / `__APP_CONFIG__` 注入不得被破坏。
- Vue mount 后自然替换 `#app` 内容；不要为此加 hiding CSS。

## 设计

- **静态默认**（`frontend/index.html`）：description / canonical / og:* / twitter:* / JSON-LD（Organization + WebSite，含 `email=admin@lumio.games`）。简介是 `#app` 内的 `<main><h1>…</h1><p>…</p></main>`，**禁止** `hidden` / `display:none` / `visibility:hidden` / `aria-hidden`，也禁止 `#seo-crawler-intro` 这类外壳。
- **注入链路**：`injectSettings` 仍先写 `window.__APP_CONFIG__`，再 `injectSiteTitle` → `injectSiteFavicon` → `injectSEOMeta`。
- **`injectSEOMeta`**：仅当 `site_name` 非空才改写 `<!--seo-meta-->…<!--/seo-meta-->`。无标记时才退回在 `</head>` 前插入。空 `site_name` 保留静态默认，避免把生产 LumioAPI 文案冲成 Sub2API。
- **字段**：description / OG / Twitter 用 `{site_name} 是 AI API 中转与管理平台…`。`api_base_url` 合法则写 canonical / `og:url` / JSON-LD `url`（根路径补 `/`）；否则沿用 HTML 里已有 canonical。JSON-LD 的 `email` 从现有 JSON-LD 继承，Go 源码不写死品牌邮箱。属性值 `html.EscapeString`；JSON-LD `json.Marshal`。
- **不要改** `vite.config.ts` / 路由 / HomeView 业务 UI / 公开设置 schema / robots.txt / sitemap.xml。线上 robots/sitemap 是 Caddy 静态文件，不在本 git 树。

## 已决策

- 静态模板使用生产站默认文案；运行时仍可由 `site_name` / `api_base_url` 覆盖头部标签。
- 简介必须放在 `#app` 内且对首轮解析可见。Vue mount 替换即可；藏在 `#app` 外并用 `display:none` 会让 Google 把首页当成薄 SPA 壳（GSC 2026-09-03：crawled, not indexed）。
- JSON-LD 走 `json.Marshal`：Go 会把 `<>&` 编成 `\u003c` 等，二次 HTML escape 会破坏 JSON。

## 待解决

- 无。

## 相关

- 代码：`frontend/index.html`、`backend/internal/web/embed_on.go`、`backend/internal/web/embed_test.go`
- 测试：`frontend/src/views/__tests__/HomeViewSeo.spec.ts`；`go test -tags=embed ./internal/web/`（必须先 `pnpm build` 刷新 `backend/internal/web/dist/`）
