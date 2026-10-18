package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s *SettingService) passkeyConfigured() bool {
	return s != nil && s.cfg != nil && s.cfg.WebAuthn.Enabled
}

// PasskeyEnabled reports the effective runtime switch. WebAuthn deployment
// configuration remains the security boundary.
func (s *SettingService) PasskeyEnabled(ctx context.Context) (bool, error) {
	if !s.passkeyConfigured() {
		return false, nil
	}
	if s.settingRepo == nil {
		return false, fmt.Errorf("read passkey setting: settings store is unavailable")
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPasskeyEnabled)
	if errors.Is(err, ErrSettingNotFound) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read passkey setting: %w", err)
	}
	return strings.TrimSpace(value) == "true", nil
}

// ModelPlazaRuntime is the public model-plaza feature view.
type ModelPlazaRuntime struct {
	Enabled     bool
	RequireAuth bool
	Description string
}

// GetModelPlazaRuntime reads the model-plaza switches. Fail-closed on error.
func (s *SettingService) GetModelPlazaRuntime(ctx context.Context) ModelPlazaRuntime {
	if s == nil || s.settingRepo == nil {
		return ModelPlazaRuntime{}
	}
	vals, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyModelPlazaEnabled,
		SettingKeyModelPlazaRequireAuth,
		SettingKeyModelPlazaDescription,
	})
	if err != nil {
		return ModelPlazaRuntime{}
	}
	return ModelPlazaRuntime{
		Enabled:     vals[SettingKeyModelPlazaEnabled] == "true",
		RequireAuth: vals[SettingKeyModelPlazaRequireAuth] == "true",
		Description: vals[SettingKeyModelPlazaDescription],
	}
}
