//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBestCodexFirstPartyTelemetryMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("937_bestcodex_first_party_telemetry.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists telemetry_events")
	require.Contains(t, sql, "create table if not exists user_first_party_attribution")
	require.Contains(t, sql, "unique")
	require.Contains(t, sql, "dedup_key")
	require.Contains(t, sql, "where dedup_key <> ''")
	require.NotContains(t, sql, "drop ")
	require.NotContains(t, sql, "drop table")
	require.NotContains(t, sql, "drop index")
}
