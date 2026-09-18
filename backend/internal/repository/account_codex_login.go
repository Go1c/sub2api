package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Merge only authentication fields, preserving simultaneous edits to model mappings/settings.
func (r *accountRepository) UpdateCodexLoginCredentials(ctx context.Context, id int64, previousToken string, credentials map[string]any) error {
	raw, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	result, err := r.sql.ExecContext(ctx, `WITH updated AS (
 UPDATE accounts SET credentials=COALESCE(credentials,'{}'::jsonb)||$2::jsonb,updated_at=NOW()
 WHERE id=$1 AND deleted_at IS NULL AND platform='openai' AND type='oauth' AND parent_account_id IS NULL
 AND COALESCE(credentials->>'access_token','')=$3 RETURNING id)
 INSERT INTO scheduler_outbox(event_type,account_id) SELECT $4,id FROM updated`, id, string(raw), previousToken, service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("账号凭据已变更，未覆盖")
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

// FindCodexLoginAccount binds login material to the same user and Team, not just a shared Team ID.
func (r *accountRepository) FindCodexLoginAccount(ctx context.Context, email, workspace string) (*service.Account, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT id FROM accounts WHERE deleted_at IS NULL AND platform='openai' AND type='oauth'
 AND parent_account_id IS NULL AND lower(credentials->>'email')=$1 AND credentials->>'chatgpt_account_id'=$2 ORDER BY id LIMIT 2`, email, workspace)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > 1 {
		return nil, errors.New("同一邮箱与 Team 存在多个账号，请先消除重复")
	}
	return r.GetByID(ctx, ids[0])
}

// CompleteCodexLoginRecovery only re-enables a row still owned by this recovery.
// Manual disable/error changes or a concurrently replaced token are never overwritten.
func (r *accountRepository) CompleteCodexLoginRecovery(ctx context.Context, id int64, token, reason string) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `WITH restored AS (
 UPDATE accounts SET status='active',schedulable=TRUE,error_message='',updated_at=NOW()
 WHERE id=$1 AND deleted_at IS NULL AND status='error' AND error_message=$2 AND credentials->>'access_token'=$3
 RETURNING id)
 INSERT INTO scheduler_outbox(event_type,account_id) SELECT $4,id FROM restored`, id, reason, token, service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return false, err
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return true, nil
}
