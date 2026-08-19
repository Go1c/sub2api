package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectAccountForModelWithExclusions_ExplicitLunaOptIn(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Priority:    1,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"}},
			},
			{
				ID:          2,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Priority:    20,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"}},
			},
		}},
		cache: &stubGatewayCache{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(context.Background(), nil, "", "gpt-5.6-luna", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(2), account.ID)
	require.Equal(t, "gpt-5.6-luna", normalizeOpenAIModelForUpstream(account, "gpt-5.6-luna"))
}

func TestSelectAccountForModelWithExclusions_AutoReviewFollowsLunaOptIn(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"}},
			},
			{
				ID:          2,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"}},
			},
		}},
		cache: &stubGatewayCache{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(context.Background(), nil, "", "codex-auto-review", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(2), account.ID)
	require.Equal(t, "gpt-5.6-luna", normalizeOpenAIModelForUpstream(account, "codex-auto-review"))
}

func TestSelectAccountForModelWithExclusions_LunaFallsBackToTerraWithoutOptIn(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"}},
			},
		}},
		cache: &stubGatewayCache{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(context.Background(), nil, "", "gpt-5.6-luna", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
	require.Equal(t, "gpt-5.6-terra", normalizeOpenAIModelForUpstream(account, "gpt-5.6-luna"))
}

func TestSelectAccountForModelWithExclusions_LunaAccountExcludedFallsBackToTerra(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"}},
			},
			{
				ID:          2,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"}},
			},
		}},
		cache: &stubGatewayCache{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(context.Background(), nil, "", "gpt-5.6-luna", map[int64]struct{}{2: {}})
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
}

func TestSelectAccountForModelWithExclusions_EmptyMappingDoesNotStealLuna(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Priority:    1,
			},
			{
				ID:          2,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Priority:    50,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"}},
			},
		}},
		cache: &stubGatewayCache{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(context.Background(), nil, "", "gpt-5.6-luna", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(2), account.ID)
}

func TestOpenAIGatewayService_ResolveOpenAIHiddenIngressModel(t *testing.T) {
	t.Parallel()

	withLuna := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"}},
			},
		}},
	}
	withoutLuna := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"}},
			},
		}},
	}

	require.Equal(t, "gpt-5.6-luna", withLuna.ResolveOpenAIHiddenIngressModel(context.Background(), nil, "gpt-5.6-luna"))
	require.Equal(t, "gpt-5.6-luna", withLuna.ResolveOpenAIHiddenIngressModel(context.Background(), nil, "codex-auto-review"))
	require.Equal(t, "gpt-5.6-terra", withoutLuna.ResolveOpenAIHiddenIngressModel(context.Background(), nil, "gpt-5.6-luna"))
	require.Equal(t, "gpt-5.6-terra", withoutLuna.ResolveOpenAIHiddenIngressModel(context.Background(), nil, "codex-auto-review"))
	require.Equal(t, "gpt-5.6-sol", withLuna.ResolveOpenAIHiddenIngressModel(context.Background(), nil, "gpt-5.6-sol"))
}

func TestResolveOpenAIAccountUpstreamModelForRequest_PassthroughHiddenLuna(t *testing.T) {
	t.Parallel()

	passthrough := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_passthrough": true,
		},
	}
	require.True(t, passthrough.IsOpenAIPassthroughEnabled())
	require.Equal(t, "gpt-5.6-terra", resolveOpenAIAccountUpstreamModelForRequest(passthrough, "gpt-5.6-luna", false))

	optIn := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_passthrough": true,
		},
		Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"},
		},
	}
	require.Equal(t, "gpt-5.6-luna", resolveOpenAIAccountUpstreamModelForRequest(optIn, "gpt-5.6-luna", false))
}
