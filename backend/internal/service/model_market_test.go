//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type modelMarketRepoStub struct {
	values map[string]string
}

func (s *modelMarketRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *modelMarketRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *modelMarketRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *modelMarketRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *modelMarketRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (s *modelMarketRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *modelMarketRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type modelMarketChannelListerStub struct {
	channels []AvailableChannel
	err      error
}

func (s modelMarketChannelListerStub) ListAvailable(context.Context) ([]AvailableChannel, error) {
	return s.channels, s.err
}

func newModelMarketTestService(channels []AvailableChannel, values map[string]string) (*ModelMarketService, *modelMarketRepoStub) {
	repo := &modelMarketRepoStub{values: values}
	settings := NewSettingService(repo, &config.Config{})
	return NewModelMarketService(settings, modelMarketChannelListerStub{channels: channels}), repo
}

func TestModelMarket_DefaultAutoSyncReadsPublicChannelPricingAndGroupRates(t *testing.T) {
	inputPrice := 0.000003
	outputPrice := 0.000015
	privatePrice := 0.99
	channels := []AvailableChannel{
		{
			ID:     1,
			Name:   "Claude",
			Status: StatusActive,
			Groups: []AvailableGroupRef{
				{ID: 10, Name: "claude-public", Platform: PlatformAnthropic, RateMultiplier: 1.4, IsExclusive: false},
				{ID: 11, Name: "claude-private", Platform: PlatformAnthropic, RateMultiplier: 0.7, IsExclusive: true},
			},
			SupportedModels: []SupportedModel{
				{
					Name:     "claude-sonnet-4-6",
					Platform: PlatformAnthropic,
					Pricing: &ChannelModelPricing{
						BillingMode: BillingModeToken,
						InputPrice:  &inputPrice,
						OutputPrice: &outputPrice,
					},
				},
			},
		},
		{
			ID:     2,
			Name:   "Private OpenAI",
			Status: StatusActive,
			Groups: []AvailableGroupRef{
				{ID: 20, Name: "openai-vip", Platform: PlatformOpenAI, RateMultiplier: 0.2, IsExclusive: true},
			},
			SupportedModels: []SupportedModel{
				{
					Name:     "gpt-private",
					Platform: PlatformOpenAI,
					Pricing:  &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: &privatePrice},
				},
			},
		},
		{
			ID:     3,
			Name:   "Disabled",
			Status: StatusDisabled,
			Groups: []AvailableGroupRef{
				{ID: 30, Name: "disabled-public", Platform: PlatformGemini, RateMultiplier: 1, IsExclusive: false},
			},
			SupportedModels: []SupportedModel{{Name: "gemini-disabled", Platform: PlatformGemini}},
		},
	}
	svc, _ := newModelMarketTestService(channels, nil)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)

	require.True(t, got.Enabled)
	require.Len(t, got.Models, 1)
	model := got.Models[0]
	require.Equal(t, "anthropic:claude-sonnet-4-6", model.Key)
	require.Equal(t, "claude-sonnet-4-6", model.Name)
	require.Equal(t, PlatformAnthropic, model.Platform)
	require.Equal(t, []string{"Claude"}, model.Channels)
	require.NotNil(t, model.Pricing)
	require.Equal(t, string(BillingModeToken), model.BillingMode)
	require.InDelta(t, inputPrice, *model.Pricing.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *model.Pricing.OutputPrice, 1e-12)
	require.Len(t, model.Groups, 1)
	require.Equal(t, int64(10), model.Groups[0].ID)
	require.Equal(t, 1.4, model.Groups[0].RateMultiplier)
	require.False(t, model.Groups[0].IsExclusive)
}

func TestModelMarket_ManualSelectionFiltersAndOrdersCandidates(t *testing.T) {
	channels := []AvailableChannel{
		{
			ID:     1,
			Name:   "OpenAI",
			Status: StatusActive,
			Groups: []AvailableGroupRef{
				{ID: 10, Name: "openai-public", Platform: PlatformOpenAI, RateMultiplier: 0.35, IsExclusive: false},
			},
			SupportedModels: []SupportedModel{
				{Name: "gpt-a", Platform: PlatformOpenAI},
				{Name: "gpt-b", Platform: PlatformOpenAI},
				{Name: "gpt-c", Platform: PlatformOpenAI},
			},
		},
	}
	svc, _ := newModelMarketTestService(channels, nil)
	_, err := svc.SetConfig(context.Background(), ModelMarketConfig{
		Enabled:  true,
		AutoSync: false,
		SelectedModels: []ModelMarketSelection{
			{Key: "openai:gpt-a", Enabled: true, SortOrder: 20},
			{Key: "openai:gpt-b", Enabled: true, SortOrder: 10},
			{Key: "openai:gpt-c", Enabled: false, SortOrder: 1},
			{Key: "openai:missing", Enabled: true, SortOrder: 0},
		},
	})
	require.NoError(t, err)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)

	require.Len(t, got.Models, 2)
	require.Equal(t, "gpt-b", got.Models[0].Name)
	require.Equal(t, "gpt-a", got.Models[1].Name)
}

func TestModelMarket_AdminCandidatesIncludeChannelModelsWithoutPublicGroups(t *testing.T) {
	inputPrice := 0.000001
	channels := []AvailableChannel{
		{
			ID:     1,
			Name:   "OpenAI Internal",
			Status: StatusActive,
			Groups: []AvailableGroupRef{
				{ID: 20, Name: "openai-vip", Platform: PlatformOpenAI, RateMultiplier: 0.2, IsExclusive: true},
			},
			SupportedModels: []SupportedModel{
				{
					Name:     "gpt-private",
					Platform: PlatformOpenAI,
					Pricing:  &ChannelModelPricing{BillingMode: BillingModeToken, InputPrice: &inputPrice},
				},
			},
		},
	}
	svc, _ := newModelMarketTestService(channels, nil)

	admin, err := svc.GetAdmin(context.Background())
	require.NoError(t, err)
	require.Len(t, admin.Candidates, 1)
	require.Equal(t, "openai:gpt-private", admin.Candidates[0].Key)
	require.Empty(t, admin.Candidates[0].Groups)
	require.Len(t, admin.Models, 0)

	public, err := svc.GetPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, public.Models, 0)
}

func TestModelMarket_CustomModelsAreDisplayedWithConfiguredPricingAndGroups(t *testing.T) {
	inputPrice := 0.000004
	outputPrice := 0.000012
	svc, _ := newModelMarketTestService(nil, nil)

	_, err := svc.SetConfig(context.Background(), ModelMarketConfig{
		Enabled:  true,
		AutoSync: true,
		CustomModels: []ModelMarketCustomModel{
			{
				Platform:    PlatformOpenAI,
				Model:       "gpt-custom",
				Enabled:     true,
				SortOrder:   15,
				BillingMode: string(BillingModeToken),
				Pricing: &ModelMarketPricing{
					BillingMode: string(BillingModeToken),
					InputPrice:  &inputPrice,
					OutputPrice: &outputPrice,
				},
				Groups: []ModelMarketGroup{
					{
						ID:               101,
						Name:             "自定义公开组",
						Platform:         PlatformOpenAI,
						SubscriptionType: SubscriptionTypeStandard,
						RateMultiplier:   0.6,
						IsExclusive:      false,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)

	require.Len(t, got.Models, 1)
	model := got.Models[0]
	require.Equal(t, "custom:openai:gpt-custom", model.Key)
	require.Equal(t, "gpt-custom", model.Name)
	require.Equal(t, PlatformOpenAI, model.Platform)
	require.Equal(t, string(BillingModeToken), model.BillingMode)
	require.Equal(t, 15, model.SortOrder)
	require.Equal(t, []string{"自定义"}, model.Channels)
	require.NotNil(t, model.Pricing)
	require.InDelta(t, inputPrice, *model.Pricing.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *model.Pricing.OutputPrice, 1e-12)
	require.Len(t, model.Groups, 1)
	require.Equal(t, "自定义公开组", model.Groups[0].Name)
	require.InDelta(t, 0.6, model.Groups[0].RateMultiplier, 1e-12)
}
