package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration171ChannelMonitorAPIMode(t *testing.T) {
	content, err := FS.ReadFile("171_channel_monitor_openai_api_mode.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions'")
	require.Contains(t, sql, "channel_monitors_api_mode_check")
	require.Contains(t, sql, "channel_monitor_request_templates_api_mode_check")
}

func TestMigration171aChannelMonitorJitter(t *testing.T) {
	content, err := FS.ReadFile("171a_channel_monitor_jitter.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS jitter_seconds INTEGER NOT NULL DEFAULT 0")
}

func TestMigration172SchedulerOutboxDedupKey(t *testing.T) {
	content, err := FS.ReadFile("172_scheduler_outbox_dedup_key.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS dedup_key TEXT")
}

func TestMigration172aSchedulerOutboxDedupKeyIndexNotx(t *testing.T) {
	content, err := FS.ReadFile("172a_scheduler_outbox_pending_dedup_key_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_scheduler_outbox_pending_dedup_key")
}

func TestMissingUpstreamColumnMigrationsSortOrder(t *testing.T) {
	require.True(t, "171_channel_monitor_openai_api_mode.sql" < "172_scheduler_outbox_dedup_key.sql")
	require.True(t, "172_scheduler_outbox_dedup_key.sql" < "172a_scheduler_outbox_pending_dedup_key_index_notx.sql")
}
