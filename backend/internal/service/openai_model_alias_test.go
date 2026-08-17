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
		"gpt-5.6-luna":       "gpt-5.6-terra",
		"codex-auto-review":  "gpt-5.6-terra",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, normalizeKnownOpenAICodexModel(input))
		})
	}
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
