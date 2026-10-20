---
name: api-key-client-import
description: API 密钥页优先用 BestCodex 深链接导入供应商，并保留 CC Switch；改导入按钮或协议参数时查
metadata:
  type: doc
  level: L2
  status: 已交付
---

# API 密钥导入到 BestCodex

简介：密钥列表仍显示原来的「导入到 CCS」按钮。点击后优先打开 BestCodex，弹层里引导打开或下载 BestCodex；CC Switch 只在用户再点一次时启动。

## 背景 / 目标

- 桌面客户端已提供 `bestcodex://`，网站不再把「导入到 CCS」当作唯一入口。
- 原版 CC Switch 不认识本站自定义 `ticket` / `configUrl`。两个客户端共用它已支持的供应商参数。
- 浏览器不能判断软件是否安装。页面不宣称已导入或已检测到未安装，也不用定时器改开另一个客户端。

## 设计

- **入口**：行内按钮保持原来的「导入到 CCS」样式，不改成 BestCodex，也不加下拉。点击后取当前用户有权读取的完整 Key（已有 `GET /api/v1/keys/:id`），再 `location.assign` 到 `bestcodex://v1/import`。
- **弹层**：第二步弹层的主按钮是「打开 BestCodex」。「未打开？使用 CC Switch」是这一次的兜底链接，不改默认客户端，也不写 `localStorage`。最下方说明 BestCodex 会自动导入、自动配置、自动下载，并且包含 CC Switch 的功能，链接到 `CODEX_DOWNLOAD_URL`（`https://bestcodex.app/`）。
- **被拦截时**：异步取 Key 之后如果导航被拦住，弹层里的「打开 BestCodex」是真实 `<a href>`，下一次点击才启动。关闭弹层时先去掉链接，再关弹层。不在加载、轮询、`blur` 或 `visibilitychange` 里启动应用。
- **参数**：`resource=provider`，`app` 按分组选择 `codex` / `claude` / `gemini`（Antigravity 仍让用户选 Claude 或 Gemini）。`name` 用密钥名和分组名。Codex 的 endpoint 规范成恰好一个 `/v1`（Lumio Responses）。Claude / Gemini 用站点已发布的 API 基址，不再追加一层 `/v1`。Antigravity 仍是去掉末尾 `/v1` 后的 `/antigravity`。`homepage` 固定 `https://api.lumio.games/`。不带 `model`、`enabled`、`config`、`usageScript`、`configUrl`、`ticket`。
- **编码**：参数只经 `URLSearchParams` 编码一次。链接超过 8000 字符时拒绝并提示手动导入。Key 与完整链接不进 localStorage、sessionStorage、剪贴板、埋点或错误文案。
- **复制配置**：未安装客户端时，原有「使用密钥」弹层和下载入口仍在。管理员「隐藏 CCS 导入按钮」继续隐藏整组导入控件。

## 已决策

- 不新增明文 Key 接口，点击时走现有按 ID 读取。
- 不改 BestCodex 客户端，也不改旧的 `buildCcswitchImportUrl` 工具；密钥页不再调用它，因此这次导入不再附带默认模型或用量脚本。
- 管理端 CCSwitch 默认模型仍只服务桌面公开配置，不进入这条深链接。

## 待解决

- 无。浏览器是否弹出「打开应用」由浏览器决定，网站不能去掉。

## 相关

- 协议函数：`frontend/src/utils/providerImport.ts`
- 按钮：`frontend/src/components/keys/KeyImportMenu.vue`，用在 `frontend/src/views/user/KeysView.vue`
- 下载地址：[[codex-download-nav]]
- 上游参数：CC Switch v4.0.15 deeplink guide（`resource=provider`）
