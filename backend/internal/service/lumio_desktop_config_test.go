//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func newLumioDesktopConfigTestService(values map[string]string) *SettingService {
	if values == nil {
		values = map[string]string{}
	}
	return NewSettingService(&settingPublicRepoStub{values: values}, &config.Config{})
}

func TestDefaultLumioDesktopConfig_DisablesUnavailableFeaturesUntilConfigured(t *testing.T) {
	cfg := DefaultLumioDesktopConfig()

	require.True(t, cfg.FeatureFlags.Registration)
	require.False(t, cfg.FeatureFlags.PaymentHandoff)
	require.False(t, cfg.FeatureFlags.KeyProvisioning)
}

func TestSettingService_GetLumioDesktopConfig_UsesSafeDefaults(t *testing.T) {
	svc := newLumioDesktopConfigTestService(nil)

	got, err := svc.GetLumioDesktopConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, "gpt-5.4", got.DefaultModel)
	require.Equal(t, "/payment", got.PaymentURL)
	require.Equal(t, "0.0.0", got.MinClientVersion)
	require.Empty(t, got.UpdateNotice)
	require.False(t, got.FeatureFlags.Registration)
	require.False(t, got.FeatureFlags.PaymentHandoff)
	require.False(t, got.FeatureFlags.KeyProvisioning)
}

func TestSettingService_GetLumioDesktopConfig_OverlaysStoredDocument(t *testing.T) {
	svc := newLumioDesktopConfigTestService(map[string]string{
		SettingKeyLumioDesktopConfig: `{
			"default_model":"gpt-test",
			"payment_url":"/payment?source=desktop",
			"min_client_version":"1.2.3-beta.1",
			"update_notice":"Restart after updating.",
			"feature_flags":{"registration":true,"payment_handoff":true,"key_provisioning":false}
		}`,
		SettingKeyRegistrationEnabled: "true",
		SettingPaymentEnabled:         "true",
	})

	got, err := svc.GetLumioDesktopConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, "gpt-test", got.DefaultModel)
	require.Equal(t, "/payment?source=desktop", got.PaymentURL)
	require.Equal(t, "1.2.3-beta.1", got.MinClientVersion)
	require.Equal(t, "Restart after updating.", got.UpdateNotice)
	require.True(t, got.FeatureFlags.Registration)
	require.True(t, got.FeatureFlags.PaymentHandoff)
	require.False(t, got.FeatureFlags.KeyProvisioning)
}

func TestSettingService_GetLumioDesktopConfig_GlobalSwitchesWin(t *testing.T) {
	svc := newLumioDesktopConfigTestService(map[string]string{
		SettingKeyLumioDesktopConfig: `{
			"feature_flags":{"registration":true,"payment_handoff":true,"key_provisioning":true}
		}`,
		SettingKeyRegistrationEnabled: "false",
		SettingPaymentEnabled:         "false",
	})

	got, err := svc.GetLumioDesktopConfig(context.Background())

	require.NoError(t, err)
	require.False(t, got.FeatureFlags.Registration)
	require.False(t, got.FeatureFlags.PaymentHandoff)
	require.True(t, got.FeatureFlags.KeyProvisioning)
}

func TestSettingService_GetLumioDesktopConfig_BackendModeDisablesRegistration(t *testing.T) {
	svc := newLumioDesktopConfigTestService(map[string]string{
		SettingKeyLumioDesktopConfig: `{
			"feature_flags":{"registration":true,"payment_handoff":false,"key_provisioning":true}
		}`,
		SettingKeyRegistrationEnabled: "true",
		SettingKeyBackendModeEnabled:  "true",
	})

	got, err := svc.GetLumioDesktopConfig(context.Background())

	require.NoError(t, err)
	require.False(t, got.FeatureFlags.Registration)
}

func TestSettingService_GetLumioDesktopConfig_UsesExistingCodexDefaultAsFallback(t *testing.T) {
	svc := newLumioDesktopConfigTestService(map[string]string{
		SettingKeyCCSwitchDefaultModelOpenAI: "gpt-existing-default",
	})

	got, err := svc.GetLumioDesktopConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, "gpt-existing-default", got.DefaultModel)
}

func TestSettingService_GetLumioDesktopConfig_InvalidStoredValuesFallBack(t *testing.T) {
	tests := []struct {
		name       string
		stored     string
		assertSafe func(*testing.T, *LumioDesktopConfig)
	}{
		{
			name:   "invalid json",
			stored: `{`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Equal(t, "gpt-5.4", got.DefaultModel)
				require.Equal(t, "/payment", got.PaymentURL)
				require.Equal(t, "0.0.0", got.MinClientVersion)
				require.False(t, got.FeatureFlags.Registration)
				require.False(t, got.FeatureFlags.PaymentHandoff)
				require.False(t, got.FeatureFlags.KeyProvisioning)
			},
		},
		{
			name:   "external payment URL",
			stored: `{"payment_url":"https://evil.example/payment"}`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Equal(t, "/payment", got.PaymentURL)
			},
		},
		{
			name:   "protocol relative payment URL",
			stored: `{"payment_url":"//evil.example/payment"}`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Equal(t, "/payment", got.PaymentURL)
			},
		},
		{
			name:   "invalid minimum version",
			stored: `{"min_client_version":"latest"}`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Equal(t, "0.0.0", got.MinClientVersion)
			},
		},
		{
			name:   "blank model uses existing Codex default",
			stored: `{"default_model":" "}`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Equal(t, "gpt-5.4", got.DefaultModel)
			},
		},
		{
			name:   "overlong notice",
			stored: `{"update_notice":"` + strings.Repeat("x", LumioDesktopUpdateNoticeMaxRunes+1) + `"}`,
			assertSafe: func(t *testing.T, got *LumioDesktopConfig) {
				require.Len(t, []rune(got.UpdateNotice), LumioDesktopUpdateNoticeMaxRunes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newLumioDesktopConfigTestService(map[string]string{
				SettingKeyLumioDesktopConfig: tt.stored,
			})

			got, err := svc.GetLumioDesktopConfig(context.Background())

			require.NoError(t, err)
			tt.assertSafe(t, got)
		})
	}
}

func TestValidateLumioDesktopConfig_RejectsUnsafeAdminValues(t *testing.T) {
	tests := []struct {
		name string
		edit func(*LumioDesktopConfig)
		code string
	}{
		{
			name: "invalid minimum version",
			edit: func(cfg *LumioDesktopConfig) { cfg.MinClientVersion = "latest" },
			code: "INVALID_LUMIO_DESKTOP_MIN_CLIENT_VERSION",
		},
		{
			name: "external payment URL",
			edit: func(cfg *LumioDesktopConfig) { cfg.PaymentURL = "https://evil.example/payment" },
			code: "INVALID_LUMIO_DESKTOP_PAYMENT_URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultLumioDesktopConfig()
			tt.edit(&cfg)

			err := ValidateLumioDesktopConfig(cfg)

			require.Error(t, err)
			require.Equal(t, tt.code, infraerrors.Reason(err))
		})
	}
}

func TestSettingService_GetAllSettings_ExposesLumioDesktopConfig(t *testing.T) {
	repo := &settingGetAllRepoStub{values: map[string]string{
		SettingKeyLumioDesktopConfig: `{
			"default_model":"gpt-admin",
			"payment_url":"/payment/admin",
			"min_client_version":"2.3.4",
			"update_notice":"Admin notice",
			"feature_flags":{"registration":false,"payment_handoff":false,"key_provisioning":true}
		}`,
	}}
	svc := NewSettingService(repo, &config.Config{})

	got, err := svc.GetAllSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, "gpt-admin", got.LumioDesktopConfig.DefaultModel)
	require.Equal(t, "/payment/admin", got.LumioDesktopConfig.PaymentURL)
	require.Equal(t, "2.3.4", got.LumioDesktopConfig.MinClientVersion)
	require.Equal(t, "Admin notice", got.LumioDesktopConfig.UpdateNotice)
	require.True(t, got.LumioDesktopConfig.FeatureFlags.KeyProvisioning)
}

func TestSettingService_UpdateSettings_PersistsLumioDesktopConfig(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	cfg := DefaultLumioDesktopConfig()
	cfg.DefaultModel = "gpt-admin-save"
	cfg.PaymentURL = "/payment?from=admin"
	cfg.MinClientVersion = "v3.4.5"
	cfg.UpdateNotice = "  Saved notice  "

	err := svc.UpdateSettings(context.Background(), &SystemSettings{LumioDesktopConfig: cfg})

	require.NoError(t, err)
	var stored LumioDesktopConfig
	require.NoError(t, json.Unmarshal([]byte(repo.updates[SettingKeyLumioDesktopConfig]), &stored))
	require.Equal(t, "gpt-admin-save", stored.DefaultModel)
	require.Equal(t, "/payment?from=admin", stored.PaymentURL)
	require.Equal(t, "3.4.5", stored.MinClientVersion)
	require.Equal(t, "Saved notice", stored.UpdateNotice)
}

func TestSettingService_UpdateSettings_RejectsUnsafeLumioDesktopConfig(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	cfg := DefaultLumioDesktopConfig()
	cfg.PaymentURL = "https://evil.example/payment"

	err := svc.UpdateSettings(context.Background(), &SystemSettings{LumioDesktopConfig: cfg})

	require.Error(t, err)
	require.Equal(t, "INVALID_LUMIO_DESKTOP_PAYMENT_URL", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}
