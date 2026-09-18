//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestCodexLoginAtomicRecoveryPostgres(t *testing.T) {
	dsn := os.Getenv("CODEX_LOGIN_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL required")
	}
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("codex_repo_%d", time.Now().UnixNano())
	_, err = admin.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	db, err := sql.Open("postgres", dsn+" search_path="+schema)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`CREATE TABLE accounts(id BIGINT PRIMARY KEY,platform TEXT,type TEXT,parent_account_id BIGINT,deleted_at TIMESTAMPTZ,
 status TEXT,schedulable BOOLEAN,error_message TEXT,credentials JSONB,updated_at TIMESTAMPTZ);
 CREATE TABLE scheduler_outbox(id BIGSERIAL PRIMARY KEY,event_type TEXT,account_id BIGINT);
 INSERT INTO accounts(id,platform,type,status,schedulable,error_message,credentials) VALUES
 (7,'openai','oauth','error',false,'recovery','{"access_token":"old","model_mapping":{"custom":"model"}}');`)
	require.NoError(t, err)
	repo := &accountRepository{sql: db}
	ctx := context.Background()
	require.NoError(t, repo.UpdateCodexLoginCredentials(ctx, 7, "old", map[string]any{"access_token": "new", "refresh_token": "new-rt"}))
	require.Error(t, repo.UpdateCodexLoginCredentials(ctx, 7, "old", map[string]any{"access_token": "stale"}))
	var model, token string
	require.NoError(t, db.QueryRow(`SELECT credentials->'model_mapping'->>'custom',credentials->>'access_token' FROM accounts WHERE id=7`).Scan(&model, &token))
	require.Equal(t, "model", model)
	require.Equal(t, "new", token)
	restored, err := repo.CompleteCodexLoginRecovery(ctx, 7, "new", "wrong-owner")
	require.NoError(t, err)
	require.False(t, restored)
	restored, err = repo.CompleteCodexLoginRecovery(ctx, 7, "new", "recovery")
	require.NoError(t, err)
	require.True(t, restored)
	var status string
	var schedulable bool
	require.NoError(t, db.QueryRow(`SELECT status,schedulable FROM accounts WHERE id=7`).Scan(&status, &schedulable))
	require.Equal(t, "active", status)
	require.True(t, schedulable)
	// A failed outbox insert must roll back the enable, not leave a half-completed recovery.
	_, err = db.Exec(`ALTER TABLE scheduler_outbox ADD COLUMN required_value TEXT NOT NULL DEFAULT 'ok';
 ALTER TABLE scheduler_outbox ALTER COLUMN required_value DROP DEFAULT;
 UPDATE accounts SET status='error',schedulable=false WHERE id=7`)
	require.NoError(t, err)
	restored, err = repo.CompleteCodexLoginRecovery(ctx, 7, "new", "recovery")
	require.Error(t, err)
	require.False(t, restored)
	require.NoError(t, db.QueryRow(`SELECT schedulable FROM accounts WHERE id=7`).Scan(&schedulable))
	require.False(t, schedulable)
}
