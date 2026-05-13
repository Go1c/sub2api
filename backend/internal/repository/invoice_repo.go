package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type invoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) service.InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) Create(ctx context.Context, req *service.InvoiceRequest) error {
	if req == nil {
		return nil
	}
	err := r.db.QueryRowContext(ctx, `
WITH next_id AS (
	SELECT nextval('invoice_requests_id_seq') AS id
)
INSERT INTO invoice_requests (
	id, order_no, user_id, user_email, title, tax_number, amount, recipient_email, status
)
SELECT
	id,
	'INV' || lpad(id::text, 8, '0'),
	$1, $2, $3, $4, $5, $6, $7
FROM next_id
RETURNING id, order_no, created_at, updated_at
`, req.UserID, req.UserEmail, req.Title, req.TaxNumber, req.Amount, req.RecipientEmail, service.InvoiceStatusProcessing).
		Scan(&req.ID, &req.OrderNo, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice request: %w", err)
	}
	req.Status = service.InvoiceStatusProcessing
	return nil
}

func (r *invoiceRepository) GetByID(ctx context.Context, id int64) (*service.InvoiceRequest, error) {
	rows, err := r.db.QueryContext(ctx, invoiceSelectSQL()+` WHERE ir.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrInvoiceNotFound
	}
	item, err := scanInvoiceRows(rows)
	if err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func (r *invoiceRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.InvoiceRequest, *pagination.PaginationResult, error) {
	total, err := countInvoiceRows(ctx, r.db, " WHERE ir.user_id = $1", []any{userID})
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, invoiceSelectSQL()+`
 WHERE ir.user_id = $1
 ORDER BY ir.created_at DESC, ir.id DESC
 LIMIT $2 OFFSET $3`, userID, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanInvoiceList(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *invoiceRepository) ListAdmin(ctx context.Context, params pagination.PaginationParams, filter service.InvoiceListFilter) ([]service.InvoiceRequest, *pagination.PaginationResult, error) {
	where, args := invoiceListWhere(filter)
	total, err := countInvoiceRows(ctx, r.db, where, args)
	if err != nil {
		return nil, nil, err
	}
	argsWithPage := append(append([]any{}, args...), params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx, invoiceSelectSQL()+where+fmt.Sprintf(`
 ORDER BY ir.created_at DESC, ir.id DESC
 LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2), argsWithPage...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanInvoiceList(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *invoiceRepository) SumActiveAmountByUser(ctx context.Context, userID int64) (float64, error) {
	return r.sumAmountByUser(ctx, userID, []string{service.InvoiceStatusProcessing, service.InvoiceStatusCompleted})
}

func (r *invoiceRepository) SumCompletedAmountByUser(ctx context.Context, userID int64) (float64, error) {
	return r.sumAmountByUser(ctx, userID, []string{service.InvoiceStatusCompleted})
}

func (r *invoiceRepository) sumAmountByUser(ctx context.Context, userID int64, statuses []string) (float64, error) {
	args := []any{userID}
	statusPlaceholders := make([]string, 0, len(statuses))
	for i, status := range statuses {
		args = append(args, status)
		statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", i+2))
	}
	var total sql.NullFloat64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM invoice_requests WHERE user_id = $1 AND status IN (`+strings.Join(statusPlaceholders, ",")+`)`,
		args...,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func (r *invoiceRepository) MarkCompletedAndDeduct(ctx context.Context, id int64, input service.CompleteInvoicePersistInput) (*service.InvoiceRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice completion tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, invoiceSelectSQL()+`
 WHERE ir.id = $1 AND ir.status = $2
 FOR UPDATE OF ir`, id, service.InvoiceStatusProcessing)
	if err != nil {
		return nil, err
	}
	item, scanErr := firstInvoiceFromRows(rows)
	if closeErr := rows.Close(); closeErr != nil {
		scanErr = errors.Join(scanErr, closeErr)
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if item == nil {
		return nil, service.ErrInvoiceInvalidStatus
	}

	rows, err = tx.QueryContext(ctx, `
UPDATE invoice_requests
SET status = $2,
    file_name = $3,
    file_path = $4,
    file_size = $5,
    content_type = $6,
    tax_rate = $7,
    tax_amount = $8,
    failure_reason = '',
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING completed_at, updated_at
`, id, service.InvoiceStatusCompleted, input.FileName, input.FilePath, input.FileSize, input.ContentType, input.TaxRate, input.TaxAmount)
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		var completedAt time.Time
		if err := rows.Scan(&completedAt, &item.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		item.CompletedAt = &completedAt
	} else {
		_ = rows.Close()
		return nil, service.ErrInvoiceInvalidStatus
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1 WHERE id = $2`, input.TaxAmount, item.UserID); err != nil {
		return nil, fmt.Errorf("deduct invoice tax: %w", err)
	}
	if input.TaxAmount > 0 {
		deductionCode := strings.TrimSpace(input.DeductionCode)
		if deductionCode == "" {
			deductionCode, err = service.GenerateRedeemCode()
			if err != nil {
				return nil, fmt.Errorf("generate invoice tax deduction code: %w", err)
			}
		}
		deductionNotes := strings.TrimSpace(input.DeductionNotes)
		if deductionNotes == "" {
			deductionNotes = fmt.Sprintf("发票税点扣除 %s，税率 %.2f%%，扣除金额 %.2f", item.OrderNo, input.TaxRate*100, input.TaxAmount)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at)
VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW())
`, deductionCode, service.AdjustmentTypeAdminBalance, -input.TaxAmount, service.StatusUsed, item.UserID, deductionNotes); err != nil {
			return nil, fmt.Errorf("record invoice tax deduction: %w", err)
		}
	}

	item.Status = service.InvoiceStatusCompleted
	item.FileName = input.FileName
	item.FilePath = input.FilePath
	item.FileSize = input.FileSize
	item.ContentType = input.ContentType
	item.TaxRate = input.TaxRate
	item.TaxAmount = input.TaxAmount
	item.FailureReason = ""
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice completion tx: %w", err)
	}
	return item, nil
}

func (r *invoiceRepository) MarkFailed(ctx context.Context, id int64, reason string) (*service.InvoiceRequest, error) {
	rows, err := r.db.QueryContext(ctx, invoiceSelectSQL()+`
 WHERE ir.id = $1
 FOR UPDATE OF ir`, id)
	if err != nil {
		return nil, err
	}
	item, scanErr := firstInvoiceFromRows(rows)
	if closeErr := rows.Close(); closeErr != nil {
		scanErr = errors.Join(scanErr, closeErr)
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if item == nil {
		return nil, service.ErrInvoiceNotFound
	}
	if item.Status == service.InvoiceStatusCompleted {
		return nil, service.ErrInvoiceInvalidStatus
	}
	_, err = r.db.ExecContext(ctx, `
UPDATE invoice_requests
SET status = $2, failure_reason = $3, updated_at = NOW()
WHERE id = $1
`, id, service.InvoiceStatusFailed, strings.TrimSpace(reason))
	if err != nil {
		return nil, err
	}
	item.Status = service.InvoiceStatusFailed
	item.FailureReason = strings.TrimSpace(reason)
	return item, nil
}

func invoiceListWhere(filter service.InvoiceListFilter) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.UserID > 0 {
		args = append(args, filter.UserID)
		clauses = append(clauses, fmt.Sprintf("ir.user_id = $%d", len(args)))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("ir.status = $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf(`(
			ir.order_no ILIKE $%d OR ir.user_email ILIKE $%d OR ir.title ILIKE $%d OR
			ir.tax_number ILIKE $%d OR ir.recipient_email ILIKE $%d
		)`, idx, idx, idx, idx, idx))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func countInvoiceRows(ctx context.Context, db *sql.DB, where string, args []any) (int64, error) {
	var total int64
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invoice_requests ir`+where, args...).Scan(&total)
	return total, err
}

func invoiceSelectSQL() string {
	return `
SELECT
	ir.id, ir.order_no, ir.user_id, ir.user_email, ir.title, ir.tax_number,
	ir.amount, ir.recipient_email, ir.status, ir.file_name, ir.file_path,
	ir.file_size, ir.content_type, ir.tax_rate, ir.tax_amount, ir.failure_reason,
	ir.created_at, ir.updated_at, ir.completed_at,
	u.email, u.username, u.total_recharged,
	COALESCE((
		SELECT SUM(ir2.amount)
		FROM invoice_requests ir2
		WHERE ir2.user_id = ir.user_id AND ir2.status = 'completed'
	), 0) AS user_completed_invoice_amount
FROM invoice_requests ir
LEFT JOIN users u ON u.id = ir.user_id`
}

func scanInvoiceList(rows *sql.Rows) ([]service.InvoiceRequest, error) {
	items := make([]service.InvoiceRequest, 0)
	for rows.Next() {
		item, err := scanInvoiceRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func firstInvoiceFromRows(rows *sql.Rows) (*service.InvoiceRequest, error) {
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	item, err := scanInvoiceRows(rows)
	if err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func scanInvoiceRows(rows *sql.Rows) (*service.InvoiceRequest, error) {
	var (
		item                   service.InvoiceRequest
		completedAt            sql.NullTime
		userEmail              sql.NullString
		username               sql.NullString
		totalRecharged         sql.NullFloat64
		completedInvoiceAmount sql.NullFloat64
	)
	if err := rows.Scan(
		&item.ID,
		&item.OrderNo,
		&item.UserID,
		&item.UserEmail,
		&item.Title,
		&item.TaxNumber,
		&item.Amount,
		&item.RecipientEmail,
		&item.Status,
		&item.FileName,
		&item.FilePath,
		&item.FileSize,
		&item.ContentType,
		&item.TaxRate,
		&item.TaxAmount,
		&item.FailureReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&completedAt,
		&userEmail,
		&username,
		&totalRecharged,
		&completedInvoiceAmount,
	); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	item.User = &service.User{
		ID:             item.UserID,
		Email:          item.UserEmail,
		Username:       username.String,
		TotalRecharged: totalRecharged.Float64,
	}
	item.UserTotalRecharged = totalRecharged.Float64
	item.UserCompletedInvoiceAmount = completedInvoiceAmount.Float64
	if userEmail.Valid && strings.TrimSpace(userEmail.String) != "" {
		item.User.Email = userEmail.String
	}
	return &item, nil
}
