package repository

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInvoiceRepositorySumInvoiceableSubscriptionPaymentsByUserCountsPaidExternalSubscriptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &invoiceRepository{db: db}
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(pay_amount\\), 0\\) FROM payment_orders(?s:.*)order_type = \\$2(?s:.*)payment_type <> \\$3(?s:.*)status IN").
		WithArgs(
			int64(7),
			payment.OrderTypeSubscription,
			payment.TypeBalance,
			service.OrderStatusCompleted,
			service.OrderStatusPaid,
			service.OrderStatusRecharging,
			service.OrderStatusFulfillmentFailed,
		).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(199.0))

	total, err := repo.SumInvoiceableSubscriptionPaymentsByUser(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, 199.0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInvoiceRepositorySumInvoiceableBalanceRechargePaymentsByUserCountsPaidExternalRecharges(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &invoiceRepository{db: db}
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(pay_amount\\), 0\\) FROM payment_orders(?s:.*)order_type = \\$2(?s:.*)payment_type <> \\$3(?s:.*)status IN").
		WithArgs(
			int64(7),
			payment.OrderTypeBalance,
			payment.TypeBalance,
			service.OrderStatusCompleted,
			service.OrderStatusPaid,
			service.OrderStatusRecharging,
			service.OrderStatusFulfillmentFailed,
		).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(120.0))

	total, err := repo.SumInvoiceableBalanceRechargePaymentsByUser(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, 120.0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInvoiceSelectSQLUsesPaymentOrdersForUserInvoiceableAmount(t *testing.T) {
	query := invoiceSelectSQL()

	require.NotContains(t, query, "u.total_recharged")
	require.Contains(t, query, "po.order_type = 'balance'")
	require.Contains(t, query, "po.order_type = 'subscription'")
}

func TestInvoiceRepositoryMarkCompletedAndDeductRecordsTaxLedger(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Unix(1778000000, 0)
	completedAt := now.Add(time.Minute)
	repo := &invoiceRepository{db: db}

	selectRows := sqlmock.NewRows([]string{
		"id", "order_no", "user_id", "user_email", "title", "tax_number",
		"amount", "recipient_email", "status", "file_name", "file_path",
		"file_size", "content_type", "tax_rate", "tax_amount", "failure_reason",
		"created_at", "updated_at", "completed_at",
		"email", "username", "total_recharged", "user_completed_invoice_amount",
	}).AddRow(
		int64(1), "INV00000001", int64(7), "user@example.com", "Acme Inc.", "TAX123",
		200.0, "billing@example.com", service.InvoiceStatusProcessing, "", "",
		int64(0), "", 0.0, 0.0, "",
		now, now, nil,
		"user@example.com", "alice", 1000.0, 0.0,
	)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT(?s:.*)FROM invoice_requests ir(?s:.*)FOR UPDATE OF ir").
		WithArgs(int64(1), service.InvoiceStatusProcessing).
		WillReturnRows(selectRows)
	mock.ExpectQuery("UPDATE invoice_requests(?s:.*)RETURNING completed_at, updated_at").
		WithArgs(int64(1), service.InvoiceStatusCompleted, "invoice.pdf", "data/invoices/INV00000001/invoice.pdf", int64(1234), "application/pdf", 0.01, 2.0).
		WillReturnRows(sqlmock.NewRows([]string{"completed_at", "updated_at"}).AddRow(completedAt, completedAt))
	mock.ExpectExec("UPDATE users SET balance = balance - \\$1 WHERE id = \\$2").
		WithArgs(2.0, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO redeem_codes").
		WithArgs(
			"1234567890abcdef1234567890abcdef",
			service.AdjustmentTypeAdminBalance,
			-2.0,
			service.StatusUsed,
			int64(7),
			"invoice tax deduction INV00000001 at 1.00%",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	item, err := repo.MarkCompletedAndDeduct(context.Background(), 1, service.CompleteInvoicePersistInput{
		FileName:       "invoice.pdf",
		FilePath:       "data/invoices/INV00000001/invoice.pdf",
		FileSize:       1234,
		ContentType:    "application/pdf",
		TaxRate:        0.01,
		TaxAmount:      2,
		DeductionCode:  "1234567890abcdef1234567890abcdef",
		DeductionNotes: "invoice tax deduction INV00000001 at 1.00%",
	})

	require.NoError(t, err)
	require.Equal(t, service.InvoiceStatusCompleted, item.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}
