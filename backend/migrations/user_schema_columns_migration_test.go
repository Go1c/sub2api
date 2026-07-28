package migrations

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fieldNameRe matches ent field declarations like field.Float("frozen_balance").
var fieldNameRe = regexp.MustCompile(`field\.\w+\("([a-z][a-z0-9_]*)"\)`)

// TestUserEntFieldsCoveredByMigrations fails when ent User schema declares a column
// that no SQL migration ever mentions. This is the regression guard for the
// 2026-07-28 production outage where frozen_balance was added to ent/schema/user.go
// without a migration in the published image.
func TestUserEntFieldsCoveredByMigrations(t *testing.T) {
	schemaPath := userSchemaPath(t)
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "read ent user schema")

	fields := fieldNameRe.FindAllStringSubmatch(string(schemaBytes), -1)
	require.NotEmpty(t, fields, "expected at least one field.X(\"name\") in user schema")

	migrationSQL, err := collectEmbeddedMigrationSQL()
	require.NoError(t, err)
	require.NotEmpty(t, migrationSQL)

	var missing []string
	for _, m := range fields {
		name := m[1]
		if !strings.Contains(migrationSQL, name) {
			missing = append(missing, name)
		}
	}
	require.Empty(t, missing,
		"ent User fields with no mention in backend/migrations/*.sql (add a migration before shipping):\n  %s",
		strings.Join(missing, "\n  "))
}

func TestMigration920AddsUserFrozenBalanceIdempotently(t *testing.T) {
	content, err := FS.ReadFile("920_add_user_frozen_balance.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS frozen_balance DECIMAL(20,8) NOT NULL DEFAULT 0")
	// Must remain safe after the manual prod hotfix and after 160 on full-dev installs.
	require.Contains(t, string(content), "IF NOT EXISTS")
	require.NotContains(t, sql, "DROP COLUMN")
}

func TestMigration920AndOptional160CoverFrozenBalance(t *testing.T) {
	// 920 is required on every lineage (publish safety net after the 2026-07-28 outage).
	// 160 exists on full-dev / upstream installs; topic publish branches may only have 920.
	content920, err := FS.ReadFile("920_add_user_frozen_balance.sql")
	require.NoError(t, err)
	require.Contains(t, string(content920), "frozen_balance")
	require.Contains(t, string(content920), "IF NOT EXISTS")

	content160, err := FS.ReadFile("160_add_user_frozen_balance.sql")
	if err != nil {
		t.Logf("160_add_user_frozen_balance.sql not in this build (publish-thin lineage): %v", err)
		return
	}
	require.Contains(t, string(content160), "frozen_balance")
	require.Contains(t, string(content160), "IF NOT EXISTS")
}

func userSchemaPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// backend/migrations/this_test.go -> backend/ent/schema/user.go
	return filepath.Join(filepath.Dir(thisFile), "..", "ent", "schema", "user.go")
}

func collectEmbeddedMigrationSQL() (string, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		raw, err := FS.ReadFile(e.Name())
		if err != nil {
			return "", err
		}
		_, _ = b.Write(raw)
		_ = b.WriteByte('\n')
	}
	return b.String(), nil
}
