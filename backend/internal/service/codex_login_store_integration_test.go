//go:build integration

package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestCodexLoginStorePostgresLeasesAndRecovery(t *testing.T) {
	dsn := os.Getenv("CODEX_LOGIN_TEST_DSN")
	if dsn == "" {
		t.Skip("set CODEX_LOGIN_TEST_DSN to an isolated PostgreSQL")
	}
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("codex_test_%d", time.Now().UnixNano())
	_, err = admin.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	db, err := sql.Open("postgres", dsn+" search_path="+schema)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`CREATE TABLE accounts(id BIGINT PRIMARY KEY,deleted_at TIMESTAMPTZ,status TEXT,error_message TEXT)`)
	require.NoError(t, err)
	migration, err := os.ReadFile("../../migrations/948_codex_login_jobs.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	store := &codexLoginStore{db}
	ctx := context.Background()
	id, err := store.Put(ctx, "one@example.com", "encrypted-only", CodexLoginOptions{})
	require.NoError(t, err)
	var jobs [2]*CodexLoginJob
	var errs [2]error
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); jobs[i], errs[i] = store.Claim(ctx, fmt.Sprintf("lease%d", i)) }(i)
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	var claimed *CodexLoginJob
	count := 0
	for _, job := range jobs {
		if job != nil {
			count++
			claimed = job
		}
	}
	require.Equal(t, 1, count)
	require.Equal(t, id, claimed.ID)
	_, err = db.Exec(`INSERT INTO accounts(id) VALUES(7)`)
	require.NoError(t, err)
	require.NoError(t, store.Bind(ctx, claimed, 7))
	require.NoError(t, store.Finish(ctx, claimed, ""))
	require.NoError(t, store.Queue(ctx, id, false))
	recovered, err := store.Claim(ctx, "recover")
	require.NoError(t, err)
	require.NotNil(t, recovered)
	require.Equal(t, int64(7), *recovered.AccountID)
	require.NoError(t, store.Finish(ctx, recovered, "验证失败"))
	require.Error(t, store.Queue(ctx, id, true), "failed login must cool down")
	rows, err := store.List(ctx)
	require.NoError(t, err)
	require.Equal(t, "failed", rows[0].Status)
	// Reimport updates the same email row; manual retry keeps its configured group.
	group2, group1 := int64(2), int64(1)
	retryID, err := store.Put(ctx, "retry@example.com", "encrypted-only", CodexLoginOptions{ProxyIPGroupID: &group2})
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE codex_login_jobs SET status='failed',attempts=3,run_after=NOW() WHERE id=$1`, retryID)
	require.NoError(t, err)
	require.NoError(t, store.Queue(ctx, retryID, true))
	retryJob, err := store.Claim(ctx, "manual")
	require.NoError(t, err)
	require.Equal(t, group2, *retryJob.Options.ProxyIPGroupID)
	require.Equal(t, 1, retryJob.Attempts) // Queue reset to zero, Claim increments once.
	require.NoError(t, store.Finish(ctx, retryJob, "failed"))
	sameID, err := store.Put(ctx, "retry@example.com", "encrypted-replacement", CodexLoginOptions{ProxyIPGroupID: &group1})
	require.NoError(t, err)
	require.Equal(t, retryID, sameID)
	var attempts, rowCount int
	require.NoError(t, db.QueryRow(`SELECT attempts FROM codex_login_jobs WHERE id=$1`, retryID).Scan(&attempts))
	require.Zero(t, attempts)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM codex_login_jobs WHERE email='retry@example.com'`).Scan(&rowCount))
	require.Equal(t, 1, rowCount)
	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Equal(t, &group1, listed[0].ProxyIPGroupID)
	_, err = db.Exec(`DELETE FROM codex_login_jobs WHERE id=$1`, retryID)
	require.NoError(t, err)

	_, err = db.Exec(`DELETE FROM accounts WHERE id=7`)
	require.NoError(t, err)
	rows, err = store.List(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)
}
