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
