package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) CreateUserRequestMonitor(ctx context.Context, input *service.OpsCreateUserRequestMonitorRecord) (*service.OpsUserRequestMonitor, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return nil, fmt.Errorf("nil input")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var lockedUserID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, input.UserID).Scan(&lockedUserID); err != nil {
		return nil, err
	}
	var existingID int64
	activeLookupErr := tx.QueryRowContext(ctx, `
SELECT id
FROM ops_user_request_monitors
WHERE user_id = $1
  AND status = 'active'
  AND starts_at <= $2
  AND ends_at > $2
LIMIT 1`, input.UserID, input.CreatedAt.UTC()).Scan(&existingID)
	if activeLookupErr == nil {
		return nil, service.ErrOpsUserRequestMonitorAlreadyActive
	}
	if activeLookupErr != nil && activeLookupErr != sql.ErrNoRows {
		return nil, activeLookupErr
	}
	q := `
INSERT INTO ops_user_request_monitors (
  user_id,
  target_email,
  status,
  duration_seconds,
  max_captures_per_minute,
  sample_rate_percent,
  retention_days,
  created_by,
  created_at,
  starts_at,
  ends_at
) VALUES (
  $1,$2,'active',$3,$4,$5,$6,$7,$8,$9,$10
) RETURNING
  id,
  user_id,
  target_email,
  status,
  duration_seconds,
  max_captures_per_minute,
  sample_rate_percent,
  retention_days,
  created_by,
  created_at,
  starts_at,
  ends_at,
  stopped_at,
  last_capture_at,
  capture_count`
	monitor, err := scanOpsUserRequestMonitor(tx.QueryRowContext(
		ctx,
		q,
		input.UserID,
		strings.TrimSpace(input.TargetEmail),
		input.DurationSeconds,
		input.MaxCapturesPerMinute,
		input.SampleRatePercent,
		input.RetentionDays,
		input.CreatedBy,
		input.CreatedAt.UTC(),
		input.StartsAt.UTC(),
		input.EndsAt.UTC(),
	))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return monitor, nil
}

func (r *opsRepository) ListUserRequestMonitors(ctx context.Context, filter *service.OpsUserRequestMonitorFilter) ([]*service.OpsUserRequestMonitor, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		filter = &service.OpsUserRequestMonitorFilter{}
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where, args := buildOpsUserRequestMonitorsWhere(filter)
	countSQL := "SELECT COUNT(*) FROM ops_user_request_monitors m " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	argsWithLimit := append(args, pageSize, offset)
	q := `
SELECT
  m.id,
  m.user_id,
  m.target_email,
  m.status,
  m.duration_seconds,
  m.max_captures_per_minute,
  m.sample_rate_percent,
  m.retention_days,
  m.created_by,
  m.created_at,
  m.starts_at,
  m.ends_at,
  m.stopped_at,
  m.last_capture_at,
  m.capture_count
FROM ops_user_request_monitors m
` + where + `
ORDER BY m.created_at DESC, m.id DESC
LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	rows, err := r.db.QueryContext(ctx, q, argsWithLimit...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsUserRequestMonitor, 0, pageSize)
	for rows.Next() {
		item, err := scanOpsUserRequestMonitor(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *opsRepository) GetUserRequestMonitorByID(ctx context.Context, id int64) (*service.OpsUserRequestMonitor, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	q := `
SELECT
  id,
  user_id,
  target_email,
  status,
  duration_seconds,
  max_captures_per_minute,
  sample_rate_percent,
  retention_days,
  created_by,
  created_at,
  starts_at,
  ends_at,
  stopped_at,
  last_capture_at,
  capture_count
FROM ops_user_request_monitors
WHERE id = $1
LIMIT 1`
	return scanOpsUserRequestMonitor(r.db.QueryRowContext(ctx, q, id))
}

func (r *opsRepository) GetActiveUserRequestMonitors(ctx context.Context, userID int64, now time.Time) ([]*service.OpsUserRequestMonitor, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if userID <= 0 {
		return []*service.OpsUserRequestMonitor{}, nil
	}
	q := `
SELECT
  id,
  user_id,
  target_email,
  status,
  duration_seconds,
  max_captures_per_minute,
  sample_rate_percent,
  retention_days,
  created_by,
  created_at,
  starts_at,
  ends_at,
  stopped_at,
  last_capture_at,
  capture_count
FROM ops_user_request_monitors
WHERE user_id = $1
  AND status = 'active'
  AND starts_at <= $2
  AND ends_at > $2
ORDER BY created_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, q, userID, now.UTC())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.OpsUserRequestMonitor, 0, 2)
	for rows.Next() {
		item, err := scanOpsUserRequestMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *opsRepository) StopUserRequestMonitor(ctx context.Context, id int64, stoppedAt time.Time) (*service.OpsUserRequestMonitor, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	q := `
UPDATE ops_user_request_monitors
SET status = 'stopped',
    stopped_at = $2
WHERE id = $1
  AND status = 'active'
RETURNING
  id,
  user_id,
  target_email,
  status,
  duration_seconds,
  max_captures_per_minute,
  sample_rate_percent,
  retention_days,
  created_by,
  created_at,
  starts_at,
  ends_at,
  stopped_at,
  last_capture_at,
  capture_count`
	return scanOpsUserRequestMonitor(r.db.QueryRowContext(ctx, q, id, stoppedAt.UTC()))
}

func (r *opsRepository) DeleteUserRequestMonitor(ctx context.Context, id int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("nil ops repository")
	}
	if id <= 0 {
		return false, fmt.Errorf("invalid id")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM ops_user_request_monitors WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *opsRepository) InsertUserRequestCapture(ctx context.Context, input *service.OpsInsertUserRequestCaptureInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return 0, fmt.Errorf("nil input")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	q := `
INSERT INTO ops_user_request_captures (
  monitor_id,
  user_id,
  api_key_id,
  account_id,
  group_id,
  request_id,
  model,
  inbound_endpoint,
  method,
  content_type,
  body,
  body_bytes,
  body_truncated,
  sample_rate_percent,
  capture_minute,
  created_at,
  expires_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
) RETURNING id`
	var id int64
	err = tx.QueryRowContext(
		ctx,
		q,
		input.MonitorID,
		input.UserID,
		opsNullInt64(input.APIKeyID),
		opsNullInt64(input.AccountID),
		opsNullInt64(input.GroupID),
		opsNullString(input.RequestID),
		opsNullString(input.Model),
		opsNullString(input.InboundEndpoint),
		opsNullString(input.Method),
		opsNullString(input.ContentType),
		input.Body,
		input.BodyBytes,
		input.BodyTruncated,
		input.SampleRatePercent,
		input.CaptureMinute.UTC(),
		input.CreatedAt.UTC(),
		input.ExpiresAt.UTC(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `
UPDATE ops_user_request_monitors
SET last_capture_at = $2,
    capture_count = capture_count + 1
WHERE id = $1`, input.MonitorID, input.CreatedAt.UTC())
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *opsRepository) ListUserRequestCaptures(ctx context.Context, filter *service.OpsUserRequestCaptureFilter) ([]*service.OpsUserRequestCapture, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil ops repository")
	}
	if filter == nil || filter.MonitorID <= 0 {
		return []*service.OpsUserRequestCapture{}, 0, nil
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ops_user_request_captures WHERE monitor_id = $1`, filter.MonitorID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	q := `
SELECT
  c.id,
  c.monitor_id,
  c.user_id,
  c.api_key_id,
  c.account_id,
  COALESCE(a.name, ''),
  c.group_id,
  COALESCE(g.name, ''),
  COALESCE(c.request_id, ''),
  COALESCE(c.model, ''),
  COALESCE(c.inbound_endpoint, ''),
  COALESCE(c.method, ''),
  COALESCE(c.content_type, ''),
  '' AS body,
  c.body_bytes,
  c.body_truncated,
  c.sample_rate_percent,
  c.capture_minute,
  c.created_at,
  c.expires_at
FROM ops_user_request_captures c
LEFT JOIN accounts a ON c.account_id = a.id
LEFT JOIN groups g ON c.group_id = g.id
WHERE c.monitor_id = $1
ORDER BY c.created_at DESC, c.id DESC
LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, filter.MonitorID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.OpsUserRequestCapture, 0, pageSize)
	for rows.Next() {
		item, err := scanOpsUserRequestCapture(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *opsRepository) GetUserRequestCapture(ctx context.Context, monitorID, captureID int64) (*service.OpsUserRequestCapture, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if monitorID <= 0 || captureID <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	q := `
SELECT
  c.id,
  c.monitor_id,
  c.user_id,
  c.api_key_id,
  c.account_id,
  COALESCE(a.name, ''),
  c.group_id,
  COALESCE(g.name, ''),
  COALESCE(c.request_id, ''),
  COALESCE(c.model, ''),
  COALESCE(c.inbound_endpoint, ''),
  COALESCE(c.method, ''),
  COALESCE(c.content_type, ''),
  c.body,
  c.body_bytes,
  c.body_truncated,
  c.sample_rate_percent,
  c.capture_minute,
  c.created_at,
  c.expires_at
FROM ops_user_request_captures c
LEFT JOIN accounts a ON c.account_id = a.id
LEFT JOIN groups g ON c.group_id = g.id
WHERE c.monitor_id = $1 AND c.id = $2
LIMIT 1`
	return scanOpsUserRequestCapture(r.db.QueryRowContext(ctx, q, monitorID, captureID))
}

func (r *opsRepository) DeleteUserRequestCapture(ctx context.Context, monitorID, captureID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("nil ops repository")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM ops_user_request_captures WHERE monitor_id = $1 AND id = $2`, monitorID, captureID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *opsRepository) StreamUserRequestCaptures(ctx context.Context, monitorID int64, handle func(*service.OpsUserRequestCapture) error) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if monitorID <= 0 {
		return fmt.Errorf("invalid monitor id")
	}
	if handle == nil {
		return fmt.Errorf("nil capture handler")
	}
	q := `
SELECT
  c.id,
  c.monitor_id,
  c.user_id,
  c.api_key_id,
  c.account_id,
  COALESCE(a.name, ''),
  c.group_id,
  COALESCE(g.name, ''),
  COALESCE(c.request_id, ''),
  COALESCE(c.model, ''),
  COALESCE(c.inbound_endpoint, ''),
  COALESCE(c.method, ''),
  COALESCE(c.content_type, ''),
  c.body,
  c.body_bytes,
  c.body_truncated,
  c.sample_rate_percent,
  c.capture_minute,
  c.created_at,
  c.expires_at
FROM ops_user_request_captures c
LEFT JOIN accounts a ON c.account_id = a.id
LEFT JOIN groups g ON c.group_id = g.id
WHERE c.monitor_id = $1
ORDER BY c.created_at DESC, c.id DESC`
	rows, err := r.db.QueryContext(ctx, q, monitorID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		capture, err := scanOpsUserRequestCapture(rows)
		if err != nil {
			return err
		}
		if err := handle(capture); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r *opsRepository) ExpireUserRequestMonitors(ctx context.Context, now time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE ops_user_request_monitors
SET status = 'expired'
WHERE status = 'active'
  AND ends_at <= $1`, now.UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *opsRepository) DeleteExpiredUserRequestCaptures(ctx context.Context, now time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM ops_user_request_captures WHERE expires_at <= $1`, now.UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func buildOpsUserRequestMonitorsWhere(filter *service.OpsUserRequestMonitorFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 4)
	if filter != nil {
		switch strings.ToLower(strings.TrimSpace(filter.Status)) {
		case service.OpsUserRequestMonitorStatusActive:
			clauses = append(clauses, "m.status = 'active' AND m.ends_at > NOW()")
		case service.OpsUserRequestMonitorStatusExpired:
			clauses = append(clauses, "(m.status = 'expired' OR (m.status = 'active' AND m.ends_at <= NOW()))")
		case service.OpsUserRequestMonitorStatusStopped:
			clauses = append(clauses, "m.status = 'stopped'")
		}
		if q := strings.TrimSpace(filter.UserQuery); q != "" {
			args = append(args, "%"+q+"%")
			clauses = append(clauses, "m.target_email ILIKE $"+itoa(len(args)))
		}
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

type opsUserRequestScanner interface {
	Scan(dest ...any) error
}

func scanOpsUserRequestMonitor(scanner opsUserRequestScanner) (*service.OpsUserRequestMonitor, error) {
	item := &service.OpsUserRequestMonitor{}
	var stoppedAt sql.NullTime
	var lastCaptureAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.TargetEmail,
		&item.Status,
		&item.DurationSeconds,
		&item.MaxCapturesPerMinute,
		&item.SampleRatePercent,
		&item.RetentionDays,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.StartsAt,
		&item.EndsAt,
		&stoppedAt,
		&lastCaptureAt,
		&item.CaptureCount,
	); err != nil {
		return nil, err
	}
	if stoppedAt.Valid {
		t := stoppedAt.Time
		item.StoppedAt = &t
	}
	if lastCaptureAt.Valid {
		t := lastCaptureAt.Time
		item.LastCaptureAt = &t
	}
	return item, nil
}

func scanOpsUserRequestCapture(scanner opsUserRequestScanner) (*service.OpsUserRequestCapture, error) {
	item := &service.OpsUserRequestCapture{}
	var apiKeyID sql.NullInt64
	var accountID sql.NullInt64
	var groupID sql.NullInt64
	if err := scanner.Scan(
		&item.ID,
		&item.MonitorID,
		&item.UserID,
		&apiKeyID,
		&accountID,
		&item.AccountName,
		&groupID,
		&item.GroupName,
		&item.RequestID,
		&item.Model,
		&item.InboundEndpoint,
		&item.Method,
		&item.ContentType,
		&item.Body,
		&item.BodyBytes,
		&item.BodyTruncated,
		&item.SampleRatePercent,
		&item.CaptureMinute,
		&item.CreatedAt,
		&item.ExpiresAt,
	); err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if accountID.Valid {
		v := accountID.Int64
		item.AccountID = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		item.GroupID = &v
	}
	return item, nil
}
