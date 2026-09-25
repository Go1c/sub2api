package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type BasisPointsImageRelaySettings struct {
	Enabled bool
	BaseURL string
}

func normalizeBasisPointsImageRelaySettings(enabled bool, baseURL string) (BasisPointsImageRelaySettings, error) {
	baseURL = strings.TrimSpace(baseURL)
	if enabled || baseURL != "" {
		if err := ValidateImageRelayOrigin(baseURL); err != nil {
			return BasisPointsImageRelaySettings{}, infraerrors.BadRequest("INVALID_EXCEL_BPS_IMAGE_BASE_URL", err.Error())
		}
	}
	return BasisPointsImageRelaySettings{Enabled: enabled, BaseURL: strings.TrimRight(baseURL, "/")}, nil
}

// GetBasisPointsImageRelaySettings reads the two image-relay keys on each
// request so a save takes effect without a restart.
func (s *SettingService) GetBasisPointsImageRelaySettings(ctx context.Context) (BasisPointsImageRelaySettings, error) {
	if s == nil || s.settingRepo == nil {
		return BasisPointsImageRelaySettings{}, nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, gatewayForwardingDBTimeout)
	defer cancel()
	values, err := s.settingRepo.GetMultiple(dbCtx, []string{SettingKeyBasisPointsImageRelayEnabled, SettingKeyBasisPointsImageBaseURL})
	if err != nil {
		return BasisPointsImageRelaySettings{}, infraerrors.ServiceUnavailable("EXCEL_BPS_IMAGE_SETTINGS_UNAVAILABLE", "Excel BPS image settings are unavailable")
	}
	return normalizeBasisPointsImageRelaySettings(values[SettingKeyBasisPointsImageRelayEnabled] == "true", values[SettingKeyBasisPointsImageBaseURL])
}
