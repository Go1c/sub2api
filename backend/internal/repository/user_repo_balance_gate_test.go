package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositorySumBalanceUsageGateRechargeAmountCountsPaidExternalSubscriptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUserRepositoryWithSQL(nil, db)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(po\\.pay_amount\\), 0\\)(?s:.*)order_type = \\$2(?s:.*)payment_type <> \\$3(?s:.*)status IN").
		WithArgs(
			int64(7),
			payment.OrderTypeSubscription,
			payment.TypeBalance,
			service.OrderStatusCompleted,
			service.OrderStatusPaid,
			service.OrderStatusRecharging,
			service.OrderStatusFulfillmentFailed,
		).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(79.0))

	total, err := repo.SumBalanceUsageGateRechargeAmount(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, 79.0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}
