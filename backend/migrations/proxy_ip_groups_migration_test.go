//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyIPGroupsMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("945_proxy_ip_groups.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists proxy_ip_groups")
	require.Contains(t, sql, "per_ip_concurrency")
	require.Contains(t, sql, "create table if not exists proxy_ip_group_members")
	require.Contains(t, sql, "add column if not exists proxy_ip_group_id")
	require.Contains(t, sql, "references proxy_ip_groups(id)")
	require.Contains(t, sql, "proxy_ip_groups_name_alive")
	require.Contains(t, sql, "accounts_proxy_ip_group_id_idx")
	require.NotContains(t, sql, "234_proxy")
	require.NotContains(t, sql, "account_codex_device")
}
