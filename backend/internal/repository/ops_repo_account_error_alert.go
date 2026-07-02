package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) ListAccountErrorAlertCandidates(ctx context.Context, filter *service.OpsAccountErrorAlertCandidateFilter) ([]*service.OpsAccountErrorAlertCandidate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return []*service.OpsAccountErrorAlertCandidate{}, nil
	}
	start := filter.StartTime.UTC()
	end := filter.EndTime.UTC()
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return []*service.OpsAccountErrorAlertCandidate{}, nil
	}
	minCount := filter.MinErrorCount
	if minCount <= 0 {
		minCount = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	const q = `
WITH event_errors AS (
  SELECT
    (ev->>'account_id')::bigint AS account_id,
    NULLIF(ev->>'account_name', '') AS account_name,
    CASE
      WHEN COALESCE(ev->>'upstream_status_code', '') ~ '^[0-9]+$' THEN (ev->>'upstream_status_code')::int
      ELSE COALESCE(o.upstream_status_code, o.status_code, 0)
    END AS status_code,
    CASE
      WHEN COALESCE(ev->>'at_unix_ms', '') ~ '^[0-9]+$' THEN to_timestamp((ev->>'at_unix_ms')::double precision / 1000.0)
      ELSE o.created_at
    END AS occurred_at,
    COALESCE(NULLIF(ev->>'message', ''), NULLIF(ev->>'detail', ''), NULLIF(o.upstream_error_message, ''), NULLIF(o.error_message, ''), '') AS error_message
  FROM ops_error_logs o
  CROSS JOIN LATERAL jsonb_array_elements(COALESCE(NULLIF(o.upstream_errors, 'null'::jsonb), '[]'::jsonb)) AS ev
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.is_count_tokens, FALSE) = FALSE
    AND COALESCE(ev->>'account_id', '') ~ '^[0-9]+$'
),
row_errors AS (
  SELECT
    o.account_id AS account_id,
    NULL::text AS account_name,
    COALESCE(o.upstream_status_code, o.status_code, 0) AS status_code,
    o.created_at AS occurred_at,
    COALESCE(NULLIF(o.upstream_error_message, ''), NULLIF(o.error_message, ''), '') AS error_message
  FROM ops_error_logs o
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.is_count_tokens, FALSE) = FALSE
    AND o.account_id IS NOT NULL
    AND jsonb_array_length(COALESCE(NULLIF(o.upstream_errors, 'null'::jsonb), '[]'::jsonb)) = 0
),
base_errors AS (
  SELECT * FROM event_errors
  UNION ALL
  SELECT * FROM row_errors
),
account_totals AS (
  SELECT account_id, COUNT(*)::bigint AS error_count, MAX(occurred_at) AS latest_at
  FROM base_errors
  GROUP BY account_id
  HAVING COUNT(*) >= $3
),
status_counts AS (
  SELECT b.account_id, b.status_code, COUNT(*)::bigint AS status_count, MAX(b.occurred_at) AS latest_at
  FROM base_errors b
  JOIN account_totals t ON t.account_id = b.account_id
  GROUP BY b.account_id, b.status_code
),
top_status AS (
  SELECT DISTINCT ON (account_id) account_id, status_code
  FROM status_counts
  ORDER BY account_id, status_count DESC, latest_at DESC, status_code DESC
),
latest_messages AS (
  SELECT DISTINCT ON (b.account_id)
    b.account_id,
    NULLIF(b.account_name, '') AS account_name,
    COALESCE(NULLIF(b.error_message, ''), '') AS error_message
  FROM base_errors b
  JOIN account_totals t ON t.account_id = b.account_id
  ORDER BY b.account_id, b.occurred_at DESC
)
SELECT
  t.account_id,
  COALESCE(NULLIF(a.name, ''), lm.account_name, 'Account #' || t.account_id::text) AS account_name,
  COALESCE(ts.status_code, 0) AS status_code,
  t.error_count,
  t.latest_at,
  COALESCE(lm.error_message, '') AS error_message
FROM account_totals t
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN top_status ts ON ts.account_id = t.account_id
LEFT JOIN latest_messages lm ON lm.account_id = t.account_id
ORDER BY t.error_count DESC, t.latest_at DESC
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, q, start, end, minCount, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*service.OpsAccountErrorAlertCandidate, 0, limit)
	for rows.Next() {
		var item service.OpsAccountErrorAlertCandidate
		var accountName sql.NullString
		var statusCode sql.NullInt64
		var errorMessage sql.NullString
		if err := rows.Scan(
			&item.AccountID,
			&accountName,
			&statusCode,
			&item.ErrorCount,
			&item.LatestAt,
			&errorMessage,
		); err != nil {
			return nil, err
		}
		item.AccountName = strings.TrimSpace(accountName.String)
		if item.AccountName == "" {
			item.AccountName = fmt.Sprintf("Account #%d", item.AccountID)
		}
		if statusCode.Valid {
			item.StatusCode = int(statusCode.Int64)
		}
		item.LatestAt = item.LatestAt.UTC()
		item.ErrorMessage = strings.TrimSpace(errorMessage.String)
		out = append(out, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *opsRepository) ListAccountErrorAlertTopUsers(ctx context.Context, filter *service.OpsAccountErrorAlertTopUserFilter) ([]*service.OpsAccountErrorAlertTopUser, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return []*service.OpsAccountErrorAlertTopUser{}, nil
	}
	start := filter.StartTime.UTC()
	end := filter.EndTime.UTC()
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return []*service.OpsAccountErrorAlertTopUser{}, nil
	}
	minCount := filter.MinErrorCount
	if minCount <= 0 {
		minCount = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		return []*service.OpsAccountErrorAlertTopUser{}, nil
	}
	if limit > 10 {
		limit = 10
	}

	const q = `
WITH event_errors AS (
  SELECT
    (ev->>'account_id')::bigint AS account_id,
    o.user_id AS user_id,
    CASE
      WHEN COALESCE(ev->>'at_unix_ms', '') ~ '^[0-9]+$' THEN to_timestamp((ev->>'at_unix_ms')::double precision / 1000.0)
      ELSE o.created_at
    END AS occurred_at
  FROM ops_error_logs o
  CROSS JOIN LATERAL jsonb_array_elements(COALESCE(NULLIF(o.upstream_errors, 'null'::jsonb), '[]'::jsonb)) AS ev
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.is_count_tokens, FALSE) = FALSE
    AND o.user_id IS NOT NULL
    AND COALESCE(ev->>'account_id', '') ~ '^[0-9]+$'
),
row_errors AS (
  SELECT
    o.account_id AS account_id,
    o.user_id AS user_id,
    o.created_at AS occurred_at
  FROM ops_error_logs o
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.is_count_tokens, FALSE) = FALSE
    AND o.account_id IS NOT NULL
    AND o.user_id IS NOT NULL
    AND jsonb_array_length(COALESCE(NULLIF(o.upstream_errors, 'null'::jsonb), '[]'::jsonb)) = 0
),
base_errors AS (
  SELECT * FROM event_errors
  UNION ALL
  SELECT * FROM row_errors
),
account_totals AS (
  SELECT account_id, COUNT(*)::bigint AS error_count
  FROM base_errors
  GROUP BY account_id
  HAVING COUNT(*) >= $3
),
user_totals AS (
  SELECT b.user_id, COUNT(*)::bigint AS error_count, MAX(b.occurred_at) AS latest_at
  FROM base_errors b
  JOIN account_totals t ON t.account_id = b.account_id
  GROUP BY b.user_id
)
SELECT
  u.email AS user_email,
  ut.error_count
FROM user_totals ut
JOIN users u ON u.id = ut.user_id AND NULLIF(u.email, '') IS NOT NULL
ORDER BY ut.error_count DESC, ut.latest_at DESC, ut.user_id ASC
LIMIT $4`

	rows, err := r.db.QueryContext(ctx, q, start, end, minCount, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*service.OpsAccountErrorAlertTopUser, 0, limit)
	for rows.Next() {
		var item service.OpsAccountErrorAlertTopUser
		var userEmail sql.NullString
		if err := rows.Scan(
			&userEmail,
			&item.ErrorCount,
		); err != nil {
			return nil, err
		}
		item.UserEmail = strings.TrimSpace(userEmail.String)
		if item.UserEmail == "" {
			continue
		}
		out = append(out, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
