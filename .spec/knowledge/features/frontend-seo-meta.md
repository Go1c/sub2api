---
name: frontend-seo-meta
description: 前端 GEO/SEO 头部元数据与爬虫可见简介——服务端注入、字段来源、降级与 JSON-LD 约定；改首页爬虫可见性或 injectSiteTitle 链路时查这篇
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 前端 GEO/SEO 头部元数据

简介：生产站 `GET /` 由 Go embed 直接吐 `index.html`。`injectSEOMeta` 在 `</head>` 前写入 description / OG / Twitter / JSON-LD；`#app` 内放品牌中立静态简介，给不执行 JS 的爬虫（GPTBot / ClaudeBot）看。

## 背景 / 目标

- 生产路径是 `backend/internal/web` 的 embed 注入，不是 Vite 开发注入。
- 静态 `index.html` 原先只有 title `Sub2API - AI API Gateway`，`#app` 为空，爬虫看不到简介。
- 品牌来自公开设置，禁止把 `LumioAPI` 写进可同步源码；默认品牌与静态 title 一致，用 `Sub2API`。

## 设计

- **注入链路**：`FrontendServer.injectSettings` 先写 `window.__APP_CONFIG__`（带 `__CSP_NONCE_VALUE__`），再 `injectSiteTitle` → `injectSiteFavicon` → `injectSEOMeta`。
- **字段来源**：公开设置 JSON 的 `site_name` / `site_subtitle` / `api_base_url`。`site_name` 空则 SEO 用 `Sub2API`，但仍注入 meta；`injectSiteTitle` 行为不变（空则保留静态 title）。
- **description**：有 subtitle 为 `{site_name} - {site_subtitle}。…`，无 subtitle 为 `{site_name}。…`。HTML 属性一律 `html.EscapeString`。
- **URL**：`api_base_url` 仅当 trim 后是带 host 的 `http`/`https` 才写 canonical / `og:url` / JSON-LD `url`；空或 `javascript:` 等非法值跳过这三项。
- **JSON-LD**：`json.Marshal` 一个 `@graph`（`Organization` + `WebSite`），禁止字符串拼接 JSON，也不要对 JSON 正文再做 `html.EscapeString`。`<script type="application/ld+json">` 带 CSP nonce placeholder。
- **爬虫简介**：`frontend/index.html` 的 `#app` 内静态 `Sub2API` + `/login` + `/register`，不要后端替换、不要加 CSS。Vue mount 后自然替换。
- **不要改** `vite.config.ts` / 公开设置 schema / robots.txt。

## 已决策

- 沿用 `injectSiteTitle` 的服务端注入，不另开 Vite 生产插件。
- 默认品牌 `Sub2API`，与静态 title 对齐，避免 fork 品牌进 upstream 可同步文件。
- JSON-LD 走 `json.Marshal`：Go 会把 `<>&` 编成 `\u003c` 等，二次 HTML escape 会破坏 JSON。

## 待解决

- 无。

## 相关

- 代码：`backend/internal/web/embed_on.go`、`backend/internal/web/embed_test.go`、`frontend/index.html`
- 测试：`go test -tags=embed ./internal/web/`（必须先 `pnpm build` 刷新 `backend/internal/web/dist/`）
