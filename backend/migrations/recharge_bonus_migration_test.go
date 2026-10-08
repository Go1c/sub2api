//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRechargeBonusMigrationContract(t *testing.T) {
	raw, err := FS.ReadFile("948_recharge_bonus.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "alter table payment_orders")
	require.Contains(t, sql, "add column if not exists bonus_amount")
	require.Regexp(t, `bonus_amount\s+decimal\(20,2\)`, sql)
	require.Contains(t, sql, "not null default 0")
}
