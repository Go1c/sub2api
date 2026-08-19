---
name: daily-checkin
description: 独立每日签到模块，包含原子发奖、预算控制、用户与管理员界面
status: completed
---

# 每日签到独立模块

- [x] 独立迁移创建配置、流水和每日预算表，默认关闭。
- [x] 后端独立模块完成原子签到、重复请求、预算控制、缓存失效和用户/管理员 API。
- [x] 前端独立模块完成 Store、用户页、管理流水、设置卡、侧栏与中英文文案。
- [x] 后端 unit/integration/vet/lint 与前端 Vitest/typecheck/build 完成或记录环境阻塞。
- [x] 桌面和 375px 布局完成浏览器验收。
- [x] 功能知识文档及索引已同步。

## 验证记录

- `go test -tags=unit ./internal/checkin ./migrations`、`go vet -tags integration ./...` 通过。
- `go test -tags=integration -v ./internal/checkin` 在当前机器因 Docker 不可用而由 `TestMain` 明确跳过。
- `/Users/cui/go/bin/golangci-lint run ./internal/checkin/... ./migrations/...` 通过（`0 issues`）。完整 lint 仅报告两个既有测试文件的 6 条 `SA5011`：`internal/service/account_usage_service_test.go`、`internal/service/ops_service_user_error_test.go`。
- 完整 `go test -tags=unit ./...` 仅有既有 Settings API 快照差异：`ccswitch_default_model_openai` 预期空值，实际为 `gpt-5.4`。
- 前端签到 Vitest 8 个文件、21 个测试通过；`pnpm typecheck`、`pnpm build`、`pnpm lint:check` 通过。
- 浏览器已验收 1440x900 桌面布局和 375x811 移动布局，用户页、管理员流水、设置卡及返利关闭时的导航门控均无溢出或遮挡。
