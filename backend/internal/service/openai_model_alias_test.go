package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteOpenAIHiddenIngressModel_LunaAndAutoReviewBecomeTerra(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6-luna":             "gpt-5.6-terra",
		"openai/gpt-5.6-luna":      "gpt-5.6-terra",
		"gpt-5.6-luna-2026-07-09":  "gpt-5.6-terra",
		"gpt-5.6-luna-max":         "gpt-5.6-terra",
		"codex-auto-review":        "gpt-5.6-terra",
		"openai/codex-auto-review": "gpt-5.6-terra",
		"codex-auto-review-high":   "gpt-5.6-terra",
		"gpt-5.6-terra":            "gpt-5.6-terra",
		"gpt-5.6-sol":              "gpt-5.6-sol",
		"gpt-5.4":                  "gpt-5.4",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, RewriteOpenAIHiddenIngressModel(input))
		})
	}
}

func TestNormalizeKnownOpenAICodexModel_BareGPT56RoutesToSol(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6":            "gpt-5.6-sol",
		"openai/gpt-5.6":     "gpt-5.6-sol",
		"gpt5.6":             "gpt-5.6-sol",
		"gpt-5.6-high":       "gpt-5.6-sol",
		"gpt-5.6-max":        "gpt-5.6-sol",
		"gpt-5.6-2026-07-09": "gpt-5.6-sol",
		"openai/gpt-5.6-max": "gpt-5.6-sol",
		"gpt-5.6-luna":       "gpt-5.6-luna",
		"codex-auto-review":  "gpt-5.6-luna",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, normalizeKnownOpenAICodexModel(input))
		})
	}
}

func TestAccountExplicitlyServesOpenAIHiddenLuna(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{name: "nil account", want: false},
		{name: "empty mapping", account: &Account{Credentials: map[string]any{}}, want: false},
		{name: "nil credentials", account: &Account{}, want: false},
		{
			name: "wide gpt-5.6 wildcard does not opt in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.6-*": "gpt-5.6-sol"},
			}},
			want: false,
		},
		{
			name: "star wildcard does not opt in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"*": "gpt-5.6-terra"},
			}},
			want: false,
		},
		{
			name: "terra only does not opt in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.6-terra": "gpt-5.6-terra"},
			}},
			want: false,
		},
		{
			name: "explicit luna key opts in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-luna"},
			}},
			want: true,
		},
		{
			name: "luna-max key opts in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.6-luna-max": "gpt-5.6-luna"},
			}},
			want: true,
		},
		{
			name: "luna mapped to terra still opts in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.6-luna": "gpt-5.6-terra"},
			}},
			want: true,
		},
		{
			name: "auto-review key mapped to luna opts in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"codex-auto-review": "gpt-5.6-luna"},
			}},
			want: true,
		},
		{
			name: "identity whitelist auto-review does not opt in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"codex-auto-review": "codex-auto-review"},
			}},
			want: false,
		},
		{
			name: "auto-review mapped to terra does not opt in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"codex-auto-review": "gpt-5.6-terra"},
			}},
			want: false,
		},
		{
			name: "prefixed luna key opts in",
			account: &Account{Credentials: map[string]any{
				"model_mapping": map[string]any{"openai/gpt-5.6-luna": "gpt-5.6-luna"},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, accountExplicitlyServesOpenAIHiddenLuna(tt.account))
		})
	}
}

func TestCanonicalizeOpenAIHiddenIngressModel(t *testing.T) {
	t.Parallel()

	require.Equal(t, "gpt-5.6-luna", canonicalizeOpenAIHiddenIngressModel("gpt-5.6-luna"))
	require.Equal(t, "gpt-5.6-luna-max", canonicalizeOpenAIHiddenIngressModel("gpt-5.6-luna-max"))
	require.Equal(t, "gpt-5.6-luna", canonicalizeOpenAIHiddenIngressModel("codex-auto-review"))
	require.Equal(t, "gpt-5.6-luna", canonicalizeOpenAIHiddenIngressModel("openai/codex-auto-review"))
	require.Equal(t, "gpt-5.6-luna", canonicalizeOpenAIHiddenIngressModel("codex-auto-review-high"))
	require.Equal(t, "gpt-5.6-terra", canonicalizeOpenAIHiddenIngressModel("gpt-5.6-terra"))
	require.Equal(t, "gpt-5.4", canonicalizeOpenAIHiddenIngressModel("gpt-5.4"))
}

func TestResolveOpenAIHiddenIngressModel_FallbackVsOptIn(t *testing.T) {
	t.Parallel()

	require.Equal(t, "gpt-5.6-terra", ResolveOpenAIHiddenIngressModel("gpt-5.6-luna", false))
	require.Equal(t, "gpt-5.6-terra", ResolveOpenAIHiddenIngressModel("codex-auto-review", false))
	require.Equal(t, "gpt-5.6-luna", ResolveOpenAIHiddenIngressModel("gpt-5.6-luna", true))
	require.Equal(t, "gpt-5.6-luna", ResolveOpenAIHiddenIngressModel("codex-auto-review", true))
	require.Equal(t, "gpt-5.6-luna-max", ResolveOpenAIHiddenIngressModel("gpt-5.6-luna-max", true))
	require.Equal(t, "gpt-5.6-sol", ResolveOpenAIHiddenIngressModel("gpt-5.6-sol", true))
}

func TestUsageBillingModelCandidates_BareGPT56IncludesSol(t *testing.T) {
	require.Equal(t,
		[]string{"gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("gpt-5.6"),
	)
	require.Equal(t,
		[]string{"openai/gpt-5.6", "gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("openai/gpt-5.6"),
	)
}
