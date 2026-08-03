//go:build unit

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	selectBalanceForUpdateSQL = `(?s)SELECT balance FROM users\s+WHERE id = \$1 AND deleted_at IS NULL\s+FOR UPDATE`
	updateBalanceDeductSQL    = `(?s)UPDATE users\s+SET balance = balance - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL\s+RETURNING balance`
	reserveBatchImageHoldSQL  = `(?s)UPDATE users\s+SET balance = balance - \$1,\s+frozen_balance = COALESCE\(frozen_balance, 0\) \+ \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL AND balance >= \$1\s+RETURNING balance, frozen_balance`
	captureBatchImageHoldSQL  = `(?s)UPDATE users\s+SET balance = balance\s+\+ CASE WHEN \$1 > \$2 THEN \$1 - \$2 ELSE 0 END\s+- CASE WHEN \$2 > \$1 THEN \$2 - \$1 ELSE 0 END,\s+frozen_balance = COALESCE\(frozen_balance, 0\) - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$3 AND deleted_at IS NULL AND COALESCE\(frozen_balance, 0\) >= \$1\s+RETURNING balance, frozen_balance`
	releaseBatchImageHoldSQL  = `(?s)UPDATE users\s+SET balance = balance \+ \$1,\s+frozen_balance = COALESCE\(frozen_balance, 0\) - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL AND COALESCE\(frozen_balance, 0\) >= \$1\s+RETURNING balance, frozen_balance`
	userExistsForBillingSQL   = `(?s)SELECT 1\s+FROM users\s+WHERE id = \$1 AND deleted_at IS NULL`
)

func TestDeductUsageBillingBalance_FloorsAtAvailableBalance(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(selectBalanceForUpdateSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10.0))
	mock.ExpectQuery(updateBalanceDeductSQL).
		WithArgs(2.5, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(7.5))
	mock.ExpectCommit()

	newBalance, deducted, err := deductUsageBillingBalance(ctx, tx, 42, 2.5)
	require.NoError(t, err)
	require.InDelta(t, 2.5, deducted, 0.000001)
	require.InDelta(t, 7.5, newBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeductUsageBillingBalance_PartialWhenInsufficient(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(selectBalanceForUpdateSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(3.0))
	mock.ExpectQuery(updateBalanceDeductSQL).
		WithArgs(3.0, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(0.0))
	mock.ExpectCommit()

	newBalance, deducted, err := deductUsageBillingBalance(ctx, tx, 42, 10)
	require.NoError(t, err)
	require.InDelta(t, 3.0, deducted, 0.000001)
	require.InDelta(t, 0.0, newBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyUsageBillingEffects_UsesActualDeductedAmount(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(selectBalanceForUpdateSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(4.0))
	mock.ExpectQuery(updateBalanceDeductSQL).
		WithArgs(4.0, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(0.0))
	mock.ExpectCommit()

	cmd := &service.UsageBillingCommand{
		UserID:      42,
		BalanceCost: 10,
	}
	result := &service.UsageBillingApplyResult{Applied: true}
	err = (&usageBillingRepository{}).applyUsageBillingEffects(ctx, tx, cmd, result)
	require.NoError(t, err)
	require.NotNil(t, result.NewBalance)
	require.InDelta(t, 0.0, *result.NewBalance, 0.000001)
	require.InDelta(t, 4.0, result.BalanceCost, 0.000001)
	require.InDelta(t, 4.0, cmd.BalanceCost, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeductUsageBillingBalance_ReturnsUserNotFoundWhenNoUser(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(selectBalanceForUpdateSQL).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, _, err = deductUsageBillingBalance(ctx, tx, 42, 10)
	require.ErrorIs(t, err, service.ErrUserNotFound)
	_ = tx.Rollback()
	require.NoError(t, mock.ExpectationsWereMet())
}
