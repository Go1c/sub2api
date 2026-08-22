//go:build unit

package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureCodexFingerprintSeedSQLPreservesValidSeed(t *testing.T) {
	sql := ensureCodexFingerprintSeedSQL("COALESCE(extra, '{}'::jsonb) || $1::jsonb")
	require.Contains(t, sql, "codex_fingerprint_seed")
	require.Contains(t, sql, "to_jsonb(extra ->> 'codex_fingerprint_seed')")
	require.Contains(t, sql, "gen_random_uuid()")
}

func TestStripCodexFingerprintSeedFromExtraUpdate(t *testing.T) {
	require.Nil(t, stripCodexFingerprintSeedFromExtraUpdate(nil))
	unchanged := map[string]any{"codex_fingerprint_mode": "session"}
	require.Equal(t, unchanged, stripCodexFingerprintSeedFromExtraUpdate(unchanged))
	stripped := stripCodexFingerprintSeedFromExtraUpdate(map[string]any{
		"codex_fingerprint_mode": "session",
		"codex_fingerprint_seed": "11111111-1111-4111-8111-111111111111",
	})
	require.Equal(t, "session", stripped["codex_fingerprint_mode"])
	_, exists := stripped["codex_fingerprint_seed"]
	require.False(t, exists)
}
