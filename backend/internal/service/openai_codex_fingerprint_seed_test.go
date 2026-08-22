//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldEnsureCodexFingerprintSeedForExtraUpdates(t *testing.T) {
	require.False(t, ShouldEnsureCodexFingerprintSeedForExtraUpdates(nil))
	require.False(t, ShouldEnsureCodexFingerprintSeedForExtraUpdates(map[string]any{"grok_needs_reauth": true}))
	require.False(t, ShouldEnsureCodexFingerprintSeedForExtraUpdates(map[string]any{"codex_fingerprint_mode": "off"}))
	require.True(t, ShouldEnsureCodexFingerprintSeedForExtraUpdates(map[string]any{"codex_fingerprint_mode": "session"}))
	require.True(t, ShouldEnsureCodexFingerprintSeedForExtraUpdates(map[string]any{"codex_fingerprint_mode": "device"}))
}
