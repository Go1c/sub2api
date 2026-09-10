package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelIQRepository struct {
	db *sql.DB
}

func NewChannelIQRepository(db *sql.DB) service.ChannelIQStore {
	return &channelIQRepository{db: db}
}

func (r *channelIQRepository) GetSettings(ctx context.Context) (*service.ChannelIQSettings, error) {
	if _, err := r.db.ExecContext(ctx, `INSERT INTO channel_iq_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT group_ids, auto_enabled, interval_seconds, prompt, model, updated_at
		FROM channel_iq_settings WHERE id = 1
	`)
	var raw []byte
	out := &service.ChannelIQSettings{}
	if err := row.Scan(&raw, &out.AutoEnabled, &out.IntervalSeconds, &out.Prompt, &out.Model, &out.UpdatedAt); err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.GroupIDs)
	}
	if out.GroupIDs == nil {
		out.GroupIDs = []int64{}
	}
	return out, nil
}

func (r *channelIQRepository) SaveSettings(ctx context.Context, settings *service.ChannelIQSettings) error {
	if settings == nil {
		return nil
	}
	raw, err := json.Marshal(settings.GroupIDs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO channel_iq_settings (id, group_ids, auto_enabled, interval_seconds, prompt, model, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE SET
			group_ids = EXCLUDED.group_ids,
			auto_enabled = EXCLUDED.auto_enabled,
			interval_seconds = EXCLUDED.interval_seconds,
			prompt = EXCLUDED.prompt,
			model = EXCLUDED.model,
			updated_at = NOW()
	`, raw, settings.AutoEnabled, settings.IntervalSeconds, settings.Prompt, settings.Model)
	return err
}

func (r *channelIQRepository) GetResults(ctx context.Context, accountIDs []int64) (map[int64]*service.ChannelIQResult, error) {
	out := map[int64]*service.ChannelIQResult{}
	if len(accountIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT account_id, status, test_count, model, reasoning_effort, duration_ms, total_tokens, svg, error, last_run_at
		FROM channel_iq_results
		WHERE account_id = ANY($1)
	`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		item := &service.ChannelIQResult{}
		var lastRun sql.NullTime
		if err := rows.Scan(
			&item.AccountID, &item.Status, &item.TestCount, &item.Model, &item.ReasoningEffort,
			&item.DurationMs, &item.TotalTokens, &item.SVG, &item.Error, &lastRun,
		); err != nil {
			return nil, err
		}
		if lastRun.Valid {
			t := lastRun.Time
			item.LastRunAt = &t
		}
		out[item.AccountID] = item
	}
	return out, rows.Err()
}

func (r *channelIQRepository) MarkRunning(ctx context.Context, accountID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO channel_iq_results (account_id, status, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (account_id) DO UPDATE SET
			status = EXCLUDED.status,
			error = '',
			updated_at = NOW()
	`, accountID, service.ChannelIQStatusRunning)
	return err
}

func (r *channelIQRepository) SaveResult(ctx context.Context, result *service.ChannelIQResult) error {
	if result == nil {
		return nil
	}
	var lastRun any
	if result.LastRunAt != nil {
		lastRun = *result.LastRunAt
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO channel_iq_results (
			account_id, status, test_count, model, reasoning_effort, duration_ms, total_tokens, svg, error, last_run_at, updated_at
		) VALUES ($1, $2, 1, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (account_id) DO UPDATE SET
			status = EXCLUDED.status,
			test_count = channel_iq_results.test_count + 1,
			model = EXCLUDED.model,
			reasoning_effort = EXCLUDED.reasoning_effort,
			duration_ms = EXCLUDED.duration_ms,
			total_tokens = EXCLUDED.total_tokens,
			svg = EXCLUDED.svg,
			error = EXCLUDED.error,
			last_run_at = EXCLUDED.last_run_at,
			updated_at = NOW()
	`, result.AccountID, result.Status, result.Model, result.ReasoningEffort, result.DurationMs, result.TotalTokens, result.SVG, result.Error, lastRun)
	return err
}
