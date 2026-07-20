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

type modelMarketGroupListerStub struct {
	groups []Group
	err    error
}

func (s modelMarketGroupListerStub) ListActive(context.Context) ([]Group, error) {
	return s.groups, s.err
}

type modelMarketAccountListerStub struct {
	accountsByGroup map[int64][]Account
	err             error
}

func (s modelMarketAccountListerStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	accounts := s.accountsByGroup[groupID]
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out, nil
}

func newModelMarketTestService(groups []Group, accountsByGroup map[int64][]Account, values map[string]string) (*ModelMarketService, *modelMarketRepoStub) {
	repo := &modelMarketRepoStub{values: values}
	settings := NewSettingService(repo, &config.Config{})
	billing := NewBillingService(&config.Config{}, nil)
	return NewModelMarketService(
		settings,
		modelMarketGroupListerStub{groups: groups},
		modelMarketAccountListerStub{accountsByGroup: accountsByGroup},
		billing,
	), repo
}

func TestModelMarket_DefaultAutoSyncReadsPublicGroupAccountModelsAndRates(t *testing.T) {
	groups := []Group{
		{
			ID:             10,
			Name:           "claude-public",
			Platform:       PlatformAnthropic,
			RateMultiplier: 1.4,
			Status:         StatusActive,
			IsExclusive:    false,
		},
		{
			ID:             20,
			Name:           "openai-vip",
			Platform:       PlatformOpenAI,
			RateMultiplier: 0.2,
			Status:         StatusActive,
			IsExclusive:    true,
		},
		{
			ID:             30,
			Name:           "disabled-public",
			Platform:       PlatformGemini,
			RateMultiplier: 1,
			Status:         StatusDisabled,
			IsExclusive:    false,
		},
	}
	accountsByGroup := map[int64][]Account{
		10: {
			{
				ID:       1,
				Name:     "Claude account",
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"claude-sonnet-4-6": "claude-sonnet-4-6",
					},
				},
			},
		},
		20: {
			{
				ID:       2,
				Name:     "OpenAI private account",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-private": "gpt-private",
					},
				},
			},
		},
		30: {
			{
				ID:       3,
				Name:     "Gemini disabled account",
				Platform: PlatformGemini,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gemini-disabled": "gemini-disabled",
					},
				},
			},
		},
	}
	svc, _ := newModelMarketTestService(groups, accountsByGroup, nil)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)

	require.True(t, got.Enabled)
	require.Len(t, got.Models, 1)
	model := got.Models[0]
	require.Equal(t, "anthropic:claude-sonnet-4-6", model.Key)
	require.Equal(t, "claude-sonnet-4-6", model.Name)
	require.Equal(t, PlatformAnthropic, model.Platform)
	require.Equal(t, []string{"Claude account"}, model.Channels)
	require.NotNil(t, model.Pricing)
	require.Equal(t, string(BillingModeToken), model.BillingMode)
	require.NotNil(t, model.Pricing.InputPrice)
	require.NotNil(t, model.Pricing.OutputPrice)
	require.Len(t, model.Groups, 1)
	require.Equal(t, int64(10), model.Groups[0].ID)
	require.Equal(t, 1.4, model.Groups[0].RateMultiplier)
	require.False(t, model.Groups[0].IsExclusive)
}

func TestModelMarket_ManualSelectionFiltersAndOrdersCandidates(t *testing.T) {
	groups := []Group{
		{
			ID:             10,
			Name:           "openai-public",
			Platform:       PlatformOpenAI,
			RateMultiplier: 0.35,
			Status:         StatusActive,
			IsExclusive:    false,
		},
	}
	accountsByGroup := map[int64][]Account{
		10: {
			{
				ID:       1,
				Name:     "OpenAI",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-a": "gpt-a",
						"gpt-b": "gpt-b",
						"gpt-c": "gpt-c",
					},
				},
			},
		},
	}
	svc, _ := newModelMarketTestService(groups, accountsByGroup, nil)
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

func TestModelMarket_AutoSyncAppliesConfiguredBillingOverrides(t *testing.T) {
	perRequestPrice := 0.42
	image1KPrice := 0.05
	image4KPrice := 0.15
	inputPrice := 0.0000015
	outputPrice := 0.000009
	groups := []Group{
		{
			ID:             10,
			Name:           "openai-public",
			Platform:       PlatformOpenAI,
			RateMultiplier: 0.35,
			Status:         StatusActive,
			IsExclusive:    false,
		},
	}
	accountsByGroup := map[int64][]Account{
		10: {
			{
				ID:       1,
				Name:     "OpenAI",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-image":  "gpt-image",
						"gpt-manual": "gpt-manual",
						"gpt-once":   "gpt-once",
					},
				},
			},
		},
	}
	svc, _ := newModelMarketTestService(groups, accountsByGroup, nil)
	_, err := svc.SetConfig(context.Background(), ModelMarketConfig{
		Enabled:  true,
		AutoSync: true,
		SelectedModels: []ModelMarketSelection{
			{
				Key:         "openai:gpt-once",
				Platform:    PlatformOpenAI,
				Model:       "gpt-once",
				Enabled:     true,
				BillingMode: string(BillingModePerRequest),
				Pricing: &ModelMarketPricing{
					BillingMode:     string(BillingModePerRequest),
					PerRequestPrice: &perRequestPrice,
				},
			},
			{
				Key:         "openai:gpt-manual",
				Platform:    PlatformOpenAI,
				Model:       "gpt-manual",
				Enabled:     true,
				BillingMode: string(BillingModeToken),
				Pricing: &ModelMarketPricing{
					BillingMode: string(BillingModeToken),
					InputPrice:  &inputPrice,
					OutputPrice: &outputPrice,
				},
			},
			{
				Key:         "openai:gpt-image",
				Platform:    PlatformOpenAI,
				Model:       "gpt-image",
				Enabled:     true,
				BillingMode: string(BillingModeImage),
				Pricing: &ModelMarketPricing{
					BillingMode: string(BillingModeImage),
					Intervals: []ModelMarketPricingInterval{
						{TierLabel: "1K", PerRequestPrice: &image1KPrice},
						{TierLabel: "4K", PerRequestPrice: &image4KPrice},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, got.Models, 3)

	byKey := map[string]ModelMarketModel{}
	for _, model := range got.Models {
		byKey[model.Key] = model
	}
	perRequestModel := byKey["openai:gpt-once"]
	require.Equal(t, string(BillingModePerRequest), perRequestModel.BillingMode)
	require.NotNil(t, perRequestModel.Pricing)
	require.Equal(t, string(BillingModePerRequest), perRequestModel.Pricing.BillingMode)
	require.NotNil(t, perRequestModel.Pricing.PerRequestPrice)
	require.InDelta(t, perRequestPrice, *perRequestModel.Pricing.PerRequestPrice, 1e-12)

	tokenModel := byKey["openai:gpt-manual"]
	require.Equal(t, string(BillingModeToken), tokenModel.BillingMode)
	require.NotNil(t, tokenModel.Pricing)
	require.Equal(t, string(BillingModeToken), tokenModel.Pricing.BillingMode)
	require.NotNil(t, tokenModel.Pricing.InputPrice)
	require.NotNil(t, tokenModel.Pricing.OutputPrice)
	require.InDelta(t, inputPrice, *tokenModel.Pricing.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *tokenModel.Pricing.OutputPrice, 1e-12)

	imageModel := byKey["openai:gpt-image"]
	require.Equal(t, string(BillingModeImage), imageModel.BillingMode)
	require.NotNil(t, imageModel.Pricing)
	require.Equal(t, string(BillingModeImage), imageModel.Pricing.BillingMode)
	require.Len(t, imageModel.Pricing.Intervals, 2)
	require.Equal(t, "1K", imageModel.Pricing.Intervals[0].TierLabel)
	require.NotNil(t, imageModel.Pricing.Intervals[0].PerRequestPrice)
	require.InDelta(t, image1KPrice, *imageModel.Pricing.Intervals[0].PerRequestPrice, 1e-12)
	require.Equal(t, "4K", imageModel.Pricing.Intervals[1].TierLabel)
	require.NotNil(t, imageModel.Pricing.Intervals[1].PerRequestPrice)
	require.InDelta(t, image4KPrice, *imageModel.Pricing.Intervals[1].PerRequestPrice, 1e-12)
}

func TestModelMarket_AdminCandidatesIncludeExclusiveGroupModelsWithoutPublicGroups(t *testing.T) {
	groups := []Group{
		{
			ID:             20,
			Name:           "openai-vip",
			Platform:       PlatformOpenAI,
			RateMultiplier: 0.2,
			Status:         StatusActive,
			IsExclusive:    true,
		},
	}
	accountsByGroup := map[int64][]Account{
		20: {
			{
				ID:       1,
				Name:     "OpenAI Internal",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-private": "gpt-private",
					},
				},
			},
		},
	}
	svc, _ := newModelMarketTestService(groups, accountsByGroup, nil)

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
	svc, _ := newModelMarketTestService(nil, nil, nil)

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

func TestModelMarket_DefaultAutoSyncReadsExactGroupRoutingWithoutChannels(t *testing.T) {
	groups := []Group{
		{
			ID:                  10,
			Name:                "claude-routed",
			Platform:            PlatformAnthropic,
			RateMultiplier:      1.2,
			Status:              StatusActive,
			IsExclusive:         false,
			ModelRoutingEnabled: true,
			ModelRouting: map[string][]int64{
				"claude-sonnet-4-6": {1},
				"claude-opus-*":     {2},
			},
		},
	}
	svc, _ := newModelMarketTestService(groups, nil, nil)

	got, err := svc.GetPublic(context.Background())
	require.NoError(t, err)

	require.Len(t, got.Models, 1)
	model := got.Models[0]
	require.Equal(t, "anthropic:claude-sonnet-4-6", model.Key)
	require.Equal(t, []string{"分组路由"}, model.Channels)
	require.Len(t, model.Groups, 1)
	require.Equal(t, int64(10), model.Groups[0].ID)
	require.NotNil(t, model.Pricing)
}
