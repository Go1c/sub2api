package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration170AccountSparkShadowColumns(t *testing.T) {
	content, err := FS.ReadFile("170_account_spark_shadow.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS parent_account_id BIGINT")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS quota_dimension VARCHAR(20) NOT NULL DEFAULT 'global'")
	require.Contains(t, sql, "chk_accounts_quota_dimension")
	require.Contains(t, sql, "chk_accounts_parent_dimension")
	require.Contains(t, sql, "chk_accounts_parent_not_self")
	require.Contains(t, sql, "fk_accounts_parent_account_id")
	require.Contains(t, sql, "VALIDATE CONSTRAINT")
}

func TestMigration170aAccountSparkShadowIndexesNotx(t *testing.T) {
	content, err := FS.ReadFile("170a_account_spark_shadow_indexes_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounts_parent_account_id")
	require.Contains(t, sql, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_accounts_spark_shadow_per_parent")
	require.Contains(t, sql, "quota_dimension = 'spark'")
}

func TestMigration170SortsBeforeLongContextBilling(t *testing.T) {
	// Filename order drives apply order; 170* must precede 175*.
	require.True(t, "170_account_spark_shadow.sql" < "175_default_openai_long_context_billing.sql")
	require.True(t, "170a_account_spark_shadow_indexes_notx.sql" < "175_default_openai_long_context_billing.sql")
}
