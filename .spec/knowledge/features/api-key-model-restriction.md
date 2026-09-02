---
name: api-key-model-restriction
description: 用户 API 密钥的模型允许列表：创建/编辑密钥时从模型广场按分组选择可用模型，网关按原始请求模型拦截未授权请求
metadata:
  type: doc
  level: L2
  status: 已交付
---

# API 密钥模型限制

简介：用户创建或编辑 API 密钥时，可以开启“允许使用模型”限制。开启后，密钥只能请求勾选的模型；未开启或列表为空时保持不限制。

## 背景 / 目标

- 用户需要把单把密钥限制在指定模型集合内，避免密钥被用于更高成本或不该开放的模型。
- 可选模型必须来自当前分组在模型广场中的可用模型，避免用户填写不存在或不属于该分组的模型名。

## 设计

- **前端**：用户密钥弹窗读取公开模型广场接口 `/api/v1/model-market/public`，按当前选择的分组过滤模型，提供搜索和多选。切换分组时会裁剪已选模型到新分组可用范围内。
- **接口**：API key 创建/更新请求支持 `allowed_models`。空数组表示清空限制；更新请求未传该字段表示不修改已有限制。
- **存储**：`api_keys.allowed_models` 保存模型名数组；迁移默认值为 `[]`。响应 DTO 对空限制返回 `[]`，保持前端类型稳定。
- **网关**：请求进入模型映射和账号选择前，按客户端原始请求模型校验。未授权时返回 403，中文错误文案为 `当前密钥无权限使用模型：<model>`；模型为空时为 `当前密钥无权限使用该模型`。
- **模型列表**：`/v1/models`、`/antigravity/models` 和 Gemini 模型列表会按密钥允许列表过滤；空列表仍返回完整可用模型。

## 已决策

- 允许列表做精确模型名匹配；Gemini `models/<name>` 形式会兼容匹配无前缀名称。
- 校验发生在渠道/账号模型映射前，防止通过映射目标绕过密钥限制。
- 密钥级限制不改变模型广场公开范围；模型广场仍只负责展示公开可用模型。

## 待解决

- 私有/专属分组不在公开模型广场输出中，用户侧选择器因此可能显示暂无可选模型；若后续要支持私有分组选项，需要新增受登录态保护的模型候选接口。

## 相关

- 模型广场：[`model-market.md`](model-market.md)
- 前端：`frontend/src/views/user/KeysView.vue`
- 后端服务：`backend/internal/service/api_key.go`
- 网关拦截：`backend/internal/handler/gateway_api_key_model_permission.go`
- 数据迁移：`backend/migrations/157_add_api_key_allowed_models.sql`
