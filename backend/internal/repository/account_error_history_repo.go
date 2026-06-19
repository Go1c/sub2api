package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/accounterrorhistory"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// accountErrorHistoryRepository 实现 service.AccountErrorHistoryRepository。
//
// 选型说明（与 channel_monitor_repo 一致的分层）：
//   - 查询（ListRecent）走 ent，复用项目模型互转；
//   - 写入的去重折叠 / 节流 / 裁剪走原生 SQL 单语句，保证命中索引、减少往返。
//
// 这是 best-effort 监控数据：偶发并发竞态导致多写可接受，不引入重锁。
type accountErrorHistoryRepository struct {
	client *dbent.Client
	db     *sql.DB
	// minInterval 每账号最小写入间隔；默认 accountErrorHistoryMinInterval，
	// 测试可调（例如设 0 关闭节流以验证裁剪上限）。
	minInterval time.Duration
}

// NewAccountErrorHistoryRepository 创建仓储实例。
func NewAccountErrorHistoryRepository(client *dbent.Client, db *sql.DB) service.AccountErrorHistoryRepository {
	return &accountErrorHistoryRepository{client: client, db: db, minInterval: accountErrorHistoryMinInterval}
}

const (
	// accountErrorHistoryMaxRows 每账号保留的最近行数上限，超出在插入后裁剪。
	accountErrorHistoryMaxRows = 50
	// accountErrorHistoryDedupWindow 去重折叠窗口：窗口内同维度只折叠为一行。
	accountErrorHistoryDedupWindow = 60 * time.Second
	// accountErrorHistoryMinInterval 每账号最小写入间隔：间隔内的「新」错误被节流丢弃，
	// 折叠命中（同维度重复）不受此限。
	accountErrorHistoryMinInterval = 2 * time.Second
)

// dedupSQL：60s 窗口内命中同 (account_id, source, upstream_status_code, fingerprint) 的最近一行，
// 则折叠 dup_count+1 并刷新 created_at/updated_at（刷新 created_at 让该错误在「最近」视图上浮）。
// upstream_status_code 可空，用 IS NOT DISTINCT FROM 正确处理 NULL=NULL。
const accountErrorHistoryDedupSQL = `
UPDATE account_error_histories
SET dup_count = dup_count + 1, created_at = NOW(), updated_at = NOW()
WHERE id = (
    SELECT id FROM account_error_histories
    WHERE account_id = $1
      AND source = $2
      AND upstream_status_code IS NOT DISTINCT FROM $3
      AND fingerprint = $4
      AND created_at > NOW() - ($5::int || ' seconds')::interval
    ORDER BY created_at DESC
    LIMIT 1
)
RETURNING id
`

// insertSQL 插入新行。
const accountErrorHistoryInsertSQL = `
INSERT INTO account_error_histories
    (account_id, user_id, user_email, model, upstream_status_code, source, message, fingerprint)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

// throttleSQL 取该账号最近一行的 created_at，用于最小写入间隔节流判断。
const accountErrorHistoryThrottleSQL = `
SELECT created_at FROM account_error_histories
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT 1
`

// trimSQL 裁剪：删除该账号超出最近 accountErrorHistoryMaxRows 条的旧行。
const accountErrorHistoryTrimSQL = `
DELETE FROM account_error_histories
WHERE account_id = $1
  AND id NOT IN (
      SELECT id FROM account_error_histories
      WHERE account_id = $1
      ORDER BY created_at DESC, id DESC
      LIMIT $2
  )
`

func (r *accountErrorHistoryRepository) Record(ctx context.Context, row *service.AccountErrorRow) error {
	if row == nil {
		return nil
	}

	statusCode := nullIntFromPtr(row.UpstreamStatusCode)
	windowSeconds := int(accountErrorHistoryDedupWindow / time.Second)

	// 1) 去重折叠：命中则只刷新，直接返回。
	var hitID int64
	err := r.db.QueryRowContext(ctx, accountErrorHistoryDedupSQL,
		row.AccountID, row.Source, statusCode, row.Fingerprint, windowSeconds,
	).Scan(&hitID)
	if err == nil {
		return nil // 折叠命中
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("dedup account error history: %w", err)
	}

	// 2) 节流：未命中折叠时，若距该账号最近一条写入不足最小间隔，best-effort 丢弃。
	throttled, err := r.shouldThrottle(ctx, row.AccountID)
	if err != nil {
		return err
	}
	if throttled {
		return nil
	}

	// 3) 插入新行。
	if _, err := r.db.ExecContext(ctx, accountErrorHistoryInsertSQL,
		row.AccountID,
		nullInt64(row.UserID),
		nullStringFromPtr(row.UserEmail),
		nullStringFromPtr(row.Model),
		statusCode,
		row.Source,
		row.Message,
		row.Fingerprint,
	); err != nil {
		return fmt.Errorf("insert account error history: %w", err)
	}

	// 4) 裁剪：超出上限的旧行删除（best-effort，失败不影响已写入的行）。
	if _, err := r.db.ExecContext(ctx, accountErrorHistoryTrimSQL, row.AccountID, accountErrorHistoryMaxRows); err != nil {
		return fmt.Errorf("trim account error history: %w", err)
	}
	return nil
}

// shouldThrottle 返回距该账号最近一条写入是否不足最小间隔。
func (r *accountErrorHistoryRepository) shouldThrottle(ctx context.Context, accountID int64) (bool, error) {
	var last time.Time
	err := r.db.QueryRowContext(ctx, accountErrorHistoryThrottleSQL, accountID).Scan(&last)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("throttle check account error history: %w", err)
	}
	return time.Since(last) < r.minInterval, nil
}

// ListRecent 按 created_at 倒序返回某账号最近 limit 条历史。
func (r *accountErrorHistoryRepository) ListRecent(ctx context.Context, accountID int64, limit int) ([]*service.AccountErrorHistoryEntry, error) {
	rows, err := r.client.AccountErrorHistory.Query().
		Where(accounterrorhistory.AccountIDEQ(accountID)).
		Order(dbent.Desc(accounterrorhistory.FieldCreatedAt), dbent.Desc(accounterrorhistory.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account error history: %w", err)
	}
	out := make([]*service.AccountErrorHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &service.AccountErrorHistoryEntry{
			ID:                 row.ID,
			AccountID:          row.AccountID,
			UserID:             row.UserID,
			UserEmail:          row.UserEmail,
			Model:              row.Model,
			UpstreamStatusCode: row.UpstreamStatusCode,
			Source:             string(row.Source),
			Message:            row.Message,
			DupCount:           row.DupCount,
			CreatedAt:          row.CreatedAt,
		})
	}
	return out, nil
}

// nullIntFromPtr 把 *int 解为 sql.NullInt64（nil -> NULL）。
func nullIntFromPtr(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

// nullStringFromPtr 把 *string 解为 sql.NullString（nil -> NULL）。
func nullStringFromPtr(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}
