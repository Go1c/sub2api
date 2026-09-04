package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration939UsageLogCorrelationID(t *testing.T) {
	content, err := FS.ReadFile("939_usage_log_correlation_id.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(64) NULL")
	require.NotContains(t, sql, "CREATE INDEX")
	require.NotRegexp(t, `(?i)ADD COLUMN[^;]*DEFAULT`, sql)
}
