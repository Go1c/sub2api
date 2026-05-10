package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type siteMessageSettingRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *siteMessageSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *siteMessageSettingRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *siteMessageSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *siteMessageSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *siteMessageSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.updates[key] = value
	}
	return nil
}

func (s *siteMessageSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *siteMessageSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingServiceSiteMessageSettingsDefaultsAndPublicFlag(t *testing.T) {
	svc := NewSettingService(&siteMessageSettingRepoStub{values: map[string]string{}}, &config.Config{})

	siteMessageSettings, err := svc.GetSiteMessageSettings(context.Background())
	require.NoError(t, err)
	require.False(t, siteMessageSettings.Enabled)
	require.Equal(t, SiteMessagesDailySendLimitDefault, siteMessageSettings.DailySendLimit)
	require.Equal(t, SiteMessagesRetentionDaysDefault, siteMessageSettings.RetentionDays)

	publicSettings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, publicSettings.SiteMessagesEnabled)
}

func TestSettingServiceSiteMessageSettingsParsesAndClamps(t *testing.T) {
	svc := NewSettingService(&siteMessageSettingRepoStub{
		values: map[string]string{
			SettingKeySiteMessagesEnabled:        "true",
			SettingKeySiteMessagesDailySendLimit: "5000",
			SettingKeySiteMessagesRetentionDays:  "0",
		},
	}, &config.Config{})

	settings, err := svc.GetSiteMessageSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.Enabled)
	require.Equal(t, SiteMessagesDailySendLimitMax, settings.DailySendLimit)
	require.Equal(t, SiteMessagesRetentionDaysDefault, settings.RetentionDays)
}

func TestSettingServiceUpdateSettingsPersistsSiteMessageSettings(t *testing.T) {
	repo := &siteMessageSettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		SiteMessagesEnabled:        true,
		SiteMessagesDailySendLimit: 25,
		SiteMessagesRetentionDays:  45,
	})
	require.NoError(t, err)

	require.Equal(t, "true", repo.updates[SettingKeySiteMessagesEnabled])
	require.Equal(t, "25", repo.updates[SettingKeySiteMessagesDailySendLimit])
	require.Equal(t, "45", repo.updates[SettingKeySiteMessagesRetentionDays])
}
