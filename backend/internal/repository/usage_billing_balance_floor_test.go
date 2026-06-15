package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// deductUsageBillingBalance 必须以 0 为下限：单次扣减最多把余额扣到 0，
// 绝不扣成负数；超出可用余额的部分由平台吸收（实际扣减 < 请求金额）。
func TestDeductUsageBillingBalanceFloorsAtZero(t *testing.T) {
	t.Run("sufficient balance deducts full amount", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT balance FROM users").
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10.0))
		mock.ExpectQuery("UPDATE users").
			WithArgs(5.0, int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(5.0))

		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)

		newBalance, deducted, err := deductUsageBillingBalance(context.Background(), tx, 7, 5.0)
		require.NoError(t, err)
		require.Equal(t, 5.0, deducted)
		require.Equal(t, 5.0, newBalance)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("insufficient balance deducts only what is available", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT balance FROM users").
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(3.0))
		// 请求 13.57，但只能扣到余额为 0 → 实际扣减 3.0。
		mock.ExpectQuery("UPDATE users").
			WithArgs(3.0, int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(0.0))

		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)

		newBalance, deducted, err := deductUsageBillingBalance(context.Background(), tx, 7, 13.57)
		require.NoError(t, err)
		require.Equal(t, 3.0, deducted)
		require.Equal(t, 0.0, newBalance)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("non-positive balance deducts nothing and never goes more negative", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT balance FROM users").
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(-2.0))
		// 不应发出任何 UPDATE。

		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)

		newBalance, deducted, err := deductUsageBillingBalance(context.Background(), tx, 7, 5.0)
		require.NoError(t, err)
		require.Equal(t, 0.0, deducted)
		require.Equal(t, -2.0, newBalance)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
