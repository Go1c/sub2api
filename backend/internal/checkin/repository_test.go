//go:build unit

package checkin

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func settingsRows(enabled bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"enabled", "min_reward", "max_reward", "timezone", "daily_cap", "milestones", "updated_at"}).
		AddRow(enabled, "0.10000000", "0.10000000", "Asia/Shanghai", "10.00000000", []byte(`[{"day":7,"bonus":"1.0000"}]`), time.Now())
}

func recordRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "user_email", "username", "business_date", "checked_at", "timezone", "streak_days", "cycle_day", "milestone_day", "base_reward", "milestone_bonus", "actual_reward", "status", "balance_after", "client_ip", "user_agent"}).
		AddRow(8, 17, "u@example.com", "user", now, now, "Asia/Shanghai", 2, 2, nil, "0.1", "0", "0.1", StatusAwarded, "1.1", "127.0.0.1", "test")
}

func TestRepositoryCheckInAwardsAtomically(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FROM daily_checkin_settings.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FROM users.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.00000000"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO daily_checkin_daily_counters (business_date)")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days.+ORDER BY business_date DESC`).WillReturnRows(sqlmock.NewRows([]string{"business_date", "streak_days"}).AddRow(time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), 1))
	mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("1.10000000"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records.+RETURNING id, checked_at`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(8, now))
	mock.ExpectExec(`(?s)UPDATE daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), "checkin_balance", "0.1000", "used", int64(17), "daily_checkin:8").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{IP: "127.0.0.1", UserAgent: "test"})
	require.NoError(t, err)
	require.False(t, result.AlreadyCheckedIn)
	require.Equal(t, "0.1000", formatAmount(result.Record.ActualReward))
	require.Equal(t, 2, result.Record.StreakDays)
	require.Equal(t, 0, result.Record.MilestoneDay)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInSplitsMilestoneRedeemCodes(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FROM daily_checkin_settings.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FROM users.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.00000000"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO daily_checkin_daily_counters (business_date)")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days.+ORDER BY business_date DESC`).WillReturnRows(sqlmock.NewRows([]string{"business_date", "streak_days"}).AddRow(time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), 6))
	mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).
		WithArgs(int64(17), "1.1000").
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("2.10000000"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records.+RETURNING id, checked_at`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(8, now))
	mock.ExpectExec(`(?s)UPDATE daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), "checkin_balance", "0.1000", "used", int64(17), "daily_checkin:8").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), "checkin_milestone", "1.0000", "used", int64(17), "daily_checkin_milestone:8:7").
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{IP: "127.0.0.1", UserAgent: "test"})
	require.NoError(t, err)
	require.False(t, result.AlreadyCheckedIn)
	require.Equal(t, 7, result.Record.StreakDays)
	require.Equal(t, 7, result.Record.MilestoneDay)
	require.Equal(t, "0.1000", formatAmount(result.Record.BaseReward))
	require.Equal(t, "1.0000", formatAmount(result.Record.MilestoneBonus))
	require.Equal(t, "1.1000", formatAmount(result.Record.ActualReward))
	require.Equal(t, "2.1000", formatAmount(result.Record.BalanceAfter))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInSkipsRedeemCodeWhenBudgetExhausted(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.00000000"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("10.00000000"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(9, now))
	mock.ExpectCommit()

	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, StatusBudgetExhausted, result.Record.Status)
	require.Equal(t, "0.0000", formatAmount(result.Record.ActualReward))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInSkipsRedeemCodeWhenActualIsZero(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})
	zeroRows := sqlmock.NewRows([]string{"enabled", "min_reward", "max_reward", "timezone", "daily_cap", "milestones", "updated_at"}).
		AddRow(true, "0.00000000", "0.00000000", "Asia/Shanghai", "10.00000000", []byte(`[]`), now)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(zeroRows)
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.00000000"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(10, now))
	mock.ExpectCommit()

	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, StatusAwarded, result.Record.Status)
	require.Equal(t, "0.0000", formatAmount(result.Record.ActualReward))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInRollsBackWhenRedeemCodeInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("1.1"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(8, now))
	mock.ExpectExec(`(?s)UPDATE daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).WillReturnError(errors.New("redeem write failed"))
	mock.ExpectRollback()

	_, err = repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.ErrorContains(t, err, "redeem write failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInRollsBackWhenMilestoneRedeemCodeInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FROM daily_checkin_settings.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FROM users.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.00000000"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO daily_checkin_daily_counters (business_date)")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days.+ORDER BY business_date DESC`).WillReturnRows(sqlmock.NewRows([]string{"business_date", "streak_days"}).AddRow(time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC), 6))
	mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).
		WithArgs(int64(17), "1.1000").
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("2.10000000"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records.+RETURNING id, checked_at`).WillReturnRows(sqlmock.NewRows([]string{"id", "checked_at"}).AddRow(8, now))
	mock.ExpectExec(`(?s)UPDATE daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), "checkin_balance", "0.1000", "used", int64(17), "daily_checkin:8").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO redeem_codes`).
		WithArgs(sqlmock.AnyArg(), "checkin_milestone", "1.0000", "used", int64(17), "daily_checkin_milestone:8:7").
		WillReturnError(errors.New("milestone redeem write failed"))
	mock.ExpectRollback()

	_, err = repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.ErrorContains(t, err, "milestone redeem write failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInReplaysExistingRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.1"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnRows(recordRows(now))
	mock.ExpectCommit()
	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.NoError(t, err)
	require.True(t, result.AlreadyCheckedIn)
	require.Equal(t, int64(8), result.Record.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInRollsBackWhenRecordInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	repo := newSQLRepository(db, fixedRandom{})
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("1.1"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records`).WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()
	_, err = repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.ErrorContains(t, err, "write failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryAdminStatsAggregatesAwardedRewards(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newSQLRepository(db, fixedRandom{})
	from := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)WITH filtered AS.+percentile_cont\(0.5\).+percentile_cont\(0.9\).+status = 'awarded'`).
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{
			"checkin_count", "unique_users", "total_amount", "avg_amount", "p50_amount", "p90_amount", "max_amount",
		}).AddRow(int64(4), int64(3), "2.5000", "0.8333", "0.8000", "1.2000", "1.5000"))

	stats, err := repo.AdminStats(context.Background(), AdminRecordFilter{BusinessDateFrom: &from, BusinessDateTo: &to})
	require.NoError(t, err)
	require.Equal(t, int64(4), stats.CheckInCount)
	require.Equal(t, int64(3), stats.UniqueUsers)
	require.Equal(t, "2.5000", formatAmount(stats.TotalAmount))
	require.Equal(t, "0.8333", formatAmount(stats.AvgAmount))
	require.Equal(t, "0.8000", formatAmount(stats.P50Amount))
	require.Equal(t, "1.2000", formatAmount(stats.P90Amount))
	require.Equal(t, "1.5000", formatAmount(stats.MaxAmount))
	require.NoError(t, mock.ExpectationsWereMet())
}
