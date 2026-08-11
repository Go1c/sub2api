# Lumio Desktop Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan task-by-task (hosts without subagents: its Inline Fallback section). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a public, cacheable, strictly whitelisted desktop configuration endpoint backed by remotely managed settings and safe defaults.

**Architecture:** A typed `LumioDesktopConfig` is serialized under one existing settings row. The service normalizes persisted values and applies global registration/payment kill switches; the public handler emits only the typed document with ETag caching. The existing admin settings GET/PUT carries the same typed document.

**Tech Stack:** Go 1.25.7, Gin, existing settings repository, `golang.org/x/mod/semver`, unit tests with `testify`.

## Global Constraints

- Public route is exactly `GET /api/v1/desktop/config` and requires no JWT.
- Response fields are exactly `default_model`, `payment_url`, `min_client_version`, `update_notice`, and `feature_flags`.
- `payment_url` must be a same-origin path and must never accept a scheme, host, or protocol-relative value.
- Invalid persisted data falls back safely; invalid admin writes are rejected.
- No new dependency and no unrelated settings refactor.

---

### Task 1: Typed service contract and safe parsing

**Files:**
- Create: `backend/internal/service/lumio_desktop_config.go`
- Create: `backend/internal/service/lumio_desktop_config_test.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/service/setting_parse.go`
- Modify: `backend/internal/service/setting_update.go`

**Interfaces:**
- Produces: `DefaultLumioDesktopConfig() LumioDesktopConfig`
- Produces: `(*SettingService).GetLumioDesktopConfig(context.Context) (*LumioDesktopConfig, error)`
- Produces: `ValidateLumioDesktopConfig(LumioDesktopConfig) error`

- [ ] **Step 1: Write failing service tests**

```go
func TestGetLumioDesktopConfigUsesSafeDefaults(t *testing.T) {
    svc := newLumioDesktopConfigTestService(nil)
    got, err := svc.GetLumioDesktopConfig(context.Background())
    require.NoError(t, err)
    require.Equal(t, DefaultLumioDesktopConfig(), *got)
}

func TestGetLumioDesktopConfigOverlaysStoredDocumentAndGlobalSwitches(t *testing.T) {
    svc := newLumioDesktopConfigTestService(map[string]string{
        SettingKeyLumioDesktopConfig: `{"default_model":"gpt-test","payment_url":"/payment","min_client_version":"1.2.3","feature_flags":{"registration":true,"payment_handoff":true,"key_provisioning":false}}`,
        SettingKeyRegistrationEnabled: "false",
        SettingPaymentEnabled: "false",
    })
    got, err := svc.GetLumioDesktopConfig(context.Background())
    require.NoError(t, err)
    require.Equal(t, "gpt-test", got.DefaultModel)
    require.False(t, got.FeatureFlags.Registration)
    require.False(t, got.FeatureFlags.PaymentHandoff)
    require.False(t, got.FeatureFlags.KeyProvisioning)
}

func TestGetLumioDesktopConfigRejectsUnsafePersistedPaymentURL(t *testing.T) {
    svc := newLumioDesktopConfigTestService(map[string]string{
        SettingKeyLumioDesktopConfig: `{"payment_url":"https://evil.example/payment"}`,
    })
    got, err := svc.GetLumioDesktopConfig(context.Background())
    require.NoError(t, err)
    require.Equal(t, "/payment", got.PaymentURL)
}

func TestValidateLumioDesktopConfigRejectsInvalidVersion(t *testing.T) {
    cfg := DefaultLumioDesktopConfig()
    cfg.MinClientVersion = "latest"
    require.Error(t, ValidateLumioDesktopConfig(cfg))
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd backend && go test -tags=unit ./internal/service -run 'LumioDesktopConfig'`

Expected: compile failure because the desktop config contract and setting key do not exist.

- [ ] **Step 3: Implement the minimal typed setting**

```go
type LumioDesktopFeatureFlags struct {
    Registration    bool `json:"registration"`
    PaymentHandoff  bool `json:"payment_handoff"`
    KeyProvisioning bool `json:"key_provisioning"`
}

type LumioDesktopConfig struct {
    DefaultModel     string                   `json:"default_model"`
    PaymentURL       string                   `json:"payment_url"`
    MinClientVersion string                   `json:"min_client_version"`
    UpdateNotice     string                   `json:"update_notice"`
    FeatureFlags     LumioDesktopFeatureFlags `json:"feature_flags"`
}
```

Add `SettingKeyLumioDesktopConfig = "lumio_desktop_config"`, parse from defaults before JSON overlay, gate registration/payment with existing settings, and marshal the validated normalized document during `buildSystemSettingsUpdates`.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `cd backend && go test -tags=unit ./internal/service -run 'LumioDesktopConfig'`

Expected: PASS.

### Task 2: Public handler, route, cache semantics, and admin round trip

**Files:**
- Modify: `backend/internal/handler/setting_handler.go`
- Modify: `backend/internal/handler/setting_handler_public_test.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Modify: `backend/internal/handler/admin/setting_handler.go`
- Modify: `backend/internal/handler/admin/setting_handler_update.go`
- Modify: `backend/internal/server/routes/public.go`

**Interfaces:**
- Consumes: `(*SettingService).GetLumioDesktopConfig(context.Context)`
- Produces: `(*SettingHandler).GetLumioDesktopConfig(*gin.Context)` at `GET /api/v1/desktop/config`

- [ ] **Step 1: Write failing handler tests**

```go
func TestSettingHandlerGetLumioDesktopConfigReturnsWhitelistAndCacheHeaders(t *testing.T) {
    handler := newLumioDesktopConfigHandler(t)
    recorder := httptest.NewRecorder()
    ctx, _ := gin.CreateTestContext(recorder)
    ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/config", nil)
    handler.GetLumioDesktopConfig(ctx)
    require.Equal(t, http.StatusOK, recorder.Code)
    require.NotEmpty(t, recorder.Header().Get("ETag"))
    require.Equal(t, "public, max-age=300, stale-if-error=86400", recorder.Header().Get("Cache-Control"))
    require.JSONEq(t, `{"code":0,"message":"success","data":{"default_model":"gpt-5.4","payment_url":"/payment","min_client_version":"0.0.0","update_notice":"","feature_flags":{"registration":false,"payment_handoff":false,"key_provisioning":true}}}`, recorder.Body.String())
}

func TestSettingHandlerGetLumioDesktopConfigHonorsIfNoneMatch(t *testing.T) {
    handler := newLumioDesktopConfigHandler(t)
    first := httptest.NewRecorder()
    firstCtx, _ := gin.CreateTestContext(first)
    firstCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/config", nil)
    handler.GetLumioDesktopConfig(firstCtx)

    second := httptest.NewRecorder()
    secondCtx, _ := gin.CreateTestContext(second)
    secondCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/config", nil)
    secondCtx.Request.Header.Set("If-None-Match", first.Header().Get("ETag"))
    handler.GetLumioDesktopConfig(secondCtx)
    require.Equal(t, http.StatusNotModified, second.Code)
    require.Empty(t, second.Body.String())
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd backend && go test -tags=unit ./internal/handler -run 'LumioDesktopConfig'`

Expected: compile failure because the handler method does not exist.

- [ ] **Step 3: Implement handler and admin mappings**

The handler serializes only `service.LumioDesktopConfig`, hashes that JSON for a quoted ETag, sets `Cache-Control: public, max-age=300, stale-if-error=86400`, returns 304 for an exact `If-None-Match`, and otherwise uses the standard success envelope. Add the route under `/desktop/config`. Add a typed `lumio_desktop_config` field to admin GET/PUT and preserve the previous value when omitted.

- [ ] **Step 4: Run focused and package tests**

Run: `cd backend && go test -tags=unit ./internal/handler ./internal/handler/admin ./internal/server/routes ./internal/service`

Expected: PASS.

- [ ] **Step 5: Review the task diff and commit**

Run: `git diff --check && git diff --stat`

Commit: `feat(desktop): add public Lumio bootstrap config`
