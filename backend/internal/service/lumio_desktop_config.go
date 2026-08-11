package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/mod/semver"
)

const (
	LumioDesktopDefaultModel            = "gpt-5.4"
	LumioDesktopDefaultPaymentURL       = "/payment"
	LumioDesktopDefaultMinClientVersion = "0.0.0"
	LumioDesktopUpdateNoticeMaxRunes    = 2000
)

// LumioDesktopFeatureFlags contains only remotely switchable desktop behavior.
// Registration and payment handoff are additionally gated by their global settings.
type LumioDesktopFeatureFlags struct {
	Registration    bool `json:"registration"`
	PaymentHandoff  bool `json:"payment_handoff"`
	KeyProvisioning bool `json:"key_provisioning"`
}

// LumioDesktopConfig is the stable, public bootstrap contract for Lumio Codex.
type LumioDesktopConfig struct {
	DefaultModel     string                   `json:"default_model"`
	PaymentURL       string                   `json:"payment_url"`
	MinClientVersion string                   `json:"min_client_version"`
	UpdateNotice     string                   `json:"update_notice"`
	FeatureFlags     LumioDesktopFeatureFlags `json:"feature_flags"`
}

func DefaultLumioDesktopConfig() LumioDesktopConfig {
	return LumioDesktopConfig{
		DefaultModel:     LumioDesktopDefaultModel,
		PaymentURL:       LumioDesktopDefaultPaymentURL,
		MinClientVersion: LumioDesktopDefaultMinClientVersion,
		FeatureFlags: LumioDesktopFeatureFlags{
			Registration:    true,
			PaymentHandoff:  false,
			KeyProvisioning: false,
		},
	}
}

func normalizeLumioDesktopVersion(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	if value == "" || !semver.IsValid("v"+value) {
		return "", false
	}
	return value, true
}

func isSafeLumioDesktopPaymentURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, `\`) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "" && parsed.Host == "" && parsed.User == nil && parsed.Opaque == "" && strings.HasPrefix(parsed.Path, "/")
}

func truncateLumioDesktopNotice(raw string) string {
	runes := []rune(strings.TrimSpace(raw))
	if len(runes) > LumioDesktopUpdateNoticeMaxRunes {
		runes = runes[:LumioDesktopUpdateNoticeMaxRunes]
	}
	return string(runes)
}

func normalizeLumioDesktopConfig(raw, defaultModel string) LumioDesktopConfig {
	defaults := DefaultLumioDesktopConfig()
	defaults.DefaultModel = firstNonEmpty(defaultModel, defaults.DefaultModel)
	cfg := defaults
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			cfg = defaults
		}
	}

	cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = firstNonEmpty(defaultModel, defaults.DefaultModel)
	}
	if !isSafeLumioDesktopPaymentURL(cfg.PaymentURL) {
		cfg.PaymentURL = defaults.PaymentURL
	} else {
		cfg.PaymentURL = strings.TrimSpace(cfg.PaymentURL)
	}
	if version, ok := normalizeLumioDesktopVersion(cfg.MinClientVersion); ok {
		cfg.MinClientVersion = version
	} else {
		cfg.MinClientVersion = defaults.MinClientVersion
	}
	cfg.UpdateNotice = truncateLumioDesktopNotice(cfg.UpdateNotice)
	return cfg
}

func normalizeLumioDesktopConfigForWrite(cfg LumioDesktopConfig) (LumioDesktopConfig, error) {
	defaults := DefaultLumioDesktopConfig()
	cfg.DefaultModel = firstNonEmpty(cfg.DefaultModel, defaults.DefaultModel)
	cfg.PaymentURL = firstNonEmpty(cfg.PaymentURL, defaults.PaymentURL)
	cfg.MinClientVersion = firstNonEmpty(cfg.MinClientVersion, defaults.MinClientVersion)
	cfg.UpdateNotice = strings.TrimSpace(cfg.UpdateNotice)

	if !isSafeLumioDesktopPaymentURL(cfg.PaymentURL) {
		return LumioDesktopConfig{}, infraerrors.BadRequest(
			"INVALID_LUMIO_DESKTOP_PAYMENT_URL",
			"lumio desktop payment URL must be a same-origin absolute path",
		)
	}
	version, ok := normalizeLumioDesktopVersion(cfg.MinClientVersion)
	if !ok {
		return LumioDesktopConfig{}, infraerrors.BadRequest(
			"INVALID_LUMIO_DESKTOP_MIN_CLIENT_VERSION",
			"lumio desktop minimum client version must be valid semver",
		)
	}
	if len([]rune(cfg.UpdateNotice)) > LumioDesktopUpdateNoticeMaxRunes {
		return LumioDesktopConfig{}, infraerrors.BadRequest(
			"INVALID_LUMIO_DESKTOP_UPDATE_NOTICE",
			"lumio desktop update notice is too long",
		)
	}
	cfg.PaymentURL = strings.TrimSpace(cfg.PaymentURL)
	cfg.MinClientVersion = version
	return cfg, nil
}

func ValidateLumioDesktopConfig(cfg LumioDesktopConfig) error {
	_, err := normalizeLumioDesktopConfigForWrite(cfg)
	return err
}

// GetLumioDesktopConfig returns the public desktop bootstrap document with
// global registration and payment kill switches applied.
func (s *SettingService) GetLumioDesktopConfig(ctx context.Context) (*LumioDesktopConfig, error) {
	keys := []string{
		SettingKeyLumioDesktopConfig,
		SettingKeyCCSwitchDefaultModelOpenAI,
		SettingKeyRegistrationEnabled,
		SettingKeyBackendModeEnabled,
		SettingPaymentEnabled,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, infraerrors.InternalServer("GET_LUMIO_DESKTOP_CONFIG_FAILED", "failed to load lumio desktop configuration").WithCause(err)
	}

	cfg := normalizeLumioDesktopConfig(
		settings[SettingKeyLumioDesktopConfig],
		settings[SettingKeyCCSwitchDefaultModelOpenAI],
	)
	cfg.FeatureFlags.Registration = cfg.FeatureFlags.Registration &&
		settings[SettingKeyRegistrationEnabled] == "true" &&
		settings[SettingKeyBackendModeEnabled] != "true"
	cfg.FeatureFlags.PaymentHandoff = cfg.FeatureFlags.PaymentHandoff && settings[SettingPaymentEnabled] == "true"
	return &cfg, nil
}
