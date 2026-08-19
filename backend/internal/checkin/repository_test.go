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
	require.NoError(t, err); t.Cleanup(func() { _ = db.Close() })
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
	mock.ExpectCommit()

	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{IP: "127.0.0.1", UserAgent: "test"})
	require.NoError(t, err)
	require.False(t, result.AlreadyCheckedIn)
	require.Equal(t, "0.1000", formatAmount(result.Record.ActualReward))
	require.Equal(t, 2, result.Record.StreakDays)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInReplaysExistingRecord(t *testing.T) {
	db, mock, err := sqlmock.New(); require.NoError(t, err); t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC); repo := newSQLRepository(db, fixedRandom{})
	mock.ExpectBegin(); mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1.1"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnRows(recordRows(now)); mock.ExpectCommit()
	result, err := repo.CheckIn(context.Background(), 17, now, ClientInfo{})
	require.NoError(t, err); require.True(t, result.AlreadyCheckedIn); require.Equal(t, int64(8), result.Record.ID); require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryCheckInRollsBackWhenRecordInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New(); require.NoError(t, err); t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC); repo := newSQLRepository(db, fixedRandom{})
	mock.ExpectBegin(); mock.ExpectQuery(`(?s)SELECT enabled.+FOR UPDATE`).WillReturnRows(settingsRows(true))
	mock.ExpectQuery(`(?s)SELECT email, username, status, balance::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"email", "username", "status", "balance"}).AddRow("u@example.com", "user", "active", "1"))
	mock.ExpectQuery(`(?s)SELECT .+FROM daily_checkin_records.+business_date = \$2`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO daily_checkin_daily_counters`).WillReturnResult(sqlmock.NewResult(0, 1)); mock.ExpectQuery(`(?s)SELECT awarded_total::text.+FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"awarded_total"}).AddRow("0"))
	mock.ExpectQuery(`(?s)SELECT business_date, streak_days`).WillReturnError(sql.ErrNoRows); mock.ExpectQuery(`(?s)UPDATE users.+RETURNING balance::text`).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("1.1"))
	mock.ExpectQuery(`(?s)INSERT INTO daily_checkin_records`).WillReturnError(errors.New("write failed")); mock.ExpectRollback()
	_, err = repo.CheckIn(context.Background(), 17, now, ClientInfo{}); require.ErrorContains(t, err, "write failed"); require.NoError(t, mock.ExpectationsWereMet())
}
