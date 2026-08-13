---
status: completed
title: 修复公开接口漏传 site_pages 导致服务条款/隐私不跳转
---

# 修复公开接口漏传 site_pages 导致服务条款/隐私不跳转

## 现象

- 管理员在「公开页面」给 `terms` / `privacy` 填了链接模式 URL 并保存成功。
- 首页「文档」能打开外链；「服务条款」「隐私保护」点击不跳到填写的 URL，只落到 `#footer`。
- 线上 `GET /api/v1/settings/public`：`doc_url` 有值，`site_pages` 为 `null`。

## 根因

1. `backend/internal/handler/setting_handler.go` 的 `GetPublicSettings` DTO 有 `SitePages` 字段，但从未赋值。管理端保存和 HTML 注入都有 `site_pages`，公开 API 没有。
2. 首页 / `AppHeader` 里「文档」可退回独立字段 `doc_url`（外链）；服务条款 / 隐私只能从 `site_pages` 解析，解析不到就 `#footer`。
3. `resolveSitePageNavigationTarget` 在 `mode=link` 时仍返回站内 `/doc/<slug>`，不把 `content` URL 当跳转目标。用户预期与「文档」一致：点了就打开填写的 URL。

## 做什么

最小修复，不重构 settings，不改 `release/*` / `publish`。

### 后端

- `GetPublicSettings` 增加 `SitePages: dto.ParseSitePages(settings.SitePages)`。
- `settings.SitePages` 已经过 `filterEnabledSitePages`，保持只暴露启用页。

### 前端导航

- `resolveSitePageNavigationTarget`：`mode=link` 且 `content` 是 http(s) URL 时返回 `{ kind: "external", target: url }`；markdown / 无 URL 仍走 `/doc/<slug>`。
- `HomeView.vue` / `AppHeader.vue`：`kind === "external"` 时按外链打开（`external: true` / `href`），与现有 `doc_url` 行为一致。
- 「文档」保底：site_pages 的 docs 若是空 markdown（默认种子），继续走 `doc_url`，禁止修完后把现网能用的文档外链打坏。
- 直接访问 `/doc/<slug>` 的 iframe 嵌入逻辑保持不变。

## TDD

1. 先写失败测试再实现。
2. 后端：`setting_handler_public_test.go` 增加公开接口暴露启用 `site_pages`（含 `mode`/`content`）、过滤 `enabled:false` 的回归测试。
3. 前端：改 `sitePages.spec.ts`——link 页解析为 external URL；补空 markdown docs 不抢 `doc_url` 的用例（若抽了 helper）。
4. 更新被行为变化影响的现有测试，不要删覆盖。

## 验收标准

- [x] `GET /api/v1/settings/public` 返回启用的 `site_pages` 数组（含 key/title/slug/mode/content），不是 `null`。
- [x] 禁用页不出现在公开 `site_pages`。
- [x] 链接模式的 terms/privacy 点击打开填写的 http(s) URL，而不是 `#footer` 或只进 `/doc/<slug>`。
- [x] 空 markdown 的 docs 仍走 `doc_url` 外链。
- [x] markdown 有正文的页面仍进 `/doc/<slug>`。
- [x] 后端相关 unit 测试通过；前端相关 vitest + `pnpm typecheck` 通过。
- [x] 不顺手重构无关代码，不 commit（主 loop 收口）。
