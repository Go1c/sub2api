//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newLumioDesktopConfigHandler(values map[string]string) *SettingHandler {
	repo := &settingHandlerPublicRepoStub{values: values}
	return NewSettingHandler(service.NewSettingService(repo, &config.Config{}), "test-version")
}

func invokeLumioDesktopConfigHandler(t *testing.T, h *SettingHandler, etag string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/config", nil)
	if etag != "" {
		c.Request.Header.Set("If-None-Match", etag)
	}
	h.GetLumioDesktopConfig(c)
	c.Writer.WriteHeaderNow()
	return recorder
}

func TestSettingHandler_GetLumioDesktopConfig_ReturnsWhitelistAndCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newLumioDesktopConfigHandler(map[string]string{
		service.SettingKeyRegistrationEnabled: "true",
		service.SettingPaymentEnabled:         "true",
		service.SettingKeyLumioDesktopConfig: `{
			"default_model":"gpt-test",
			"payment_url":"/payment?source=desktop",
			"min_client_version":"1.2.3",
			"update_notice":"Update available",
			"feature_flags":{"registration":true,"payment_handoff":true,"key_provisioning":false}
		}`,
	})

	recorder := invokeLumioDesktopConfigHandler(t, h, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "public, max-age=300, stale-if-error=86400", recorder.Header().Get("Cache-Control"))
	require.Regexp(t, `^"[a-f0-9]{64}"$`, recorder.Header().Get("ETag"))

	var envelope struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Code)
	require.ElementsMatch(t, []string{
		"default_model",
		"payment_url",
		"min_client_version",
		"update_notice",
		"feature_flags",
	}, mapKeys(envelope.Data))
	require.Equal(t, "gpt-test", envelope.Data["default_model"])
	require.Equal(t, "/payment?source=desktop", envelope.Data["payment_url"])
	require.Equal(t, "1.2.3", envelope.Data["min_client_version"])
	require.Equal(t, "Update available", envelope.Data["update_notice"])
	require.Equal(t, map[string]any{
		"registration":     true,
		"payment_handoff":  true,
		"key_provisioning": false,
	}, envelope.Data["feature_flags"])
}

func TestSettingHandler_GetLumioDesktopConfig_HonorsIfNoneMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newLumioDesktopConfigHandler(map[string]string{})
	first := invokeLumioDesktopConfigHandler(t, h, "")
	require.Equal(t, http.StatusOK, first.Code)

	second := invokeLumioDesktopConfigHandler(t, h, first.Header().Get("ETag"))

	require.Equal(t, http.StatusNotModified, second.Code)
	require.Empty(t, second.Body.String())
	require.Equal(t, first.Header().Get("ETag"), second.Header().Get("ETag"))
	require.Equal(t, "public, max-age=300, stale-if-error=86400", second.Header().Get("Cache-Control"))
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
