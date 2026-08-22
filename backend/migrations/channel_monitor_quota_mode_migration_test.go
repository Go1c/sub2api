//go:build unit

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorQuotaModeMigration(t *testing.T) {
	content, err := FS.ReadFile("933_channel_monitor_quota_mode.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	// fork 不扩 provider CHECK（已是 openai/anthropic/gemini/grok）。
	require.NotContains(t, sql, "channel_monitors_provider_check")
	require.NotContains(t, sql, "antigravity")
	require.NotContains(t, sql, "kimi")
	require.NotContains(t, sql, "zhipu")
	require.NotContains(t, sql, "deepseek")

	// check_mode 三态，默认 probe。
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS check_mode VARCHAR(32) NOT NULL DEFAULT 'probe'")
	require.Contains(t, sql, "CHECK (check_mode IN ('probe', 'quota', 'quota_probe'))")

	// account_id 关联账号，账号删除置空（监控保留，运行时报「账号未关联」）。
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_channel_monitors_account_id ON channel_monitors(account_id)")

	// 历史表配额快照列。
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS quota JSONB")

	// 公开设置默认关闭。
	require.Contains(t, sql, "VALUES ('channel_monitor_show_quota', 'false')")
	require.Contains(t, sql, "ON CONFLICT (key) DO NOTHING")
}
