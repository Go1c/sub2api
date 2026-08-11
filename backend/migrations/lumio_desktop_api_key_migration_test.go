package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration923PreservesDuplicateCredentialsAndAddsPartialUniqueIndex(t *testing.T) {
	raw, err := FS.ReadFile("923_lumio_desktop_api_key_unique.sql")
	require.NoError(t, err)

	sql := string(raw)
	normalized := strings.Join(strings.Fields(sql), " ")
	require.Contains(t, normalized, "ROW_NUMBER() OVER")
	require.Contains(t, normalized, "PARTITION BY user_id ORDER BY created_at, id")
	require.Contains(t, normalized, "Lumio Codex Desktop (legacy ")
	require.Contains(t, normalized, "CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_lumio_desktop_active_unique")
	require.Contains(t, normalized, "deleted_at IS NULL AND name = 'Lumio Codex Desktop'")
	require.NotContains(t, strings.ToUpper(sql), "DELETE FROM API_KEYS")
}
