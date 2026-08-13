---
name: model-market
description: 公共模型广场页面与后台配置：按平台 / 分组 / 计费类型展示可用模型，自动模型默认读取当前计算价格并支持展示计费覆盖
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 公共模型广场

简介：`/models` 是面向访客和登录用户的模型广场页面，用来按供应商平台、公开分组和计费类型浏览模型。后台在「系统设置 → 模型广场」配置展示开关、自动同步 / 手动选择、标题说明、排序、候选模型展示计费覆盖和自定义模型；自动读取模型来自公开活跃分组下的可调度账号与分组模型路由，模型默认价格读取当前全局计算价格，分组倍率读取分组配置。管理员可把单个候选模型的展示计费模式覆盖为 token / per_request / image，并手动填写 token 输入 / 输出价、按次单价，或图片 1K / 2K / 4K 分档单价；未填写的图片分档不会在前台展示。该覆盖只影响模型广场展示，不改变网关真实计费。自定义模型可在模型广场配置中单独填写展示价格和倍率。

## 背景 / 目标

- 用户需要一个类似 `Go1c/new-api` 模型广场的可视化页面，快速查看不同平台 / 分组下有哪些模型可用。
- 首页旧「模型定价」为手填价目表，和真实计算价格容易漂移；模型广场改为只配置展示范围，自动模型价格使用当前 BillingService 全局价格。

## 设计

- **入口**：公共路由 `GET /models`，首页顶部导航跳转到该路由，不再跳转页面内 `#features`。
- **公开接口**：`GET /api/v1/model-market/public` 返回启用状态、页面标题说明和可公开展示模型。
- **后台接口**：`GET /api/v1/admin/settings/model-market` 返回当前配置、候选模型和最终展示模型；`PUT /api/v1/admin/settings/model-market` 保存配置。
- **后台配置**：配置存 `enabled`、`auto_sync`、`title`、`description`、手动模式下的 `selected_models` / `sort_order`，候选模型展示计费覆盖 `selected_models.billing_mode` / `selected_models.pricing`，以及用于补充展示的 `custom_models`。候选模型即使没有默认计算价格，管理员也可以创建 token 展示价格覆盖；图片计费分档复用 `pricing.intervals[].tier_label` + `per_request_price` 保存 `1K` / `2K` / `4K` 单价。
- **数据来源**：候选模型来自活跃公开分组下的可调度账号模型列表，以及分组模型路由中的具体模型名；通配符路由规则不展开为具体卡片。模型价格来自 `BillingService.GetModelPricing` 的全局当前计算价格，分组倍率来自分组配置。后台候选列表会显示专属分组中的模型但不带公开分组，方便管理员排查和选择；公开输出仍只展示非专属公开分组模型。
- **自定义模型**：管理员可新增自定义模型，选择平台、模型名、计费模式、分组、倍率和价格。token 价格在前端按每 1M tokens 输入，保存为后端每 token 价格；图片价格按可选的 1K / 2K / 4K 分档输入，空分档不保存也不展示；自定义模型不依赖自动候选，启用且有分组后会进入公开模型广场。前端保存前会拦截空模型名或启用但无分组的自定义项，避免无效配置被静默过滤。
- **公开范围**：自动读取模型只展示非专属公开分组；私有 / exclusive 分组不暴露给公开模型广场。自定义模型只展示管理员显式保存的分组快照。
- **交互**：左侧筛选供应商、分组、计费类型；主区域支持搜索、复制当前可见模型、卡片 / 表格视图切换。按量计费卡片第一行展示折扣倍率 × 订阅倍率（最低 0.75x），第二行右侧是「去订阅」入口，未登录跳转登录后进入购买页。
- **首页**：旧首页「模型定价」区块和 `/pricing/public` 公开接口已移除，首页导航保留「模型广场」入口。

## 已决策

- 新增独立模型广场接口，避免前台依赖登录用户可用渠道接口，也避免公开 private/exclusive 分组。
- 自动读取模型不复制价格配置：后台只决定展示哪些模型；价格由当前全局计算价格决定，倍率由分组配置决定。
- 候选模型允许配置展示计费覆盖：默认按 token 当前价格展示；管理员可手动填写 token 展示价，也可改成 per_request 或 image 并填写展示单价。image 使用可选的 1K / 2K / 4K 分档，未配置分档不会输出到公开页。覆盖只作用于模型广场公开输出，不参与 BillingService / 网关结算。
- 自定义模型用于补足自动读取不到的模型，因此允许在模型广场配置中保存展示价格和分组倍率快照。
- 不复用用户后台 `AvailableChannelsView` 的表格：模型广场是公开浏览页，交互重点是模型卡片和筛选，而不是渠道行表。

## 待解决

- 若后续需要展示供应商介绍、标签、端点类型等更细元数据，需要后端补充公共模型元数据接口。

## 相关

- 代码：`frontend/src/views/ModelMarketView.vue`
- 后台设置页：`frontend/src/views/admin/SettingsView.vue`
- 后端服务：`backend/internal/service/model_market.go`
- 后端 handler：`backend/internal/handler/model_market_handler.go`
- 路由：`frontend/src/router/index.ts`
- 首页导航：`frontend/src/views/HomeView.vue`
- 参考：`https://github.com/Go1c/new-api` 的模型广场形态
