package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCodexClientIdentityToCLIIsPairedAndIdempotent(t *testing.T) {
	originator, userAgent, changed := NormalizeCodexClientIdentityToCLI(
		CodexTUIOriginator,
		"codex-tui/0.163.0 (Windows 11; x86_64) WindowsTerminal (codex-tui; 0.163.0)",
	)
	require.True(t, changed)

	pairedOriginator, pairedUserAgent, ok := PairCodexClientIdentity(userAgent)
	require.True(t, ok)
	require.Equal(t, originator, pairedOriginator)
	require.Equal(t, userAgent, pairedUserAgent)

	againOriginator, againUserAgent, changedAgain := NormalizeCodexClientIdentityToCLI(originator, userAgent)
	require.False(t, changedAgain)
	require.Equal(t, originator, againOriginator)
	require.Equal(t, userAgent, againUserAgent)
}
