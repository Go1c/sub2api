---
name: frontend-seo-meta
description: 前端 GEO/SEO 头部元数据与爬虫可见简介——静态 index.html 默认值、injectSEOMeta 替换、隐藏简介；改首页爬虫可见性或 injectSiteTitle 链路时查这篇
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 前端 GEO/SEO 头部元数据

简介：`frontend/index.html` 自带 description / canonical / OG / Twitter / JSON-LD，以及 `#app` 外一段 `display:none` 的爬虫简介。Go embed 在配置了 `site_name` 时用 `injectSEOMeta` 替换 `<!--seo-meta-->` 块，不追加第二份标签。

## 背景 / 目标

- 不执行 JS 的搜索引擎和 AI 爬虫（GPTBot / ClaudeBot / PerplexityBot）必须从原始 HTML 读到站点简介。
- 生产默认文案面向 `https://api.lumio.games/`；`injectSiteTitle` / `__APP_CONFIG__` 注入不得被破坏。
- 浏览器首屏不能出现这段简介。

## 设计

- **静态默认**（`frontend/index.html`）：description / canonical / og:* / twitter:* / JSON-LD（Organization + WebSite，含 `email=admin@lumio.games`）。简介在 `#app` 外的 `#seo-crawler-intro[hidden]`，并用 `#seo-crawler-intro { display: none !important; }` 从首帧隐藏。
- **注入链路**：`injectSettings` 仍先写 `window.__APP_CONFIG__`，再 `injectSiteTitle` → `injectSiteFavicon` → `injectSEOMeta`。
- **`injectSEOMeta`**：仅当 `site_name` 非空才改写 `<!--seo-meta-->…<!--/seo-meta-->`。无标记时才退回在 `</head>` 前插入。空 `site_name` 保留静态默认，避免把生产 LumioAPI 文案冲成 Sub2API。
- **字段**：description / OG / Twitter 用 `{site_name} 是 AI API 中转与管理平台…`。`api_base_url` 合法则写 canonical / `og:url` / JSON-LD `url`（根路径补 `/`）；否则沿用 HTML 里已有 canonical。JSON-LD 的 `email` 从现有 JSON-LD 继承，Go 源码不写死品牌邮箱。属性值 `html.EscapeString`；JSON-LD `json.Marshal`。
- **不要改** `vite.config.ts` / 路由 / 公开设置 schema / robots.txt。

## 已决策

- 静态模板使用生产站默认文案；运行时仍可由 `site_name` / `api_base_url` 覆盖头部标签。
- 简介放在 `#app` 外并默认隐藏，避免 Vue 异步 mount 前闪现。
- JSON-LD 走 `json.Marshal`：Go 会把 `<>&` 编成 `\u003c` 等，二次 HTML escape 会破坏 JSON。

## 待解决

- 无。

## 相关

- 代码：`frontend/index.html`、`backend/internal/web/embed_on.go`、`backend/internal/web/embed_test.go`
- 测试：`go test -tags=embed ./internal/web/`（必须先 `pnpm build` 刷新 `backend/internal/web/dist/`）
