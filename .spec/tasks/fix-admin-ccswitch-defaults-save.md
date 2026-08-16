---
status: completed
title: 修复管理员设置 CCSwitch 默认模型保存后读回为空
---

# 修复管理员设置 CCSwitch 默认模型保存后读回为空

## 现象

管理员后台「站点」页的 CCSwitch 默认模型（OpenAI / Claude / Gemini / Antigravity Claude / Antigravity Gemini）保存后仍显示占位符（留空默认）。API 端点地址能回显，说明设置页本身能加载；模型输入保存后被立刻清空。

## 根因

`GET /api/v1/admin/settings` 与 `PUT` 成功响应都走 `SettingService.GetAllSettings` → `parseSettings`。

`parseSettings`（`backend/internal/service/setting_parse.go`）从未把这 5 个 key 填进 `SystemSettings`：

- `ccswitch_default_model_anthropic`
- `ccswitch_default_model_openai`
- `ccswitch_default_model_gemini`
- `ccswitch_default_model_antigravity`
- `ccswitch_default_model_antigravity_gemini`

写入链路是通的（`setting_update.go` 会 `SetMultiple`）。公开设置也是通的（`setting_public.go` 直接读 raw map）。

管理员前端 `SettingsView.vue` 在保存成功后用响应体 `Object.assign` 回填表单；响应里这些字段是空字符串（非 null），于是把用户刚填的值覆盖成空。看起来像「点保存没反应 / 一直是留空默认」。

这是上游同步后 fork 字段只补了 defaults + DTO + write、漏了 parse 的缺口（同类：`9eb849924`）。

## 做什么

最小修复：在 `parseSettings` 读回这 5 个字段。OpenAI 空值回退 `gpt-5.4`，与 write / public 一致。不要改前端，不要重构 settings。

## TDD 要求

1. 先写失败测试：仓储里已有这 5 个值时，`GetAllSettings` 必须原样返回（OpenAI 空值回退 `gpt-5.4`）。确认红灯是字段为空，不是测错了。
2. 最小改 `parseSettings` 让测试转绿。
3. 需要的话补 Update → GetAll 往返，证明保存后读回不再被空串覆盖。

可复用 `settingGetAllRepoStub` / `settingUpdateRepoStub`（`setting_service_update_test.go`）。测试文件必须 `//go:build unit`。

## 涉及范围

- `backend/internal/service/setting_parse.go`
- `backend/internal/service/*_test.go`（unit）

## 验收标准

- [x] `GetAllSettings` 对已存储的 5 个 CCSwitch 默认模型返回非空值
- [x] OpenAI 缺省 / 空串回退 `gpt-5.4`；其余模型允许空串（语义：留空则不传 model）
- [x] Update 写入后再 GetAll，读回值与写入一致
- [x] 管理员 GET/PUT 响应能带上这些字段，前端回填后输入框不再被清空
- [x] `cd backend && go test -tags=unit ./internal/service/` 通过
- [x] 不改 `frontend/`，不顺手重构 settings

## 验证记录

- TDD 红灯：仓储预置 5 个非空值后 `GetAllSettings` 全是空串；OpenAI 缺省 / 空串也是空串而不是 `gpt-5.4`；Update 写入成功但 GetAll 仍空。失败断言均指向 `CCSwitchDefaultModel*` 为零值。
- 绿灯：`cd /Users/cui/Sites/sub2api/backend && go test -tags=unit ./internal/service/ -count=1` 通过。
- 未跑：全量 `go test -tags=unit ./...`、integration、`go vet -tags integration`、golangci-lint。
- 未做浏览器 / HTTP 联调；管理员 handler 已从 `GetAllSettings` 映射这 5 个字段，service 层读回修复后响应不再回填空串。
- 未改 `frontend/`。

## 依赖

无
