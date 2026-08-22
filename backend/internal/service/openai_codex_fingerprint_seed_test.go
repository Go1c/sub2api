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

func TestCanonicalCodexFingerprintSeed(t *testing.T) {
	got, ok := canonicalCodexFingerprintSeed("11111111-1111-4111-8111-111111111111")
	require.True(t, ok)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", got)
	_, ok = canonicalCodexFingerprintSeed("")
	require.False(t, ok)
	_, ok = canonicalCodexFingerprintSeed("not-a-uuid")
	require.False(t, ok)
	_, ok = canonicalCodexFingerprintSeed("00000000-0000-0000-0000-000000000000")
	require.False(t, ok)
}

func TestStripCodexFingerprintSeed(t *testing.T) {
	stripped := stripCodexFingerprintSeed(map[string]any{
		"codex_fingerprint_mode": "session",
		"codex_fingerprint_seed": "11111111-1111-4111-8111-111111111111",
	})
	require.Equal(t, "session", stripped["codex_fingerprint_mode"])
	_, exists := stripped["codex_fingerprint_seed"]
	require.False(t, exists)
}
