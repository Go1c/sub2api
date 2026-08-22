//go:build unit

package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration931AddsChannelModelTimePricing(t *testing.T) {
	content, err := FS.ReadFile("931_channel_model_time_pricing.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE channel_model_pricing")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS time_pricing JSONB")
}
