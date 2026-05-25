package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountOpenAIImageGenerationRoutingMode(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    string
	}{
		{
			name: "defaults to mixed",
			account: &Account{
				Platform: PlatformOpenAI,
			},
			want: OpenAIImageGenerationRoutingMixed,
		},
		{
			name: "top level routing mode wins",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					openAIImageGenerationRoutingExtraKey: OpenAIImageGenerationRoutingImageOnly,
				},
			},
			want: OpenAIImageGenerationRoutingImageOnly,
		},
		{
			name: "nested routing mode is normalized",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					PlatformOpenAI: map[string]any{
						openAIImageGenerationRoutingExtraKey: "text-only",
					},
				},
			},
			want: OpenAIImageGenerationRoutingTextOnly,
		},
		{
			name: "legacy disabled flag maps to text only",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					"openai_image_generation_enabled": false,
				},
			},
			want: OpenAIImageGenerationRoutingTextOnly,
		},
		{
			name: "top level routing mode overrides legacy disabled flag",
			account: &Account{
				Platform: PlatformOpenAI,
				Extra: map[string]any{
					openAIImageGenerationRoutingExtraKey: OpenAIImageGenerationRoutingImageOnly,
					"openai_image_generation_enabled":    false,
				},
			},
			want: OpenAIImageGenerationRoutingImageOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.OpenAIImageGenerationRoutingMode())
		})
	}
}

func TestAccountSupportsOpenAIImageGenerationRouting(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Extra: map[string]any{
			openAIImageGenerationRoutingExtraKey: OpenAIImageGenerationRoutingTextOnly,
		},
	}

	require.False(t, account.SupportsOpenAIImageGenerationRouting(true))
	require.True(t, account.SupportsOpenAIImageGenerationRouting(false))

	account.Extra[openAIImageGenerationRoutingExtraKey] = OpenAIImageGenerationRoutingImageOnly
	require.True(t, account.SupportsOpenAIImageGenerationRouting(true))
	require.False(t, account.SupportsOpenAIImageGenerationRouting(false))
}
