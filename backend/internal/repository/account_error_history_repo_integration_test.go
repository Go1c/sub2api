//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// newAccountErrorHistoryTestRepo 用全局集成 DB/ent client 构造 repo，并按需关闭节流。
func newAccountErrorHistoryTestRepo(minInterval time.Duration) *accountErrorHistoryRepository {
	return &accountErrorHistoryRepository{
		client:      integrationEntClient,
		db:          integrationDB,
		minInterval: minInterval,
	}
}

// mustCreateAccountForErrorHistory 直接插一行最小 accounts 记录，返回其 id。
// CASCADE 让 t.Cleanup 删账号时连带清掉 account_error_histories。
func mustCreateAccountForErrorHistory(t *testing.T, name string) int64 {
	t.Helper()
	ctx := context.Background()
	var id int64
	err := integrationDB.QueryRowContext(ctx, `
		INSERT INTO accounts (name, platform, type, status, credentials, extra)
		VALUES ($1, 'anthropic', 'api_key', 'active', '{}'::jsonb, '{}'::jsonb)
		RETURNING id
	`, name).Scan(&id)
	require.NoError(t, err, "insert account")
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM accounts WHERE id = $1`, id)
	})
	return id
}

func countAccountErrorHistory(t *testing.T, accountID int64) int {
	t.Helper()
	var n int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM account_error_histories WHERE account_id = $1`, accountID).Scan(&n))
	return n
}

func intPtr(v int) *int       { return &v }
func int64Ptr(v int64) *int64 { return &v }

// TestAccountErrorHistory_DedupFold 验证：60s 窗口内同维度只一行，重复仅 dup_count 递增。
func TestAccountErrorHistory_DedupFold(t *testing.T) {
	repo := newAccountErrorHistoryTestRepo(0) // 关闭节流，专测折叠
	accountID := mustCreateAccountForErrorHistory(t, "err-hist-dedup")
	ctx := context.Background()

	row := &service.AccountErrorRow{
		AccountID:          accountID,
		UpstreamStatusCode: intPtr(429),
		Source:             service.AccountErrorSourceGateway,
		Message:            "rate limited",
		Fingerprint:        "fp-rate-limited",
	}

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Record(ctx, row))
	}

	require.Equal(t, 1, countAccountErrorHistory(t, accountID), "should fold into a single row")

	entries, err := repo.ListRecent(ctx, accountID, 20)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, 5, entries[0].DupCount, "dup_count should increment to 5")
	require.Equal(t, 429, *entries[0].UpstreamStatusCode)
}

// TestAccountErrorHistory_DedupNullStatus 验证：upstream_status_code 为 NULL 时去重比较正确。
func TestAccountErrorHistory_DedupNullStatus(t *testing.T) {
	repo := newAccountErrorHistoryTestRepo(0)
	accountID := mustCreateAccountForErrorHistory(t, "err-hist-null-status")
	ctx := context.Background()

	row := &service.AccountErrorRow{
		AccountID:   accountID,
		Source:      service.AccountErrorSourceTest,
		Message:     "test failed",
		Fingerprint: "fp-test-failed",
	}
	require.NoError(t, repo.Record(ctx, row))
	require.NoError(t, repo.Record(ctx, row))

	require.Equal(t, 1, countAccountErrorHistory(t, accountID), "NULL status rows must fold via IS NOT DISTINCT FROM")
	entries, err := repo.ListRecent(ctx, accountID, 20)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, 2, entries[0].DupCount)
	require.Nil(t, entries[0].UpstreamStatusCode)
	require.Nil(t, entries[0].UserEmail)
}

// TestAccountErrorHistory_TrimToMax 验证：单账号行数稳定不超过上限（写 60 条不同指纹 -> 50）。
func TestAccountErrorHistory_TrimToMax(t *testing.T) {
	repo := newAccountErrorHistoryTestRepo(0) // 关闭节流，确保 60 条都尝试插入
	accountID := mustCreateAccountForErrorHistory(t, "err-hist-trim")
	ctx := context.Background()

	for i := 0; i < 60; i++ {
		err := repo.Record(ctx, &service.AccountErrorRow{
			AccountID:   accountID,
			Source:      service.AccountErrorSourceSchedule,
			Message:     fmt.Sprintf("distinct error %d", i),
			Fingerprint: fmt.Sprintf("fp-%d", i),
		})
		require.NoError(t, err)
	}

	require.Equal(t, accountErrorHistoryMaxRows, countAccountErrorHistory(t, accountID),
		"row count must be trimmed to the per-account cap")

	// 保留的应是最新的（fp-59 在最近），ListRecent 倒序首条为最后写入的指纹。
	entries, err := repo.ListRecent(ctx, accountID, 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "distinct error 59", entries[0].Message)
}

// TestAccountErrorHistory_Throttle 验证：最小写入间隔内的新错误被节流丢弃。
func TestAccountErrorHistory_Throttle(t *testing.T) {
	repo := newAccountErrorHistoryTestRepo(5 * time.Second) // 大间隔确保第二条被节流
	accountID := mustCreateAccountForErrorHistory(t, "err-hist-throttle")
	ctx := context.Background()

	require.NoError(t, repo.Record(ctx, &service.AccountErrorRow{
		AccountID:   accountID,
		Source:      service.AccountErrorSourceSchedule,
		Message:     "first",
		Fingerprint: "fp-first",
	}))
	// 不同指纹（非折叠），但在最小间隔内 -> 被丢弃。
	require.NoError(t, repo.Record(ctx, &service.AccountErrorRow{
		AccountID:   accountID,
		Source:      service.AccountErrorSourceSchedule,
		Message:     "second",
		Fingerprint: "fp-second",
	}))

	require.Equal(t, 1, countAccountErrorHistory(t, accountID), "second distinct error within min interval must be throttled")
}

// TestAccountErrorHistory_ListRecentOrderAndLimit 验证：倒序返回且 limit 生效。
func TestAccountErrorHistory_ListRecentOrderAndLimit(t *testing.T) {
	repo := newAccountErrorHistoryTestRepo(0)
	accountID := mustCreateAccountForErrorHistory(t, "err-hist-list")
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Record(ctx, &service.AccountErrorRow{
			AccountID:          accountID,
			UserID:             int64Ptr(int64(i + 1)),
			UserEmail:          strPtr(fmt.Sprintf("u%d@example.com", i)),
			Model:              strPtr("claude-3"),
			UpstreamStatusCode: intPtr(500),
			Source:             service.AccountErrorSourceGateway,
			Message:            fmt.Sprintf("err %d", i),
			Fingerprint:        fmt.Sprintf("fp-list-%d", i),
		}))
	}

	entries, err := repo.ListRecent(ctx, accountID, 3)
	require.NoError(t, err)
	require.Len(t, entries, 3, "limit must be respected")
	require.Equal(t, "err 4", entries[0].Message, "most recent first")
	require.Equal(t, "u4@example.com", *entries[0].UserEmail)
	require.Equal(t, "claude-3", *entries[0].Model)
}
