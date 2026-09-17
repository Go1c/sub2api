//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorIQMigration(t *testing.T) {
	content, err := FS.ReadFile("946_channel_monitor_iq.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS channel_monitors_check_mode_check")
	require.Contains(t, sql, "CHECK (check_mode IN ('probe', 'quota', 'quota_probe', 'iq'))")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS iq_question TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS iq_answer VARCHAR(200) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS iq_fuzzy_match BOOLEAN NOT NULL DEFAULT TRUE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS iq_status VARCHAR(32) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "CHECK (iq_status IN ('', 'iq_ok', 'iq_down', 'test_error', 'monitor_network'))")
}
