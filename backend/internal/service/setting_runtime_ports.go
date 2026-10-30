package service

import (
	"context"
)

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
