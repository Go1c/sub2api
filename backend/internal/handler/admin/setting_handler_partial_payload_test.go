//go:build unit

package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

// Saving settings is a whole-document PUT. A client that sends only the field it
// cares about must not reset everything else: a payload as small as
// `{"risk_control_enabled":true}` used to clear site_name, after which
// getStringOrDefault rendered the empty value as the built-in default and the
// login page silently changed name.

func TestUpdateSettingsPartialPayloadKeepsUnsentKeys(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName:         "Example Gateway",
		service.SettingKeySiteSubtitle:     "Example Gateway Platform",
		service.SettingKeySMTPHost:         "smtp.example.com",
		service.SettingKeySMTPFrom:         "noreply@example.com",
		service.SettingKeyTurnstileEnabled: "true",
	})

	rec := doUpdateSettings(t, h, map[string]any{"risk_control_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "true", repo.values[service.SettingKeyRiskControlEnabled],
		"the field the caller actually sent must be written")

	require.Equal(t, "Example Gateway", repo.values[service.SettingKeySiteName])
	require.Equal(t, "Example Gateway Platform", repo.values[service.SettingKeySiteSubtitle])
	require.Equal(t, "smtp.example.com", repo.values[service.SettingKeySMTPHost])
	require.Equal(t, "noreply@example.com", repo.values[service.SettingKeySMTPFrom])
	require.Equal(t, "true", repo.values[service.SettingKeyTurnstileEnabled])
}

// A full payload keeps whole-document semantics: fields explicitly set to their
// zero value are still cleared.
func TestUpdateSettingsFullPayloadStillClearsSentEmptyFields(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName: "Example Gateway",
	})

	rec := doUpdateSettings(t, h, map[string]any{"site_name": ""}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "", repo.values[service.SettingKeySiteName],
		"an explicitly sent empty value is a deliberate clear, not an omission")
}

// smtp_from_email is the one request field whose JSON name differs from its
// setting key; the alias keeps it from being treated as always-omitted.
func TestUpdateSettingsSMTPFromAliasIsWritable(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySMTPFrom: "old@example.com",
	})

	rec := doUpdateSettings(t, h, map[string]any{"smtp_from_email": "new@example.com"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "new@example.com", repo.values[service.SettingKeySMTPFrom])
}

func TestUpdateSettingsCCSwitchDefaultModelsRoundTrip(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyCCSwitchDefaultModelOpenAI: "gpt-5.4",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"ccswitch_default_model_anthropic":          "claude-sonnet-4-5",
		"ccswitch_default_model_openai":             "gpt-5.4-custom",
		"ccswitch_default_model_gemini":             "gemini-3-pro",
		"ccswitch_default_model_antigravity":        "claude-opus-4-6",
		"ccswitch_default_model_antigravity_gemini": "gemini-3-flash",
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "claude-sonnet-4-5", repo.values[service.SettingKeyCCSwitchDefaultModelAnthropic])
	require.Equal(t, "gpt-5.4-custom", repo.values[service.SettingKeyCCSwitchDefaultModelOpenAI])
	require.Equal(t, "gemini-3-pro", repo.values[service.SettingKeyCCSwitchDefaultModelGemini])
	require.Equal(t, "claude-opus-4-6", repo.values[service.SettingKeyCCSwitchDefaultModelAntigravity])
	require.Equal(t, "gemini-3-flash", repo.values[service.SettingKeyCCSwitchDefaultModelAntigravityGemini])
	require.Contains(t, rec.Body.String(), `"ccswitch_default_model_openai":"gpt-5.4-custom"`)
	require.Contains(t, rec.Body.String(), `"ccswitch_default_model_anthropic":"claude-sonnet-4-5"`)
}

func TestUpdateSettingsPartialPayloadKeepsCCSwitchDefaultModels(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyCCSwitchDefaultModelAnthropic:         "claude-sonnet-4-5",
		service.SettingKeyCCSwitchDefaultModelOpenAI:            "gpt-5.4-custom",
		service.SettingKeyCCSwitchDefaultModelGemini:            "gemini-3-pro",
		service.SettingKeyCCSwitchDefaultModelAntigravity:       "claude-opus-4-6",
		service.SettingKeyCCSwitchDefaultModelAntigravityGemini: "gemini-3-flash",
	})

	rec := doUpdateSettings(t, h, map[string]any{"risk_control_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "claude-sonnet-4-5", repo.values[service.SettingKeyCCSwitchDefaultModelAnthropic])
	require.Equal(t, "gpt-5.4-custom", repo.values[service.SettingKeyCCSwitchDefaultModelOpenAI])
	require.Equal(t, "gemini-3-pro", repo.values[service.SettingKeyCCSwitchDefaultModelGemini])
	require.Equal(t, "claude-opus-4-6", repo.values[service.SettingKeyCCSwitchDefaultModelAntigravity])
	require.Equal(t, "gemini-3-flash", repo.values[service.SettingKeyCCSwitchDefaultModelAntigravityGemini])
}
