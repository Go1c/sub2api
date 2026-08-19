package repository

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type balanceWalletRepository struct {
	db *sql.DB
}

func NewBalanceWalletRepository(db *sql.DB) service.BalanceWalletRepository {
	return &balanceWalletRepository{db: db}
}

func (r *balanceWalletRepository) CreateClient(ctx context.Context, client *service.BalanceDebitClient) error {
	if r == nil || r.db == nil || client == nil {
		return service.ErrBalanceStoreUnavailable
	}
	purposes, err := json.Marshal(client.AllowedPurposes)
	if err != nil {
		return service.ErrBalanceClientInvalid.WithCause(err)
	}
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO balance_debit_clients
			(client_id, name, secret_hash, secret_prefix, allowed_purposes, status)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6)
		RETURNING id, created_at, updated_at`,
		client.ClientID, client.Name, client.SecretHash, client.SecretPrefix, string(purposes), client.Status,
	).Scan(&client.ID, &client.CreatedAt, &client.UpdatedAt)
	if err != nil {
		if isBalanceUniqueViolation(err) {
			return service.ErrBalanceClientConflict.WithCause(err)
		}
		return mapBalanceStoreError(err)
	}
	return nil
}

func (r *balanceWalletRepository) ListClients(ctx context.Context) ([]service.BalanceDebitClient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, client_id::text, name, secret_prefix, allowed_purposes::text,
		       status, last_used_at, created_at, updated_at
		FROM balance_debit_clients
		ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.BalanceDebitClient, 0)
	for rows.Next() {
		item, err := scanBalanceClient(rows.Scan)
		if err != nil {
			return nil, mapBalanceStoreError(err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return items, nil
}

func (r *balanceWalletRepository) GetClient(ctx context.Context, clientID string) (*service.BalanceDebitClient, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, client_id::text, name, secret_prefix, allowed_purposes::text,
		       status, last_used_at, created_at, updated_at
		FROM balance_debit_clients WHERE client_id = $1::uuid`, clientID)
	item, err := scanBalanceClient(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBalanceClientNotFound
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return item, nil
}

func (r *balanceWalletRepository) GetActiveClientBySecretHash(ctx context.Context, secretHash string) (*service.BalanceDebitClient, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE balance_debit_clients
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE secret_hash = $1 AND status = 'active'
		RETURNING id, client_id::text, name, secret_prefix, allowed_purposes::text,
		          status, last_used_at, created_at, updated_at, secret_hash`, secretHash)
	item, err := scanAuthenticatedBalanceClient(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBalanceClientNotFound
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return item, nil
}

func (r *balanceWalletRepository) UpdateClient(ctx context.Context, clientID string, input service.UpdateBalanceClientInput) (*service.BalanceDebitClient, error) {
	sets := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if input.Name != nil {
		args = append(args, *input.Name)
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if input.AllowedPurposes != nil {
		raw, err := json.Marshal(*input.AllowedPurposes)
		if err != nil {
			return nil, service.ErrBalanceClientInvalid.WithCause(err)
		}
		args = append(args, string(raw))
		sets = append(sets, fmt.Sprintf("allowed_purposes = $%d::jsonb", len(args)))
	}
	if input.Status != nil {
		args = append(args, *input.Status)
		sets = append(sets, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(sets) == 0 {
		return r.GetClient(ctx, clientID)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, clientID)
	query := `UPDATE balance_debit_clients SET ` + strings.Join(sets, ", ") +
		fmt.Sprintf(` WHERE client_id = $%d::uuid
		RETURNING id, client_id::text, name, secret_prefix, allowed_purposes::text,
		          status, last_used_at, created_at, updated_at`, len(args))
	item, err := scanBalanceClient(r.db.QueryRowContext(ctx, query, args...).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBalanceClientNotFound
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return item, nil
}

func (r *balanceWalletRepository) RotateClientSecret(ctx context.Context, clientID, secretHash, secretPrefix string) (*service.BalanceDebitClient, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE balance_debit_clients
		SET secret_hash = $1, secret_prefix = $2, updated_at = NOW()
		WHERE client_id = $3::uuid
		RETURNING id, client_id::text, name, secret_prefix, allowed_purposes::text,
		          status, last_used_at, created_at, updated_at`, secretHash, secretPrefix, clientID)
	item, err := scanBalanceClient(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBalanceClientNotFound
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return item, nil
}

func (r *balanceWalletRepository) DeactivateClient(ctx context.Context, clientID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE balance_debit_clients SET status = 'inactive', updated_at = NOW()
		WHERE client_id = $1::uuid`, clientID)
	if err != nil {
		return mapBalanceStoreError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return mapBalanceStoreError(err)
	}
	if count == 0 {
		return service.ErrBalanceClientNotFound
	}
	return nil
}

func (r *balanceWalletRepository) Debit(ctx context.Context, command service.BalanceDebitCommand) (_ *service.BalanceDebitResult, retErr error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBalanceStoreUnavailable
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	defer func() {
		if retErr != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout = '3s'`); err != nil {
		return nil, mapBalanceStoreError(err)
	}

	var clientStatus, purposesRaw, currentSecretHash string
	err = tx.QueryRowContext(ctx, `
		SELECT status, allowed_purposes::text, secret_hash
		FROM balance_debit_clients WHERE id = $1 FOR UPDATE`, command.ClientID,
	).Scan(&clientStatus, &purposesRaw, &currentSecretHash)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && clientStatus != service.BalanceClientStatusActive) {
		return nil, service.ErrInvalidBalanceClient
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	if subtle.ConstantTimeCompare([]byte(command.ClientSecretHash), []byte(currentSecretHash)) != 1 {
		return nil, service.ErrInvalidBalanceClient
	}
	var purposes []string
	if err := json.Unmarshal([]byte(purposesRaw), &purposes); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	if !containsExact(purposes, command.Request.Purpose) {
		return nil, service.ErrBalancePurposeNotAllowed
	}

	var userStatus, balanceRaw string
	err = tx.QueryRowContext(ctx, `
		SELECT status, balance::text FROM users
		WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, command.UserID,
	).Scan(&userStatus, &balanceRaw)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && userStatus != service.StatusActive) {
		return nil, service.ErrBalanceUserInactive
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}

	var existing service.BalanceDebitResult
	var existingFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT txn_id::text, request_fingerprint, amount::text, balance_after::text, currency
		FROM balance_debit_transactions
		WHERE balance_client_id = $1 AND user_id = $2 AND idempotency_key_hash = $3`,
		command.ClientID, command.UserID, command.Request.IdempotencyKeyHash,
	).Scan(&existing.TxnID, &existingFingerprint, &existing.Amount, &existing.BalanceAfter, &existing.Currency)
	if err == nil {
		if existingFingerprint != command.Request.Fingerprint {
			return nil, service.ErrIdempotencyKeyConflict
		}
		existing.Replayed = true
		if err := tx.Rollback(); err != nil {
			return nil, mapBalanceStoreError(err)
		}
		return &existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, mapBalanceStoreError(err)
	}

	balance, err := decimal.NewFromString(balanceRaw)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	amount, err := decimal.NewFromString(command.Request.Amount)
	if err != nil {
		return nil, service.ErrInvalidBalanceDebitRequest
	}
	if balance.LessThan(amount) {
		return nil, service.ErrBalanceInsufficient.WithMetadata(map[string]string{
			"balance": balance.StringFixed(8), "required": amount.StringFixed(2),
		})
	}

	balanceBefore := balance.StringFixed(8)
	var balanceAfter string
	err = tx.QueryRowContext(ctx, `
		UPDATE users SET balance = balance - $1::numeric, updated_at = NOW()
		WHERE id = $2 RETURNING balance::text`, command.Request.Amount, command.UserID,
	).Scan(&balanceAfter)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	txnID := uuid.NewString()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO balance_debit_transactions
			(txn_id, user_id, balance_client_id, idempotency_key_hash, request_fingerprint,
			 amount, currency, purpose, ref, balance_before, balance_after)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::numeric, $7, $8, $9, $10::numeric, $11::numeric)`,
		txnID, command.UserID, command.ClientID, command.Request.IdempotencyKeyHash,
		command.Request.Fingerprint, command.Request.Amount, command.Request.Currency,
		command.Request.Purpose, command.Request.Ref, balanceBefore, balanceAfter,
	)
	if err != nil {
		if isBalanceUniqueViolation(err) {
			return nil, service.ErrBalanceDebitBusy
		}
		return nil, mapBalanceStoreError(err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO balance_cache_invalidation_outbox (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET
			attempts = 0, next_attempt_at = NOW(), claimed_at = NULL, claim_token = NULL,
			last_error = '', updated_at = NOW()`, command.UserID)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return &service.BalanceDebitResult{
		TxnID: txnID, Amount: command.Request.Amount, BalanceAfter: normalizeBalanceSnapshot(balanceAfter),
		Currency: command.Request.Currency,
	}, nil
}

func (r *balanceWalletRepository) ListTransactions(ctx context.Context, userID int64, filter service.BalanceTransactionFilter) (*service.BalanceTransactionPage, error) {
	where := []string{"t.user_id = $1"}
	args := []any{userID}
	add := func(column string, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("t.purpose", filter.Purpose)
	add("t.ref", filter.Ref)
	if filter.ClientID != "" {
		args = append(args, filter.ClientID)
		where = append(where, fmt.Sprintf("c.client_id = $%d::uuid", len(args)))
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM balance_debit_transactions t
		JOIN balance_debit_clients c ON c.id = t.balance_client_id`+whereSQL, args...).Scan(&total); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	offset := (filter.Page - 1) * filter.PageSize
	selectArgs := append(append([]any{}, args...), filter.PageSize, offset)
	query := `SELECT t.id, t.txn_id::text, t.user_id, t.balance_client_id,
		c.client_id::text, c.name, t.amount::text, t.balance_before::text,
		t.balance_after::text, t.currency, t.purpose, t.ref, t.created_at
		FROM balance_debit_transactions t
		JOIN balance_debit_clients c ON c.id = t.balance_client_id` + whereSQL +
		fmt.Sprintf(" ORDER BY t.created_at DESC, t.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, selectArgs...)
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.BalanceDebitTransaction, 0, filter.PageSize)
	for rows.Next() {
		item, err := scanBalanceTransaction(rows.Scan)
		if err != nil {
			return nil, mapBalanceStoreError(err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return &service.BalanceTransactionPage{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *balanceWalletRepository) GetTransaction(ctx context.Context, userID int64, txnID string) (*service.BalanceDebitTransaction, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT t.id, t.txn_id::text, t.user_id, t.balance_client_id,
		       c.client_id::text, c.name, t.amount::text, t.balance_before::text,
		       t.balance_after::text, t.currency, t.purpose, t.ref, t.created_at
		FROM balance_debit_transactions t
		JOIN balance_debit_clients c ON c.id = t.balance_client_id
		WHERE t.user_id = $1 AND t.txn_id = $2::uuid`, userID, txnID)
	item, err := scanBalanceTransaction(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBalanceTransactionNotFound
	}
	if err != nil {
		return nil, mapBalanceStoreError(err)
	}
	return item, nil
}

type scanner func(dest ...any) error

func scanBalanceClient(scan scanner) (*service.BalanceDebitClient, error) {
	item := &service.BalanceDebitClient{}
	var purposesRaw string
	var lastUsed sql.NullTime
	if err := scan(&item.ID, &item.ClientID, &item.Name, &item.SecretPrefix, &purposesRaw,
		&item.Status, &lastUsed, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(purposesRaw), &item.AllowedPurposes); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		value := lastUsed.Time
		item.LastUsedAt = &value
	}
	return item, nil
}

func scanAuthenticatedBalanceClient(scan scanner) (*service.BalanceDebitClient, error) {
	item := &service.BalanceDebitClient{}
	var purposesRaw string
	var lastUsed sql.NullTime
	if err := scan(&item.ID, &item.ClientID, &item.Name, &item.SecretPrefix, &purposesRaw,
		&item.Status, &lastUsed, &item.CreatedAt, &item.UpdatedAt, &item.SecretHash); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(purposesRaw), &item.AllowedPurposes); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		value := lastUsed.Time
		item.LastUsedAt = &value
	}
	return item, nil
}

func scanBalanceTransaction(scan scanner) (*service.BalanceDebitTransaction, error) {
	item := &service.BalanceDebitTransaction{}
	err := scan(&item.ID, &item.TxnID, &item.UserID, &item.BalanceClientID,
		&item.ClientID, &item.ClientName, &item.Amount, &item.BalanceBefore,
		&item.BalanceAfter, &item.Currency, &item.Purpose, &item.Ref, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.BalanceBefore = normalizeBalanceSnapshot(item.BalanceBefore)
	item.BalanceAfter = normalizeBalanceSnapshot(item.BalanceAfter)
	return item, nil
}

func normalizeBalanceSnapshot(raw string) string {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return raw
	}
	return value.StringFixed(8)
}

func containsExact(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func mapBalanceStoreError(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch string(pqErr.Code) {
		case "55P03", "40P01", "40001":
			return service.ErrBalanceDebitBusy.WithCause(err)
		}
	}
	return service.ErrBalanceStoreUnavailable.WithCause(err)
}

func isBalanceUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505"
}

var _ service.BalanceWalletRepository = (*balanceWalletRepository)(nil)
