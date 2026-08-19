//go:build unit

package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDailyCheckinMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("929_daily_checkin.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists daily_checkin_settings")
	require.Contains(t, sql, "create table if not exists daily_checkin_records")
	require.Contains(t, sql, "create table if not exists daily_checkin_daily_counters")
	require.Contains(t, sql, "default false")
	require.Contains(t, sql, "default 'asia/shanghai'")
	require.Contains(t, sql, "on delete set null")
	require.Contains(t, sql, "unique (user_id, business_date)")
	require.Regexp(t, regexp.MustCompile(`min_reward\s+numeric\(20,8\)`), sql)
	require.Regexp(t, regexp.MustCompile(`actual_reward\s+numeric\(20,8\)`), sql)
	require.Regexp(t, regexp.MustCompile(`awarded_total\s+numeric\(20,8\)`), sql)
	require.NotContains(t, sql, "user_affiliate_ledger")
}
