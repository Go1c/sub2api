//go:build unit

package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBalanceDebitWalletMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("928_balance_debit_wallet.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists balance_debit_clients")
	require.Contains(t, sql, "create table if not exists balance_debit_transactions")
	require.Contains(t, sql, "create table if not exists balance_cache_invalidation_outbox")
	require.Regexp(t, regexp.MustCompile(`amount\s+numeric\(20,2\)`), sql)
	require.Regexp(t, regexp.MustCompile(`balance_before\s+numeric\(20,8\)`), sql)
	require.Regexp(t, regexp.MustCompile(`balance_after\s+numeric\(20,8\)`), sql)
	require.Contains(t, sql, "unique (balance_client_id, user_id, idempotency_key_hash)")
	require.Contains(t, sql, "unique (user_id)")
	require.Contains(t, sql, "references balance_debit_clients(id) on delete restrict")
	require.NotContains(t, sql, "references users(id) on delete cascade")
	require.NotContains(t, sql, "idempotency_key varchar")
	require.NotContains(t, sql, "client_secret varchar")
}
