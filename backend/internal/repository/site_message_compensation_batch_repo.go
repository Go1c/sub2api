package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type siteMessageCompensationBatchRepository struct {
	db *sql.DB
}

func NewSiteMessageCompensationBatchRepository(db *sql.DB) service.SiteMessageCompensationBatchRepository {
	return &siteMessageCompensationBatchRepository{db: db}
}

func (r *siteMessageCompensationBatchRepository) Create(ctx context.Context, batch *service.SiteMessageCompensationBatch) error {
	codesJSON, err := json.Marshal(batch.Codes)
	if err != nil {
		return fmt.Errorf("marshal compensation codes: %w", err)
	}
	resultsJSON, err := json.Marshal(batch.Results)
	if err != nil {
		return fmt.Errorf("marshal compensation results: %w", err)
	}
	messageIDsJSON, err := json.Marshal(batch.MessageIDs)
	if err != nil {
		return fmt.Errorf("marshal message ids: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO site_message_compensation_batches (
	batch_id, subject, content, mode, audience,
	recipient_count, success_count, failed_count, amount, code_count,
	operator_email, sent_at, codes, results, message_ids
) VALUES (
	$1, $2, $3, $4, $5,
	$6, $7, $8, $9, $10,
	$11, $12, $13::jsonb, $14::jsonb, $15::jsonb
)`,
		batch.ID,
		batch.Subject,
		batch.Content,
		batch.Mode,
		batch.Audience,
		batch.RecipientCount,
		batch.SuccessCount,
		batch.FailedCount,
		batch.Amount,
		batch.CodeCount,
		batch.Operator,
		batch.SentAt,
		string(codesJSON),
		string(resultsJSON),
		string(messageIDsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert site message compensation batch: %w", err)
	}
	return nil
}

func (r *siteMessageCompensationBatchRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.SiteMessageCompensationBatch, *pagination.PaginationResult, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_message_compensation_batches`).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count site message compensation batches: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT
	batch_id, subject, content, mode, audience,
	recipient_count, success_count, failed_count, amount, code_count,
	operator_email, sent_at, codes::text, results::text, message_ids::text
FROM site_message_compensation_batches
ORDER BY sent_at DESC, id DESC
LIMIT $1 OFFSET $2`,
		params.Limit(),
		params.Offset(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list site message compensation batches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SiteMessageCompensationBatch, 0)
	for rows.Next() {
		var item service.SiteMessageCompensationBatch
		var codesJSON, resultsJSON, messageIDsJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Subject,
			&item.Content,
			&item.Mode,
			&item.Audience,
			&item.RecipientCount,
			&item.SuccessCount,
			&item.FailedCount,
			&item.Amount,
			&item.CodeCount,
			&item.Operator,
			&item.SentAt,
			&codesJSON,
			&resultsJSON,
			&messageIDsJSON,
		); err != nil {
			return nil, nil, fmt.Errorf("scan site message compensation batch: %w", err)
		}
		if err := json.Unmarshal([]byte(codesJSON), &item.Codes); err != nil {
			return nil, nil, fmt.Errorf("unmarshal compensation codes: %w", err)
		}
		if err := json.Unmarshal([]byte(resultsJSON), &item.Results); err != nil {
			return nil, nil, fmt.Errorf("unmarshal compensation results: %w", err)
		}
		if err := json.Unmarshal([]byte(messageIDsJSON), &item.MessageIDs); err != nil {
			return nil, nil, fmt.Errorf("unmarshal message ids: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate site message compensation batches: %w", err)
	}

	return items, paginationResultFromTotal(total, params), nil
}
