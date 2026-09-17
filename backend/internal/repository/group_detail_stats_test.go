package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGetGroupDetailStatsScansDistinctCostsAndCurrentBindings(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	from := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)

	mock.ExpectQuery("FROM groups g CROSS JOIN LATERAL").
		WithArgs(int64(900), &from, &to).
		WillReturnRows(sqlmock.NewRows([]string{
			"name", "total_api_keys", "active_api_keys", "total_accounts",
			"requests", "tokens", "cost", "actual", "account_cost",
			"balance", "subscription", "zero_charge", "duration",
		}).AddRow("Stats group", int64(2), int64(1), int64(4), int64(1), int64(10), 20.0, 4.0, 6.0, 0.0, 4.0, int64(0), 3000.0))

	stats, err := repo.GetGroupDetailStats(context.Background(), 900, &from, &to)
	require.NoError(t, err)
	require.Equal(t, int64(900), stats.GroupID)
	require.Equal(t, "Stats group", stats.GroupName)
	require.EqualValues(t, 2, stats.TotalAPIKeys)
	require.EqualValues(t, 1, stats.ActiveAPIKeys)
	require.EqualValues(t, 1, stats.TotalRequests)
	require.Equal(t, 20.0, stats.TotalCost)
	require.Equal(t, 4.0, stats.TotalActualCost)
	require.Equal(t, 6.0, stats.TotalAccountCost)
	require.Equal(t, from, *stats.From)
	require.Equal(t, to, *stats.To)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupDetailStatsMissingGroup(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery("FROM groups g CROSS JOIN LATERAL").
		WithArgs(int64(999), nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"name", "total_api_keys", "active_api_keys", "total_accounts",
			"requests", "tokens", "cost", "actual", "account_cost",
			"balance", "subscription", "zero_charge", "duration",
		}))

	_, err := repo.GetGroupDetailStats(context.Background(), 999, nil, nil)
	require.ErrorIs(t, err, service.ErrGroupNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
