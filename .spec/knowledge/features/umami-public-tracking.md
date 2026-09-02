---
name: umami-public-tracking
description: 前台 Umami 官方脚本埋点：SPA 条件插入、/admin 首屏跳过、docs 静态页直接埋、CSP script-src 放行
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 前台 Umami 公共页跟踪

简介：在公开前台加载自托管 Umami（`https://data.lumio.games/script.js`），管理端直开 `/admin` 不加载该脚本。

## 背景 / 目标

需要官方 `defer` 脚本统计前台访问，但不能污染管理端。主站是同一套 Vue SPA：`/` 与 `/admin` 共用 `frontend/index.html`，因此不能把官方 `<script defer src=...>` 无条件写进 SPA 入口。

## 设计

### 实现面

- **SPA**（`frontend/index.html`）：用带 `nonce="__CSP_NONCE_VALUE__"` 的 inline 读首屏 `location.pathname`。`/admin` 或 `/admin/...` 直接返回；否则 `document.createElement('script')`，`defer=true`，`src=https://data.lumio.games/script.js`，`data-website-id=423c8276-e57c-4e5b-81e6-63711a7fd1a5`。后端 `replaceNoncePlaceholder` 会把占位符换成请求级 nonce。不在 `main.ts` / router 里埋，避免 admin 包一并执行。
- **文档静态页**（`frontend/public/docs/index.html`）：独立 HTML，不经 SPA，直接写官方 `<script defer src=... data-website-id=...>`，无 inline。该文件走静态资源路径，不会做 nonce 替换。
- **CSP**：`script-src` 必须放行 `https://data.lumio.games`。三处同步：
  - `backend/internal/config/config.go` `DefaultCSPPolicy`
  - `backend/internal/server/middleware/security_headers.go` `requiredCSPDirectiveValues`（常量 `UmamiDomain`）
  - `security_headers_test.go`
- `connect-src` 已是 `https:`，Umami 上报不必再开。

### 不跟踪的范围

- 浏览器地址栏直开 `/admin` 或 `/admin/*` 的首屏。
- 不为 SPA 内后续跳转卸载已加载的脚本。

## 已决策

- 用官方 script + `defer`，不用 npm SDK / 自定义 pixel。
- SPA 用 nonce inline 条件插入；docs 静态页用官方标签。
- 不改 llms.txt、服务地区口径、#321–#324 相关正文。

## 待解决

- 无。

## 相关

- CSP nonce：`backend/internal/web/embed_on.go` `NonceHTMLPlaceholder`
- 测试：`frontend/src/__tests__/umami-public-tracking.spec.ts`、`backend/internal/server/middleware/security_headers_test.go`
