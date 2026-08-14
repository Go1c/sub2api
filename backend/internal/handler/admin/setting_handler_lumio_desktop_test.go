//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newLumioDesktopAdminSettingsHandler(repo *settingHandlerRepoStub) *SettingHandler {
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	return NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
}

func invokeLumioDesktopAdminSettingsUpdate(t *testing.T, handler *SettingHandler, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.UpdateSettings(c)
	return recorder
}

func TestSettingHandler_GetSettings_ExposesLumioDesktopConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyLumioDesktopConfig: `{
			"default_model":"gpt-admin-get",
			"payment_url":"/payment/admin",
			"min_client_version":"1.4.0",
			"update_notice":"Read from admin",
			"feature_flags":{"registration":true,"payment_handoff":false,"key_provisioning":true}
		}`,
	}}
	handler := newLumioDesktopAdminSettingsHandler(repo)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)

	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	data, ok := envelope.Data.(map[string]any)
	require.True(t, ok)
	desktop, ok := data["lumio_desktop_config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-admin-get", desktop["default_model"])
	require.Equal(t, "/payment/admin", desktop["payment_url"])
	require.Equal(t, "1.4.0", desktop["min_client_version"])
}

func TestSettingHandler_UpdateSettings_PersistsLumioDesktopConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyLumioDesktopConfig: `{"default_model":"gpt-old"}`,
	}}
	handler := newLumioDesktopAdminSettingsHandler(repo)

	recorder := invokeLumioDesktopAdminSettingsUpdate(t, handler, map[string]any{
		"lumio_desktop_config": map[string]any{
			"default_model":      "gpt-admin-put",
			"payment_url":        "/payment/new",
			"min_client_version": "2.5.0",
			"update_notice":      "Saved from admin",
			"feature_flags": map[string]any{
				"registration":     false,
				"payment_handoff":  true,
				"key_provisioning": true,
			},
		},
	})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var stored service.LumioDesktopConfig
	require.NoError(t, json.Unmarshal([]byte(repo.values[service.SettingKeyLumioDesktopConfig]), &stored))
	require.Equal(t, "gpt-admin-put", stored.DefaultModel)
	require.Equal(t, "/payment/new", stored.PaymentURL)
	require.Equal(t, "2.5.0", stored.MinClientVersion)
	require.Equal(t, "Saved from admin", stored.UpdateNotice)
	require.False(t, stored.FeatureFlags.Registration)
	require.True(t, stored.FeatureFlags.PaymentHandoff)
	require.True(t, stored.FeatureFlags.KeyProvisioning)
}

func TestSettingHandler_UpdateSettings_PreservesOmittedLumioDesktopConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const previous = `{"default_model":"gpt-preserved","payment_url":"/payment/preserved","min_client_version":"4.0.0","feature_flags":{"registration":false,"payment_handoff":false,"key_provisioning":false}}`
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyLumioDesktopConfig: previous,
	}}
	handler := newLumioDesktopAdminSettingsHandler(repo)

	recorder := invokeLumioDesktopAdminSettingsUpdate(t, handler, map[string]any{})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var stored service.LumioDesktopConfig
	require.NoError(t, json.Unmarshal([]byte(repo.values[service.SettingKeyLumioDesktopConfig]), &stored))
	require.Equal(t, "gpt-preserved", stored.DefaultModel)
	require.Equal(t, "/payment/preserved", stored.PaymentURL)
	require.Equal(t, "4.0.0", stored.MinClientVersion)
	require.False(t, stored.FeatureFlags.KeyProvisioning)
}

func TestSettingHandler_UpdateSettings_RejectsUnsafeLumioDesktopConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const previous = `{"default_model":"gpt-safe","payment_url":"/payment","min_client_version":"1.0.0"}`
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyLumioDesktopConfig: previous,
	}}
	handler := newLumioDesktopAdminSettingsHandler(repo)

	recorder := invokeLumioDesktopAdminSettingsUpdate(t, handler, map[string]any{
		"lumio_desktop_config": map[string]any{
			"default_model":      "gpt-unsafe",
			"payment_url":        "https://evil.example/payment",
			"min_client_version": "1.0.0",
		},
	})

	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Equal(t, previous, repo.values[service.SettingKeyLumioDesktopConfig])
}
