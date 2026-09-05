package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsHideLunaAndAutoReview(t *testing.T) {
	ids := DefaultModelIDs()
	require.NotContains(t, ids, "gpt-5.6-luna")
	require.NotContains(t, ids, "codex-auto-review")
	require.Contains(t, ids, "gpt-5.6-terra")
}

func TestDefaultModelsIncludeGPT6Astra(t *testing.T) {
	ids := DefaultModelIDs()
	require.Contains(t, ids, "gpt-6-astra")
	require.Contains(t, ids, "gpt-6")
	require.NotContains(t, ids, "gpt-5.6-luna")

	var astraDisplay, aliasDisplay string
	for _, model := range DefaultModels {
		switch model.ID {
		case "gpt-6-astra":
			astraDisplay = model.DisplayName
		case "gpt-6":
			aliasDisplay = model.DisplayName
		}
	}
	require.Equal(t, "GPT-6 Astra", astraDisplay)
	require.Equal(t, "GPT-6 (Astra)", aliasDisplay)
}
