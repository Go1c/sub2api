---
name: codex-download-nav
description: 首页与登录后顶栏的 Codex 下载外链，跳转到 bestcodex.app；改导航入口或下载引导时查
metadata:
  type: doc
  level: L2
  status: 已交付
---

# Codex 下载导航入口

简介：公开首页和登录后顶栏在「模型广场」旁放「Codex 下载」，新标签打开 `https://bestcodex.app/`，引导用户去该站下载 Codex 客户端。这是固定外链，不是站内落地页，也不是 Lumio 自有桌面客户端契约。

## 背景 / 目标

- 竞品在登录后顶栏把「Codex 下载」和「模型广场」并列，降低找客户端的成本。
- 用户应能从公开首页和登录后控制台两处到达同一下载地址。

## 设计

- **目标 URL**：`CODEX_DOWNLOAD_URL`（`frontend/src/constants/codexDownload.ts`）固定为 `https://bestcodex.app/`。
- **文案**：中文 `Codex 下载`，英文 `Download Codex`。主入口，不 `dim`。
- **公开首页**：`HomeView` 的 `navItems` 插在模型广场之后、状态之前；`external: true` 走现有 `onNav()` 的 `window.open(..., 'noopener,noreferrer')`。桌面胶囊和移动菜单共用这份列表。
- **登录后顶栏**：`AppHeader` 的 `homeNavItems` 同样插入外链 `<a target="_blank" rel="noopener noreferrer">`。`xl` 以下中间导航隐藏，因此右侧再放一条 `xl:hidden` 的紧凑链（下载图标 + 文案），与文档链同一模式。
- **打开方式**：一律新标签；不走 `/login?handoff=`，不加入 external auth 白名单。

## 已决策

- 硬编码 URL，不做管理员设置，避免改 `backend/`。
- 不接入 `lumio_desktop_config` / 桌面支付交接；那是 [[lumio-desktop-client]] 的范围。
- 公开 `/model-market` 自有简顶栏本轮不加这条链。
- 不进侧栏、仪表盘快捷操作或独立 `/codex` 介绍页。

## 待解决

- 无。

## 相关

- 常量：`frontend/src/constants/codexDownload.ts`
- 首页：`frontend/src/views/HomeView.vue`
- 登录后顶栏：`frontend/src/components/layout/AppHeader.vue`
- 测试：`frontend/src/views/__tests__/HomeViewNav.spec.ts`、`frontend/src/components/layout/__tests__/AppHeader.spec.ts`
- 桌面客户端契约：[[lumio-desktop-client]]
- 外部登录回跳（本功能不使用）：[[external-auth-handoff]]
