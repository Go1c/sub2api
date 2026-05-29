# Subscription Admin Issue 47 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 issue #47：撤销订阅不进浪费统计，订阅限额刷新时间可配置，订阅套餐入口移动到订阅管理。

**Architecture:** 后端把订阅刷新时间作为系统设置写入 `UsageBillingCommand`，repository 只根据 command 计算窗口。浪费统计在公共 SQL where helper 中排除 `revoked`。前端只扩展已有设置页、路由和侧边栏，不重命名支付套餐 API。

**Tech Stack:** Go 1.26、Gin、Ent/raw SQL、Vitest/Vue 3、Vue Router、pnpm、PowerShell。

---

## 文件结构

- `backend/internal/service/domain_constants.go`：新增 `SubscriptionStatusRevoked` 和两个设置 key。
- `backend/internal/service/settings_view.go`：给 `SystemSettings` 增加订阅刷新配置字段。
- `backend/internal/service/usage_billing.go`：新增 `SubscriptionQuotaResetConfig`、规范化 helper，并把配置放入 `UsageBillingCommand`。
- `backend/internal/service/setting_service.go`：默认设置、解析、保存、读取订阅刷新配置。
- `backend/internal/service/setting_service_update_test.go`：设置保存和读取的红绿测试。
- `backend/internal/service/gateway_service.go`：构建扣费 command 时带上订阅刷新配置。
- `backend/internal/repository/usage_billing_repo.go`：按配置计算日/周窗口。
- `backend/internal/repository/usage_billing_repo_test.go`：窗口计算红绿测试。
- `backend/internal/repository/subscription_waste_stats_repo.go`：公共 where 排除 revoked。
- `backend/internal/repository/subscription_waste_stats_repo_test.go`：SQL where 红绿测试。
- `backend/internal/handler/dto/settings.go`：管理员设置响应 DTO 增加字段。
- `backend/internal/handler/admin/setting_handler.go`：管理员设置请求、响应、diff 增加字段。
- `frontend/src/api/admin/settings.ts`：前端设置类型增加字段。
- `frontend/src/views/admin/SettingsView.vue`：用户订阅入口卡片增加 UTC 偏移和刷新小时输入。
- `frontend/src/router/index.ts`：新增 `/admin/subscriptions/plans`，旧 `/admin/orders/plans` 重定向。
- `frontend/src/components/layout/AppSidebar.vue`：订阅套餐菜单移动到订阅管理。
- `frontend/src/i18n/locales/{zh,en,zh-Hant}.ts`：新增设置页文案。

## Task 1: 浪费统计排除 revoked

**Files:**
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/repository/subscription_waste_stats_repo.go`
- Test: `backend/internal/repository/subscription_waste_stats_repo_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/internal/repository/subscription_waste_stats_repo_test.go` 添加：

```go
package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWasteStatsWhere_ExcludesRevokedSubscriptions(t *testing.T) {
	where, args := wasteStatsWhere(service.WasteStatsQuery{
		StartTime: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})

	require.Contains(t, where, "us.status <> $3")
	require.Len(t, args, 3)
	require.Equal(t, service.SubscriptionStatusRevoked, args[2])
}
```

- [ ] **Step 2: 验证红灯**

Run: `go test -tags unit ./internal/repository -run TestWasteStatsWhere_ExcludesRevokedSubscriptions -count=1`
Expected: FAIL，where 不包含 `us.status <> $3`。

- [ ] **Step 3: 最小实现**

在 `domain_constants.go` 订阅状态常量中加：

```go
SubscriptionStatusRevoked = "revoked"
```

在 `wasteStatsWhere()` 初始化 args/parts 时加 revoked 参数：

```go
args := []any{query.StartTime, query.EndTime, service.SubscriptionStatusRevoked}
parts := []string{"l.created_at >= $1", "l.created_at < $2", "us.status <> $3"}
```

- [ ] **Step 4: 验证绿灯**

Run: `go test -tags unit ./internal/repository -run TestWasteStatsWhere_ExcludesRevokedSubscriptions -count=1`
Expected: PASS。

## Task 2: 订阅刷新配置和窗口计算

**Files:**
- Modify: `backend/internal/service/usage_billing.go`
- Modify: `backend/internal/repository/usage_billing_repo.go`
- Test: `backend/internal/repository/usage_billing_repo_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/internal/repository/usage_billing_repo_test.go` 添加默认、UTC+08 日窗口、UTC+08 周窗口测试：

```go
package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionQuotaWindowStarts_DefaultMatchesUTC(t *testing.T) {
	now := time.Date(2026, 5, 29, 15, 30, 0, 0, time.UTC)
	cfg := service.SubscriptionQuotaResetConfig{}

	daily, weekly := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC), daily)
	require.Equal(t, time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), weekly)
}

func TestSubscriptionQuotaWindowStarts_UsesUTCOffsetAndResetHour(t *testing.T) {
	now := time.Date(2026, 5, 28, 19, 30, 0, 0, time.UTC) // UTC+08 local 03:30, before 04:00 reset
	cfg := service.SubscriptionQuotaResetConfig{UTCOffsetMinutes: 8 * 60, ResetHour: 4}

	daily, _ := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 27, 20, 0, 0, 0, time.UTC), daily)
}

func TestSubscriptionQuotaWindowStarts_WeeklyUsesMondayInConfiguredOffset(t *testing.T) {
	now := time.Date(2026, 5, 31, 23, 30, 0, 0, time.UTC) // UTC+08 Monday 07:30
	cfg := service.SubscriptionQuotaResetConfig{UTCOffsetMinutes: 8 * 60, ResetHour: 4}

	_, weekly := subscriptionQuotaWindowStarts(now, cfg)

	require.Equal(t, time.Date(2026, 5, 31, 20, 0, 0, 0, time.UTC), weekly)
}
```

- [ ] **Step 2: 验证红灯**

Run: `go test -tags unit ./internal/repository -run TestSubscriptionQuotaWindowStarts -count=1`
Expected: FAIL，`SubscriptionQuotaResetConfig` 或 `subscriptionQuotaWindowStarts` 不存在。

- [ ] **Step 3: 最小实现**

在 `usage_billing.go` 增加配置结构、默认常量、规范化函数，并给 `UsageBillingCommand` 增加字段：

```go
type SubscriptionQuotaResetConfig struct {
	UTCOffsetMinutes int
	ResetHour        int
}

const (
	SubscriptionQuotaResetMinOffsetMinutes = -12 * 60
	SubscriptionQuotaResetMaxOffsetMinutes = 14 * 60
)

func NormalizeSubscriptionQuotaResetConfig(cfg SubscriptionQuotaResetConfig) SubscriptionQuotaResetConfig {
	if cfg.UTCOffsetMinutes < SubscriptionQuotaResetMinOffsetMinutes || cfg.UTCOffsetMinutes > SubscriptionQuotaResetMaxOffsetMinutes || cfg.UTCOffsetMinutes%15 != 0 {
		cfg.UTCOffsetMinutes = 0
	}
	if cfg.ResetHour < 0 || cfg.ResetHour > 23 {
		cfg.ResetHour = 0
	}
	return cfg
}
```

在 `usage_billing_repo.go` 增加：

```go
func subscriptionQuotaWindowStarts(t time.Time, cfg service.SubscriptionQuotaResetConfig) (time.Time, time.Time) {
	cfg = service.NormalizeSubscriptionQuotaResetConfig(cfg)
	offset := time.Duration(cfg.UTCOffsetMinutes) * time.Minute
	local := t.UTC().Add(offset)
	dailyLocal := time.Date(local.Year(), local.Month(), local.Day(), cfg.ResetHour, 0, 0, 0, time.UTC)
	if local.Before(dailyLocal) {
		dailyLocal = dailyLocal.AddDate(0, 0, -1)
	}
	weekday := int(dailyLocal.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weeklyLocal := time.Date(dailyLocal.Year(), dailyLocal.Month(), dailyLocal.Day(), cfg.ResetHour, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
	return dailyLocal.Add(-offset), weeklyLocal.Add(-offset)
}
```

在 `applyUsageBillingSubscription()` 中用 `cmd.SubscriptionQuotaResetConfig` 调用新 helper。

- [ ] **Step 4: 验证绿灯**

Run: `go test -tags unit ./internal/repository -run TestSubscriptionQuotaWindowStarts -count=1`
Expected: PASS。

## Task 3: 设置服务保存和读取订阅刷新配置

**Files:**
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/service/setting_service.go`
- Test: `backend/internal/service/setting_service_update_test.go`

- [ ] **Step 1: 写失败测试**

在 `setting_service_update_test.go` 增加：

```go
func TestSettingService_UpdateSettings_SubscriptionQuotaResetSettings(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SubscriptionQuotaResetUTCOffsetMinutes: 480,
		SubscriptionQuotaResetHour:             4,
	})

	require.NoError(t, err)
	require.Equal(t, "480", repo.updates[SettingKeySubscriptionQuotaResetUTCOffsetMinutes])
	require.Equal(t, "4", repo.updates[SettingKeySubscriptionQuotaResetHour])
}

func TestSettingService_UpdateSettings_RejectsInvalidSubscriptionQuotaResetSettings(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{SubscriptionQuotaResetUTCOffsetMinutes: 481})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUBSCRIPTION_QUOTA_RESET_UTC_OFFSET", infraerrors.Reason(err))

	err = svc.UpdateSettings(context.Background(), &SystemSettings{SubscriptionQuotaResetHour: 24})
	require.Error(t, err)
	require.Equal(t, "INVALID_SUBSCRIPTION_QUOTA_RESET_HOUR", infraerrors.Reason(err))
}

func TestSettingService_ParseSettings_SubscriptionQuotaResetDefaultsAndValues(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})

	defaults := svc.parseSettings(map[string]string{})
	require.Equal(t, 0, defaults.SubscriptionQuotaResetUTCOffsetMinutes)
	require.Equal(t, 0, defaults.SubscriptionQuotaResetHour)

	parsed := svc.parseSettings(map[string]string{
		SettingKeySubscriptionQuotaResetUTCOffsetMinutes: "480",
		SettingKeySubscriptionQuotaResetHour:             "4",
	})
	require.Equal(t, 480, parsed.SubscriptionQuotaResetUTCOffsetMinutes)
	require.Equal(t, 4, parsed.SubscriptionQuotaResetHour)
}
```

- [ ] **Step 2: 验证红灯**

Run: `go test -tags unit ./internal/service -run "TestSettingService_(UpdateSettings_SubscriptionQuotaResetSettings|UpdateSettings_RejectsInvalidSubscriptionQuotaResetSettings|ParseSettings_SubscriptionQuotaResetDefaultsAndValues)" -count=1`
Expected: FAIL，新字段和 key 不存在。

- [ ] **Step 3: 最小实现**

新增两个 setting key、`SystemSettings` 字段、默认值、parseSettings 读取、buildSystemSettingsUpdates 校验和写入。校验错误 reason 分别为：

```go
INVALID_SUBSCRIPTION_QUOTA_RESET_UTC_OFFSET
INVALID_SUBSCRIPTION_QUOTA_RESET_HOUR
```

- [ ] **Step 4: 验证绿灯**

Run: `go test -tags unit ./internal/service -run "TestSettingService_(UpdateSettings_SubscriptionQuotaResetSettings|UpdateSettings_RejectsInvalidSubscriptionQuotaResetSettings|ParseSettings_SubscriptionQuotaResetDefaultsAndValues)" -count=1`
Expected: PASS。

## Task 4: Gateway 和管理员设置 API 贯通

**Files:**
- Modify: `backend/internal/service/setting_service.go`
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Modify: `backend/internal/handler/admin/setting_handler.go`

- [ ] **Step 1: 添加 service helper 测试到 Task 3 的测试文件**

```go
func TestSettingService_GetSubscriptionQuotaResetConfig(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeySubscriptionQuotaResetUTCOffsetMinutes: "480",
		SettingKeySubscriptionQuotaResetHour:             "4",
	}}
	svc := NewSettingService(repo, &config.Config{})

	got := svc.GetSubscriptionQuotaResetConfig(context.Background())

	require.Equal(t, SubscriptionQuotaResetConfig{UTCOffsetMinutes: 480, ResetHour: 4}, got)
}
```

- [ ] **Step 2: 验证红灯**

Run: `go test -tags unit ./internal/service -run TestSettingService_GetSubscriptionQuotaResetConfig -count=1`
Expected: FAIL，helper 不存在。

- [ ] **Step 3: 最小实现**

实现 `GetSubscriptionQuotaResetConfig(ctx)`，内部读 `GetMultiple`，失败或非法值返回默认配置。`buildUsageBillingCommand()` 调用一个小 helper，从 `deps.settingService` 或参数传入的 setting service 取得配置；若暂不扩展 deps，则在 `billingDeps` 中加入 `settingService *SettingService` 并由 Gateway/OpenAI Gateway 填充。

管理员 DTO 和 handler 映射新字段：

```go
SubscriptionQuotaResetUTCOffsetMinutes int `json:"subscription_quota_reset_utc_offset_minutes"`
SubscriptionQuotaResetHour             int `json:"subscription_quota_reset_hour"`
```

- [ ] **Step 4: 验证绿灯**

Run: `go test -tags unit ./internal/service -run TestSettingService_GetSubscriptionQuotaResetConfig -count=1`
Expected: PASS。

## Task 5: 前端设置表单、路由和侧边栏

**Files:**
- Modify: `frontend/src/api/admin/settings.ts`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh-Hant.ts`

- [ ] **Step 1: 写类型/静态红灯**

先修改 `SettingsView.vue` 使用两个新字段，但暂不修改 `SystemSettings` interface。然后运行：

Run: `pnpm typecheck`
Expected: FAIL，`subscription_quota_reset_utc_offset_minutes` 或 `subscription_quota_reset_hour` 不存在于 `SystemSettings`。

- [ ] **Step 2: 最小实现**

在 `SystemSettings` interface 增加两个字段。设置页默认值、保存 payload 和表单 UI 加两个控件。UTC 偏移选项由常量数组生成，小时使用 number input/select。移动 sidebar item 到 subscriptions group。新增 `/admin/subscriptions/plans` route，旧 route redirect。

- [ ] **Step 3: 验证绿灯**

Run: `pnpm typecheck`
Expected: PASS。

## Task 6: 全量验证

**Files:**
- No production edit unless verification reveals a defect.

- [ ] **Step 1: 后端目标测试**

Run: `go test -tags unit ./internal/repository ./internal/service ./internal/handler/admin -count=1`
Expected: PASS。

- [ ] **Step 2: 前端构建验证**

Run: `pnpm typecheck`
Expected: PASS。

Run: `pnpm build`
Expected: PASS。

- [ ] **Step 3: 检查 diff**

Run: `git status --short` and `git diff --stat`
Expected: 只包含 issue #47 相关文件。
