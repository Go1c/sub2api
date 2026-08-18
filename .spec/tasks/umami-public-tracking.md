---
status: completed
title: 前台埋 Umami 官方脚本，/admin 直开不加载
---

# 前台埋 Umami（官方 script defer）

## 做什么

在 **origin/dev 已切出的** `feat/umami-public-tracking` 上，给前台埋 Umami 官方脚本，并同步 CSP。PR 只打 `dev`，不 merge、不推 `publish`。

跟踪参数（不得改）：

- `src=https://data.lumio.games/script.js`
- `data-website-id=423c8276-e57c-4e5b-81e6-63711a7fd1a5`
- **必须官方 script + defer**（`<script defer src="..." data-website-id="..."></script>` 或其等价动态插入：`script.defer = true` + 同一 src / data-website-id）

## 约束（硬）

1. 只埋前台。同一套 Vue SPA，**不能**把官方脚本无条件写进 `frontend/index.html`。
2. 直开 `/admin` 或 `/admin/...` **不得**加载 `script.js`。判定用首屏 `location.pathname`（`=== '/admin'` 或 `startsWith('/admin/')`）。不要为 SPA 内后续跳转再卸脚本。
3. `frontend/index.html` 若用 inline 条件插入官方脚本，该 inline **必须** `nonce="__CSP_NONCE_VALUE__"`（后端 `replaceNoncePlaceholder` 会替换；生产 CSP 禁无 nonce 的 inline）。
4. `frontend/public/docs/index.html`（用户说的 docs/index.html）**直接**加官方 `<script defer src=... data-website-id=...>`，**无 inline**。
5. 必须同步 CSP `script-src` 放行 `https://data.lumio.games`：
   - `backend/internal/config/config.go` 的 `DefaultCSPPolicy`
   - `backend/internal/server/middleware/security_headers.go` 的 `requiredCSPDirectiveValues`（仿 CloudflareInsightsDomain，加命名常量）
   - `backend/internal/server/middleware/security_headers_test.go`
6. **不要改**：计费/用量口径、`frontend/public/llms.txt`、#321–#324 相关正文（含 docs 页「不向中国大陆地区用户提供」脚注、登录条款）、DNS、密码。docs 页只加脚本，不动正文。
7. **不要提交** `.run-grok.sh`、`GEO-SEO-TASK.md`。不要改 `release/*` / `publish`。
8. 不要把跟踪写进 Vue `main.ts` / router（admin 也会加载同一 bundle）。

## TDD

先写失败测试，再实现。

### 后端

- `DefaultCSPPolicy` 的 `script-src` 含 `https://data.lumio.games`。
- `enhanceCSPPolicy` 对缺该域名的旧策略会补上 `script-src`，已有则不重复（对齐 Cloudflare / Airwallex 测法）。
- 命名常量（如 `UmamiDomain`）出现在 `requiredCSPDirectiveValues` 的 `script-src`。

### 前端

vitest include 是 `src/**/*.{test,spec}.{js,ts}`。新建例如 `frontend/src/__tests__/umami-public-tracking.spec.ts`，用 `fs` 读仓库内 HTML（相对 `import.meta.url`）：

- `frontend/index.html`：存在 `nonce="__CSP_NONCE_VALUE__"` 的 inline；inline 会插入官方 `https://data.lumio.games/script.js` + 该 `data-website-id` + defer；**没有**无条件的 `<script defer src="https://data.lumio.games/script.js" ...>`；含 `/admin` 直开跳过逻辑。
- 可选但推荐：jsdom 执行抽取出的 inline，pathname=`/admin` 与 `/admin/dashboard` 不插入 script；`/`、`/home`、`/login`、`/docs/` 会插入一条官方 script。
- `frontend/public/docs/index.html`：有且仅用官方 `<script defer src="https://data.lumio.games/script.js" data-website-id="423c8276-e57c-4e5b-81e6-63711a7fd1a5">`；无 umami/data.lumio 相关 inline；正文/脚注相对 origin/dev 无改动。

## 实现提示

`frontend/index.html` 建议在 `</head>` 前插入（保持现有 SEO 块不动）：

```html
<script nonce="__CSP_NONCE_VALUE__">
(function () {
  var p = location.pathname;
  if (p === '/admin' || p.indexOf('/admin/') === 0) return;
  var s = document.createElement('script');
  s.defer = true;
  s.src = 'https://data.lumio.games/script.js';
  s.setAttribute('data-website-id', '423c8276-e57c-4e5b-81e6-63711a7fd1a5');
  document.head.appendChild(s);
})();
</script>
```

`connect-src` 已是 `https:`，不必为 Umami 再开。只改 `script-src`。

## 知识沉淀

用 `spec-steward` 新增 `knowledge/features/umami-public-tracking.md`（照 `_TEMPLATE.md`），并在 `knowledge/README.md` 加一行。写清：为何不能无条件进 SPA `index.html`、admin 首屏跳过、docs 静态页直接官方脚本、CSP 三处必须同步、nonce 占位符。

## 验收

- [ ] 上述 TDD 全绿
- [ ] `cd frontend && pnpm test:run`（至少新测）通过
- [ ] `cd frontend && pnpm typecheck && pnpm build` 通过
- [ ] `cd backend && go test ./internal/server/middleware/ ./internal/config/ -count=1` 通过
- [ ] `cd backend && go vet -tags integration ./internal/server/middleware/` 通过
- [ ] 未改 llms.txt / 321-324 正文 / 口径 / DNS / 密码
- [ ] 知识文档 + README 导航已更新
- [ ] **不要 commit / push / 开 PR**（主 loop 收口）

## 非目标

- 不 merge、不推 publish、不开 `--base publish`
- 不改计费、鉴权、API、数据库、部署
- 不为 SPA 客户端路由卸载已加载的 Umami
