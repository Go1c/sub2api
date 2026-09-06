package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration941UsageLogUpstreamResponseModel(t *testing.T) {
	content, err := FS.ReadFile("941_add_usage_log_upstream_response_model.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE usage_logs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS upstream_response_model VARCHAR(200)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS upstream_model_mismatch BOOLEAN")
	require.NotContains(t, sql, "CREATE INDEX")
}

func TestMigration942UsageLogUpstreamModelMismatchIndex(t *testing.T) {
	content, err := FS.ReadFile("942_add_usage_log_upstream_model_mismatch_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_upstream_model_mismatch_created_at")
	require.Contains(t, sql, "WHERE upstream_model_mismatch IS TRUE")
	require.True(t, "941_add_usage_log_upstream_response_model.sql" < "942_add_usage_log_upstream_model_mismatch_index_notx.sql")
}
