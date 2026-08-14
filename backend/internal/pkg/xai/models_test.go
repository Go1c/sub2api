//go:build unit

package xai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripGrokProviderPrefix(t *testing.T) {
	t.Parallel()

	require.Equal(t, "grok-7", StripGrokProviderPrefix("x-ai/grok-7"))
	require.Equal(t, "grok-4.5", StripGrokProviderPrefix("xai/grok-4.5"))
	require.Equal(t, "grok-4.6", StripGrokProviderPrefix("grok/grok-4.6"))
	require.Equal(t, "grok-4.5", StripGrokProviderPrefix("  grok-4.5  "))
	require.Equal(t, "claude-sonnet-4", StripGrokProviderPrefix("claude-sonnet-4"))
}
