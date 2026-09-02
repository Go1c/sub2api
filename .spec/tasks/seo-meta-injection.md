---
status: completed
---

# 前端 GEO/SEO 头部元数据与爬虫可见简介

## 目标

生产站爬虫（含不执行 JS 的 GPTBot/ClaudeBot）抓 `GET /` 时，能看到 description / OG / Twitter / JSON-LD，以及 `#app` 内的静态简介。沿用 `injectSiteTitle` 的服务端注入模式。

## 分支

当前 worktree 在无关分支 `feat/openai-hidden-luna-autoreview`。动手前必须：

```bash
git fetch origin
git checkout -b feat/seo-meta-injection origin/dev
```

不得把改动提交到 luna 分支；不得直推 `main` / `publish`。不要提交 `backend/ent/schema/.entc/`。

## 范围（最小 diff）

只改这些文件（可按实现需要微调，但不要顺手重构）：

- `frontend/index.html` — `#app` 内放品牌中立静态简介
- `backend/internal/web/embed_on.go` — 新增 `injectSEOMeta`，在 `injectSettings` 里调用
- `backend/internal/web/embed_test.go` — 补注入单测
- `.spec/knowledge/features/frontend-seo-meta.md` + `.spec/knowledge/README.md` — spec-steward 沉淀

不要改 `vite.config.ts` / `vite.config.js`（生产路径是 Go embed；Vite 仅本地 `pnpm dev`）。不要硬编码 `LumioAPI`。

## 实现约定

### 1. `injectSEOMeta(html, settingsJSON []byte) []byte`

在 `</head>` 前注入。从 settings JSON 读 `site_name` / `site_subtitle` / `api_base_url`。

- `site_name` 空 → 用静态默认 `Sub2API`（与 `index.html` `<title>` 一致），仍注入 meta；JSON 非法 / 无 `</head>` → 原样返回。
- description（中文为主，属性值一律 `html.EscapeString`）：
  - 有 subtitle：`{site_name} - {site_subtitle}。AI API 中转与接口管理平台,统一接入 Claude、GPT、Gemini 等主流模型,支持用量统计与额度管理。`
  - 无 subtitle：`{site_name}。AI API 中转与接口管理平台,统一接入 Claude、GPT、Gemini 等主流模型,支持用量统计与额度管理。`
- `api_base_url` 仅当 trim 后是带 host 的 `http`/`https` 才注入：
  - `<link rel="canonical" href="...">`
  - `og:url`
  - JSON-LD 的 `url`
  - 非法/空则跳过这三项，其余仍注入
- 始终尝试注入（在有合法 JSON 且存在 `</head>` 时）：
  - `<meta name="description" content="...">`
  - `og:type=website`、`og:site_name`、`og:title`（`{site_name} - AI API Gateway`）、`og:description`
  - `twitter:card=summary`
  - JSON-LD：`Organization(name, url?)` + `WebSite(name, url?)`
- JSON-LD **必须** `json.Marshal` 后嵌入，禁止字符串拼接 JSON。Go `json.Marshal` 会把 `<>&` 编成 `\u003c` 等，**不要**再对 JSON 正文做 `html.EscapeString`（会破坏 JSON）。JSON-LD `<script type="application/ld+json">` 建议带 `nonce="__CSP_NONCE_VALUE__"`，与现有 `__APP_CONFIG__` 脚本一致。
- 建议用 `@graph` 一个 script 同时放 Organization 和 WebSite。
- **不要**改变 `injectSiteTitle`：title 仍是 `{site_name} - AI API Gateway`，`site_name` 空则保留静态 title。

在 `injectSettings` 末尾调用：`injectSiteTitle` → `injectSiteFavicon` → `injectSEOMeta`。

### 2. 爬虫可见简介（upstream 同步负担最小）

在 `frontend/index.html` 的 `<div id="app">` 内放静态占位（不要占位符、不要后端替换）：

```html
<div id="app">
  <main>
    <h1>Sub2API</h1>
    <p>AI API 中转与接口管理平台。统一接入 Claude、GPT、Gemini 等主流模型，支持用量统计与额度管理。</p>
    <p><a href="/login">登录</a> · <a href="/register">注册</a></p>
  </main>
</div>
```

品牌中立用 `Sub2API`（已与静态 title 一致）。Vue mount 后自然替换。不要加 CSS。

## TDD

先写失败测试再实现。`embed_test.go` 已有 `//go:build embed`，跑：

```bash
cd backend && go test -tags=embed ./internal/web/ -count=1
```

至少覆盖：

- 完整配置：description / canonical / og:* / twitter:card / JSON-LD 均出现；title 仍为 `{site_name} - AI API Gateway`
- `site_name` 空：用 `Sub2API` 默认注入 description，不改 title（`injectSiteTitle` 行为不回归）
- `api_base_url` 空或 `javascript:...`：无 canonical / og:url
- `site_name` / subtitle 含 `</title><script>`、`&`：属性被 escape，HTML 中无裸 `<script>`
- JSON-LD 可用 `json.Unmarshal` 解析；含 Organization + WebSite；值为配置字段而非硬编码品牌
- 非法 JSON / 无 `</head>`：原样返回
- `injectSettings` 仍注入 `__APP_CONFIG__` + nonce placeholder，并带上 SEO 块

`injectSiteTitle` 现有用例不得改语义。

## 验收（必须全部跑完，把命令输出要点写进交付）

```bash
cd frontend && pnpm typecheck && pnpm build
cd backend && go vet ./...
cd backend && go vet -tags integration ./...
cd backend && go test -tags=embed ./internal/web/ -count=1
cd backend && go test -tags=unit ./internal/web/ -count=1
```

`pnpm build` 会刷新 `backend/internal/web/dist/`。`go test -tags=embed` 必须在 build 之后跑（embed 读的是 dist）。

curl 验收：若本机能起 embed 服务（`go build -tags embed` + 已有 Postgres/Redis），`curl -s localhost:8080/` 必须看到：

- `meta name="description"`
- `rel="canonical"`（当配置了 api_base_url）
- `application/ld+json`
- `#app` 内静态简介（Sub2API / 登录 / 注册）
- `<title>` 为 `{site_name} - AI API Gateway`（有配置时）

若完整进程起不来：写一个 `TestInjectSEOMeta`/`injectSettings` 用例，用接近真实的 `index.html` 片段断言上述片段都在，并在交付里说明未能起 8080 的原因。不要假装 curl 过。

## 知识沉淀

用 spec-steward 新增 `.spec/knowledge/features/frontend-seo-meta.md`（照 `_TEMPLATE.md`），并在 `knowledge/README.md` 加一行。记：服务端注入、字段来源、降级、禁止硬编码品牌、JSON-LD 用 Marshal。

## 提交

验证通过后再 commit。格式：`feat(seo): inject meta tags and crawler-visible intro`

不要 `git push` 到 `main`/`publish`。可以 push `feat/seo-meta-injection` 到 origin。不要自己 `gh pr create`（主 loop 收口开 PR）。

## 非目标

- 不改 robots.txt / sitemap / llms.txt（运维仓库）
- 不改 Vite 开发注入
- 不改公开设置 schema
- 不把 LumioAPI 写进可同步源码
