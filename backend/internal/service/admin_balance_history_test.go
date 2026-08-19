package service

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestMergeBalanceHistoryCodesIncludesAffiliateTransfersByDefault(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	older := now.Add(-2 * time.Hour)
	newer := now.Add(time.Hour)

	usedBy := int64(10)
	redeemCodes := []RedeemCode{
		{
			ID:        1,
			Type:      RedeemTypeBalance,
			Value:     8,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &now,
			CreatedAt: now,
		},
		{
			ID:        2,
			Type:      RedeemTypeConcurrency,
			Value:     1,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &older,
			CreatedAt: older,
		},
	}
	affiliateCodes := []RedeemCode{
		{
			ID:        -20,
			Type:      RedeemTypeAffiliateBalance,
			Value:     3.5,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &newer,
			CreatedAt: newer,
		},
	}

	got := mergeBalanceHistoryCodes(redeemCodes, affiliateCodes, nil, pagination.PaginationParams{
		Page:     1,
		PageSize: 2,
	})

	require.Len(t, got, 2)
	require.Equal(t, RedeemTypeAffiliateBalance, got[0].Type)
	require.Equal(t, RedeemTypeBalance, got[1].Type)
}

func TestMergeBalanceHistoryCodesPaginatesAfterCombiningSources(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	usedBy := int64(10)
	at := func(hours int) *time.Time {
		v := base.Add(time.Duration(hours) * time.Hour)
		return &v
	}

	got := mergeBalanceHistoryCodes(
		[]RedeemCode{
			{ID: 1, Type: RedeemTypeBalance, UsedBy: &usedBy, UsedAt: at(4), CreatedAt: *at(4)},
			{ID: 2, Type: RedeemTypeConcurrency, UsedBy: &usedBy, UsedAt: at(2), CreatedAt: *at(2)},
		},
		[]RedeemCode{
			{ID: -3, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(3), CreatedAt: *at(3)},
			{ID: -4, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(1), CreatedAt: *at(1)},
		},
		nil,
		pagination.PaginationParams{Page: 2, PageSize: 2},
	)

	require.Len(t, got, 2)
	require.Equal(t, RedeemTypeConcurrency, got[0].Type)
	require.Equal(t, int64(-4), got[1].ID)
}

func TestMergeBalanceHistoryCodesIncludesExternalSubscriptionPaymentsByDefault(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	usedBy := int64(10)
	at := func(minutes int) *time.Time {
		v := base.Add(time.Duration(minutes) * time.Minute)
		return &v
	}

	got := mergeBalanceHistoryCodes(
		[]RedeemCode{
			{ID: 1, Type: RedeemTypeBalance, UsedBy: &usedBy, UsedAt: at(10), CreatedAt: *at(10)},
		},
		[]RedeemCode{
			{ID: -2, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(20), CreatedAt: *at(20)},
		},
		[]RedeemCode{
			{ID: -900000000003, Type: RedeemTypeSubscriptionPayment, UsedBy: &usedBy, UsedAt: at(30), CreatedAt: *at(30)},
		},
		pagination.PaginationParams{Page: 1, PageSize: 3},
	)

	require.Len(t, got, 3)
	require.Equal(t, RedeemTypeSubscriptionPayment, got[0].Type)
	require.Equal(t, RedeemTypeAffiliateBalance, got[1].Type)
	require.Equal(t, RedeemTypeBalance, got[2].Type)
}

func TestAffiliateBalanceHistoryItemMarksSignupBonusAction(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 12, 9, 30, 0, 0, time.UTC)

	got := affiliateBalanceHistoryItem(88, "signup_bonus", 2.5, 42, createdAt)

	require.Equal(t, int64(-88), got.ID)
	require.Equal(t, "AFF-88", got.Code)
	require.Equal(t, RedeemTypeAffiliateBalance, got.Type)
	require.Equal(t, 2.5, got.Value)
	require.Equal(t, StatusUsed, got.Status)
	require.NotNil(t, got.UsedBy)
	require.Equal(t, int64(42), *got.UsedBy)
	require.NotNil(t, got.UsedAt)
	require.Equal(t, createdAt, *got.UsedAt)
	require.Equal(t, createdAt, got.CreatedAt)
	require.Equal(t, "signup_bonus", got.Notes)
}

func TestPromoBalanceHistoryItemUsesPromoCodeMetadata(t *testing.T) {
	t.Parallel()

	usedAt := time.Date(2026, 7, 3, 9, 30, 0, 0, time.UTC)

	got := promoBalanceHistoryItem(PromoCodeUsage{
		ID:          123,
		UserID:      42,
		BonusAmount: 1.88,
		UsedAt:      usedAt,
		PromoCode: &PromoCode{
			Code:  " LUMIOAPI ",
			Notes: "launch_bonus",
		},
	})

	require.Equal(t, int64(-800000000123), got.ID)
	require.Equal(t, "LUMIOAPI", got.Code)
	require.Equal(t, RedeemTypePromoBalance, got.Type)
	require.Equal(t, 1.88, got.Value)
	require.Equal(t, StatusUsed, got.Status)
	require.NotNil(t, got.UsedBy)
	require.Equal(t, int64(42), *got.UsedBy)
	require.NotNil(t, got.UsedAt)
	require.Equal(t, usedAt, *got.UsedAt)
	require.Equal(t, usedAt, got.CreatedAt)
	require.Equal(t, "launch_bonus", got.Notes)
}

func TestListPromoBalanceHistoryIncludesPromoCodeUsages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	usedAt := time.Date(2026, 7, 3, 10, 15, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"code",
		"bonus_amount",
		"notes",
		"used_at",
	}).AddRow(
		int64(77),
		"LUMIOAPI",
		1.88,
		"launch_bonus",
		usedAt,
	)

	mock.ExpectQuery("FROM promo_code_usages pcu(?s:.*)LEFT JOIN promo_codes pc(?s:.*)WHERE pcu.user_id = \\$1").
		WithArgs(int64(7), 0, 10).
		WillReturnRows(rows)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)(?s:.*)FROM promo_code_usages(?s:.*)WHERE user_id = \\$1").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))

	codes, total, err := listPromoBalanceHistory(context.Background(), db, 7, pagination.PaginationParams{Page: 1, PageSize: 10})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, codes, 1)
	require.Equal(t, int64(-800000000077), codes[0].ID)
	require.Equal(t, "LUMIOAPI", codes[0].Code)
	require.Equal(t, RedeemTypePromoBalance, codes[0].Type)
	require.Equal(t, 1.88, codes[0].Value)
	require.Equal(t, StatusUsed, codes[0].Status)
	require.NotNil(t, codes[0].UsedBy)
	require.Equal(t, int64(7), *codes[0].UsedBy)
	require.NotNil(t, codes[0].UsedAt)
	require.Equal(t, usedAt, *codes[0].UsedAt)
	require.Equal(t, "launch_bonus", codes[0].Notes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSumPromoBalanceHistoryAmountIncludesPromoUsages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(bonus_amount\\), 0\\)::double precision(?s:.*)FROM promo_code_usages").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(3.76))

	total, err := sumPromoBalanceHistoryAmount(context.Background(), db, 7)

	require.NoError(t, err)
	require.Equal(t, 3.76, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSubscriptionPaymentHistoryIncludesPaidExternalSubscriptionOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	paidAt := time.Date(2026, 6, 2, 10, 30, 0, 0, time.UTC)
	createdAt := paidAt.Add(-5 * time.Minute)
	rows := sqlmock.NewRows([]string{
		"id",
		"code",
		"pay_amount",
		"status",
		"payment_type",
		"subscription_validity_days",
		"plan_name",
		"paid_at",
		"completed_at",
		"created_at",
	}).AddRow(
		int64(55),
		"PAY-55-123",
		199.0,
		OrderStatusCompleted,
		payment.TypeAlipay,
		30,
		"Pro Plan",
		paidAt,
		nil,
		createdAt,
	)

	statuses := balanceHistorySubscriptionPaymentStatuses()
	listArgs := []driver.Value{int64(7), payment.OrderTypeSubscription, payment.TypeBalance}
	for _, status := range statuses {
		listArgs = append(listArgs, status)
	}
	listArgs = append(listArgs, 0, 10)
	mock.ExpectQuery("FROM payment_orders po(?s:.*)order_type = \\$2(?s:.*)payment_type <> \\$3(?s:.*)status IN").
		WithArgs(listArgs...).
		WillReturnRows(rows)

	countArgs := []driver.Value{int64(7), payment.OrderTypeSubscription, payment.TypeBalance}
	for _, status := range statuses {
		countArgs = append(countArgs, status)
	}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)(?s:.*)FROM payment_orders po(?s:.*)order_type = \\$2").
		WithArgs(countArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))

	codes, total, err := listSubscriptionPaymentHistory(context.Background(), db, 7, pagination.PaginationParams{Page: 1, PageSize: 10})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, codes, 1)
	require.Equal(t, int64(-900000000055), codes[0].ID)
	require.Equal(t, "PAY-55-123", codes[0].Code)
	require.Equal(t, RedeemTypeSubscriptionPayment, codes[0].Type)
	require.Equal(t, 199.0, codes[0].Value)
	require.Equal(t, StatusUsed, codes[0].Status)
	require.NotNil(t, codes[0].UsedBy)
	require.Equal(t, int64(7), *codes[0].UsedBy)
	require.NotNil(t, codes[0].UsedAt)
	require.Equal(t, paidAt, *codes[0].UsedAt)
	require.Contains(t, codes[0].Notes, "Pro Plan")
	require.Contains(t, codes[0].Notes, string(payment.TypeAlipay))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMergeBalanceHistoryCodesIncludesWalletDebitsByDefault(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	usedBy := int64(10)
	at := func(minutes int) *time.Time {
		v := base.Add(time.Duration(minutes) * time.Minute)
		return &v
	}

	got := mergeBalanceHistoryCodes(
		[]RedeemCode{
			{ID: 1, Type: RedeemTypeBalance, UsedBy: &usedBy, UsedAt: at(10), CreatedAt: *at(10)},
		},
		[]RedeemCode{
			{ID: -2, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(20), CreatedAt: *at(20)},
		},
		[]RedeemCode{
			{ID: -900000000003, Type: RedeemTypeSubscriptionPayment, UsedBy: &usedBy, UsedAt: at(30), CreatedAt: *at(30)},
			{ID: -700000000004, Type: RedeemTypeWalletDebit, UsedBy: &usedBy, UsedAt: at(40), CreatedAt: *at(40), Value: -19.90},
		},
		pagination.PaginationParams{Page: 1, PageSize: 4},
	)

	require.Len(t, got, 4)
	require.Equal(t, RedeemTypeWalletDebit, got[0].Type)
	require.Equal(t, -19.90, got[0].Value)
	require.Equal(t, RedeemTypeSubscriptionPayment, got[1].Type)
	require.Equal(t, RedeemTypeAffiliateBalance, got[2].Type)
	require.Equal(t, RedeemTypeBalance, got[3].Type)
}

func TestWalletDebitHistoryItemMapsExternalDebitFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 8, 19, 8, 12, 0, 0, time.UTC)
	got := walletDebitHistoryItem(walletDebitHistoryRow{
		ID:           4,
		TxnID:        "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
		ClientID:     "9668f69e-32c4-48e9-9992-280951dcb85c",
		ClientName:   "CCHaven Control",
		Amount:       "19.90",
		BalanceAfter: "583.46000000",
		Currency:     "CNY",
		Purpose:      "cchaven_monthly",
		Ref:          "CC20260819-100001",
		CreatedAt:    createdAt,
	}, 7)

	require.Equal(t, int64(-700000000004), got.ID)
	require.Equal(t, "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a", got.Code)
	require.Equal(t, RedeemTypeWalletDebit, got.Type)
	require.InDelta(t, -19.90, got.Value, 0.0001)
	require.Equal(t, StatusUsed, got.Status)
	require.NotNil(t, got.UsedBy)
	require.Equal(t, int64(7), *got.UsedBy)
	require.NotNil(t, got.UsedAt)
	require.Equal(t, createdAt, *got.UsedAt)
	require.Equal(t, createdAt, got.CreatedAt)
	require.Contains(t, got.Notes, "CCHaven Control")
	require.Contains(t, got.Notes, "purpose=cchaven_monthly")
	require.Contains(t, got.Notes, "ref=CC20260819-100001")
	require.Contains(t, got.Notes, "txn_id=8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a")
	require.Contains(t, got.Notes, "583.46000000")
	require.NotContains(t, got.Notes, "bcs_")
	require.NotContains(t, strings.ToLower(got.Notes), "secret")
}

func TestListWalletDebitHistoryIncludesCchavenMonthlyDebit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 8, 19, 8, 12, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"txn_id",
		"client_id",
		"name",
		"amount",
		"balance_after",
		"currency",
		"purpose",
		"ref",
		"created_at",
	}).AddRow(
		int64(4),
		"8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
		"9668f69e-32c4-48e9-9992-280951dcb85c",
		"CCHaven Control",
		"19.90",
		"583.46000000",
		"CNY",
		"cchaven_monthly",
		"CC20260819-100001",
		createdAt,
	)

	mock.ExpectQuery("FROM balance_debit_transactions t(?s:.*)JOIN balance_debit_clients c(?s:.*)WHERE t.user_id = \\$1").
		WithArgs(int64(7), 0, 10).
		WillReturnRows(rows)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)(?s:.*)FROM balance_debit_transactions(?s:.*)WHERE user_id = \\$1").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))

	codes, total, err := listWalletDebitHistory(context.Background(), db, 7, pagination.PaginationParams{Page: 1, PageSize: 10})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, codes, 1)
	require.Equal(t, RedeemTypeWalletDebit, codes[0].Type)
	require.Equal(t, "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a", codes[0].Code)
	require.InDelta(t, -19.90, codes[0].Value, 0.0001)
	require.Contains(t, codes[0].Notes, "purpose=cchaven_monthly")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWalletDebitTransactionsAlignsWithUserTransactionFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 8, 19, 8, 12, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"txn_id",
		"client_id",
		"name",
		"amount",
		"balance_after",
		"currency",
		"purpose",
		"ref",
		"created_at",
	}).AddRow(
		int64(4),
		"8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
		"9668f69e-32c4-48e9-9992-280951dcb85c",
		"CCHaven Control",
		"19.90",
		"583.46000000",
		"CNY",
		"cchaven_monthly",
		"CC20260819-100001",
		createdAt,
	)

	mock.ExpectQuery("FROM balance_debit_transactions t(?s:.*)JOIN balance_debit_clients c(?s:.*)WHERE t.user_id = \\$1").
		WithArgs(int64(7), 0, 20).
		WillReturnRows(rows)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)(?s:.*)FROM balance_debit_transactions(?s:.*)WHERE user_id = \\$1").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(1)))

	page, err := listWalletDebitTransactions(context.Background(), db, 7, pagination.PaginationParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, page.Items, 1)
	require.Equal(t, "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a", page.Items[0].TxnID)
	require.Equal(t, "9668f69e-32c4-48e9-9992-280951dcb85c", page.Items[0].ClientID)
	require.Equal(t, "CCHaven Control", page.Items[0].ClientName)
	require.Equal(t, "19.90", page.Items[0].Amount)
	require.Equal(t, "583.46000000", page.Items[0].BalanceAfter)
	require.Equal(t, "CNY", page.Items[0].Currency)
	require.Equal(t, "cchaven_monthly", page.Items[0].Purpose)
	require.Equal(t, "CC20260819-100001", page.Items[0].Ref)
	require.Equal(t, createdAt, page.Items[0].CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
