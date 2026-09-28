//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type availableGroupModelUserRepo struct {
	*userRepoStub
	user *User
}

func (r availableGroupModelUserRepo) GetByID(context.Context, int64) (*User, error) {
	return r.user, nil
}

type availableGroupModelGroupRepo struct {
	*stubGroupRepoForAvailable
}

type availableGroupModelSubRepo struct {
	userSubRepoNoop
}

func (availableGroupModelSubRepo) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}

func TestGetAvailableGroupModelsIncludesOnlyBindableExclusiveGroups(t *testing.T) {
	groups := []Group{
		{ID: 10, Name: "public", Platform: PlatformAnthropic, Status: StatusActive},
		{ID: 20, Name: "mine", Platform: PlatformOpenAI, Status: StatusActive, IsExclusive: true},
		{ID: 30, Name: "other", Platform: PlatformOpenAI, Status: StatusActive, IsExclusive: true},
	}
	accounts := map[int64][]Account{
		10: {{
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"claude-sonnet-4-6": "claude-sonnet-4-6"},
			},
		}},
		20: {{
			Platform: PlatformOpenAI,
			Name:     "private-account",
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-private": "gpt-private"},
			},
		}},
		30: {{
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-other": "gpt-other"},
			},
		}},
	}
	market, _ := newModelMarketTestService(groups, accounts, nil)
	svc := &APIKeyService{
		userRepo: &availableGroupModelUserRepo{
			userRepoStub: &userRepoStub{},
			user:         &User{ID: 1, AllowedGroups: []int64{20}},
		},
		groupRepo: &availableGroupModelGroupRepo{stubGroupRepoForAvailable: &stubGroupRepoForAvailable{
			activeGroups: groups,
		}},
		userSubRepo:        availableGroupModelSubRepo{},
		modelMarketService: market,
	}

	got, err := svc.GetAvailableGroupModels(context.Background(), 1)
	require.NoError(t, err)

	names := make([]string, 0, len(got))
	for _, model := range got {
		names = append(names, model.Name)
		require.NotContains(t, model.Channels, "private-account")
	}
	require.ElementsMatch(t, []string{"claude-sonnet-4-6", "gpt-private"}, names)
}
