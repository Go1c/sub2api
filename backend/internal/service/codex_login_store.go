package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type CodexLoginOptions struct {
	GroupIDs       []int64 `json:"group_ids"`
	ProxyID        *int64  `json:"proxy_id"`
	ProxyIPGroupID *int64  `json:"proxy_ip_group_id"`
}

type CodexLoginJob struct {
	ProxyID        *int64            `json:"proxy_id,omitempty"`
	ProxyIPGroupID *int64            `json:"proxy_ip_group_id,omitempty"`
	ID             int64             `json:"id"`
	Email          string            `json:"email"`
	AccountID      *int64            `json:"account_id"`
	Status         string            `json:"status"`
	ErrorMessage   string            `json:"error_message"`
	Attempts       int               `json:"attempts"`
	Encrypted      string            `json:"-"`
	Options        CodexLoginOptions `json:"-"`
	Lease          string            `json:"-"`
}

type codexLoginStore struct{ db *sql.DB }

func (s *codexLoginStore) Put(ctx context.Context, email, encrypted string, options CodexLoginOptions) (int64, error) {
	raw, err := json.Marshal(options)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `INSERT INTO codex_login_jobs(email,material_encrypted,options) VALUES($1,$2,$3)
 ON CONFLICT(email) DO UPDATE SET material_encrypted=EXCLUDED.material_encrypted,options=EXCLUDED.options,status='queued',error_message='',attempts=0,run_after=NOW(),updated_at=NOW()
 WHERE codex_login_jobs.status NOT IN ('queued','running') RETURNING id`, email, encrypted, string(raw)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("该账号正在处理，请勿重复提交")
	}
	return id, err
}

func (s *codexLoginStore) List(ctx context.Context) ([]CodexLoginJob, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,email,account_id,status,error_message,attempts,options FROM codex_login_jobs ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []CodexLoginJob{}
	for rows.Next() {
		var j CodexLoginJob
		var raw []byte
		if err = rows.Scan(&j.ID, &j.Email, &j.AccountID, &j.Status, &j.ErrorMessage, &j.Attempts, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &j.Options); err != nil {
			return nil, err
		}
		j.ProxyID, j.ProxyIPGroupID = j.Options.ProxyID, j.Options.ProxyIPGroupID
		result = append(result, j)
	}
	return result, rows.Err()
}

func (s *codexLoginStore) Claim(ctx context.Context, lease string) (*CodexLoginJob, error) {
	// Repair a crash between stopping account scheduling and enqueuing its recovery.
	_, err := s.db.ExecContext(ctx, `UPDATE codex_login_jobs AS j SET status='queued',updated_at=NOW()
 FROM accounts AS a WHERE j.account_id=a.id AND j.status='succeeded' AND a.deleted_at IS NULL AND a.status='error' AND a.error_message=$1`, codexRecoveryReason)
	if err != nil {
		return nil, err
	}
	// Unknown interrupted results require explicit retry rather than endless repeated logins.
	_, err = s.db.ExecContext(ctx, `UPDATE codex_login_jobs SET status='failed',error_message='服务中断，请重试',lease_token=NULL,lease_until=NULL,updated_at=NOW() WHERE status='running' AND lease_until<NOW()`)
	if err != nil {
		return nil, err
	}
	var j CodexLoginJob
	var raw []byte
	err = s.db.QueryRowContext(ctx, `UPDATE codex_login_jobs SET status='running',lease_token=$1,lease_until=NOW()+INTERVAL '5 minutes',attempts=attempts+1,updated_at=NOW()
 WHERE id=(SELECT id FROM codex_login_jobs WHERE status='queued' AND run_after<=NOW() ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1)
 RETURNING id,email,account_id,material_encrypted,options,attempts`, lease).Scan(&j.ID, &j.Email, &j.AccountID, &j.Encrypted, &raw, &j.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &j.Options); err != nil {
		return nil, err
	}
	j.Lease = lease
	return &j, nil
}

func (s *codexLoginStore) Finish(ctx context.Context, j *CodexLoginJob, message string) error {
	status := "succeeded"
	if message != "" {
		status = "failed"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE codex_login_jobs SET status=$3::text,error_message=$4,lease_token=NULL,lease_until=NULL,run_after=CASE WHEN $3::text='succeeded' THEN NOW() ELSE NOW()+INTERVAL '5 minutes' END,attempts=CASE WHEN $3::text='succeeded' THEN 0 ELSE attempts END,updated_at=NOW() WHERE id=$1 AND lease_token=$2`, j.ID, j.Lease, status, message)
	return err
}

func (s *codexLoginStore) Bind(ctx context.Context, j *CodexLoginJob, accountID int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE codex_login_jobs SET account_id=$3 WHERE id=$1 AND lease_token=$2`, j.ID, j.Lease, accountID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("登录任务租约已失效")
	}
	return nil
}

func (s *codexLoginStore) FindAccount(ctx context.Context, id int64) (*CodexLoginJob, error) {
	var j CodexLoginJob
	err := s.db.QueryRowContext(ctx, `SELECT id,status,attempts FROM codex_login_jobs WHERE account_id=$1`, id).Scan(&j.ID, &j.Status, &j.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &j, err
}

func (s *codexLoginStore) Queue(ctx context.Context, id int64, manual bool) error {
	query := `UPDATE codex_login_jobs SET status='queued',error_message='',updated_at=NOW() WHERE id=$1 AND status IN ('succeeded','failed') AND attempts<4`
	if manual {
		query = `UPDATE codex_login_jobs SET status='queued',error_message='',attempts=0,updated_at=NOW() WHERE id=$1 AND status IN ('succeeded','failed') AND run_after<=NOW()`
	}
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if manual && n == 0 {
		return errors.New("任务正在执行或冷却，请稍后重试")
	}
	return nil
}
