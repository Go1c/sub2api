package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newSubscriptionExpirySQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func expectExpiredSubscriptionScan(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(`(?s)SELECT id, user_id, group_id, quota_limit_usd, quota_used_usd, expires_at, expired_credit_logged_at.*FOR UPDATE SKIP LOCKED`)
}

func TestSubscriptionCreditExpiryLogsRemainingCreditAndOutbox(t *testing.T) {
	ctx := context.Background()
	db, mock := newSubscriptionExpirySQLMock(t)
	repo := &userSubscriptionRepository{db: db}
	expiresAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	expectExpiredSubscriptionScan(mock).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "group_id", "quota_limit_usd", "quota_used_usd", "expires_at", "expired_credit_logged_at",
		}).AddRow(int64(42), int64(7), nil, 10.0, 3.25, expiresAt, nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_subscriptions")).
		WithArgs(service.SubscriptionStatusExpired, int64(42), service.SubscriptionStatusActive).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO subscription_credit_ledger")).
		WithArgs(
			int64(7),
			int64(42),
			nil,
			nil,
			nil,
			nil,
			service.SubscriptionCreditLedgerExpire,
			-6.75,
			0.0,
			0.0,
			"subscription expired",
			"expire:42:2026-05-28T01:02:03Z",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scheduler_outbox")).
		WithArgs(service.SchedulerOutboxEventSubscriptionNotify, nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_subscriptions")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updated, err := repo.ExpireCreditSubscriptions(ctx)

	require.NoError(t, err)
	require.Equal(t, int64(1), updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionCreditExpirySkipsLedgerWhenNoRemainingCredit(t *testing.T) {
	ctx := context.Background()
	db, mock := newSubscriptionExpirySQLMock(t)
	repo := &userSubscriptionRepository{db: db}
	expiresAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)

	mock.ExpectBegin()
	expectExpiredSubscriptionScan(mock).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "group_id", "quota_limit_usd", "quota_used_usd", "expires_at", "expired_credit_logged_at",
		}).AddRow(int64(43), int64(8), nil, 10.0, 10.0, expiresAt, nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_subscriptions")).
		WithArgs(service.SubscriptionStatusExpired, int64(43), service.SubscriptionStatusActive).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updated, err := repo.ExpireCreditSubscriptions(ctx)

	require.NoError(t, err)
	require.Equal(t, int64(1), updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionCreditExpirySkipsLedgerWhenAlreadyLogged(t *testing.T) {
	ctx := context.Background()
	db, mock := newSubscriptionExpirySQLMock(t)
	repo := &userSubscriptionRepository{db: db}
	expiresAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)
	loggedAt := time.Date(2026, 5, 28, 2, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	expectExpiredSubscriptionScan(mock).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "group_id", "quota_limit_usd", "quota_used_usd", "expires_at", "expired_credit_logged_at",
		}).AddRow(int64(44), int64(9), nil, 10.0, 1.0, expiresAt, loggedAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_subscriptions")).
		WithArgs(service.SubscriptionStatusExpired, int64(44), service.SubscriptionStatusActive).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updated, err := repo.ExpireCreditSubscriptions(ctx)

	require.NoError(t, err)
	require.Equal(t, int64(1), updated)
	require.NoError(t, mock.ExpectationsWereMet())
}
