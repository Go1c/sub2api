//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelIQMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("943_channel_iq.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists channel_iq_settings")
	require.Contains(t, sql, "create table if not exists channel_iq_results")
	require.Contains(t, sql, "group_ids jsonb")
	require.Contains(t, sql, "auto_enabled")
	require.Contains(t, sql, "interval_seconds")
	require.Contains(t, sql, "on delete cascade")
	require.Contains(t, sql, "references accounts(id)")
}
