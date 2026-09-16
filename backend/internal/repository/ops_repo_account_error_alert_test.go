package repository

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestListAccountErrorAlertCandidates_UsesWindowAggregation(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 7, 2, 15, 32, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"account_id",
		"account_name",
		"status_code",
		"error_count",
		"latest_at",
		"error_message",
	}).AddRow(
		int64(12),
		"CC max-0.85",
		int64(502),
		int64(78),
		end.Add(-10*time.Second),
		"Upstream request failed",
	)

	mock.ExpectQuery(`WITH event_errors AS`).
		WithArgs(start.UTC(), end.UTC(), 5, 10, int64(0), "", false).
		WillReturnRows(rows)

	got, err := repo.ListAccountErrorAlertCandidates(t.Context(), &service.OpsAccountErrorAlertCandidateFilter{
		StartTime:     start,
		EndTime:       end,
		MinErrorCount: 5,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListAccountErrorAlertCandidates() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].AccountID != 12 || got[0].AccountName != "CC max-0.85" || got[0].StatusCode != 502 || got[0].ErrorCount != 78 {
		t.Fatalf("unexpected candidate: %+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListAccountErrorAlertTopUsers_UsesTriggeredAccounts(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 7, 2, 15, 32, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"user_email",
		"error_count",
	}).AddRow(
		"heavy@example.com",
		int64(31),
	)

	mock.ExpectQuery(`WITH event_errors AS`).
		WithArgs(start.UTC(), end.UTC(), 5, 3, int64(0), "", false).
		WillReturnRows(rows)

	got, err := repo.ListAccountErrorAlertTopUsers(t.Context(), &service.OpsAccountErrorAlertTopUserFilter{
		StartTime:     start,
		EndTime:       end,
		MinErrorCount: 5,
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("ListAccountErrorAlertTopUsers() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].UserEmail != "heavy@example.com" || got[0].ErrorCount != 31 {
		t.Fatalf("unexpected top user: %+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListAccountErrorAlertTopUsers_SkipsRowsWithoutEmail(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 7, 2, 15, 32, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"user_email",
		"error_count",
	}).AddRow(
		"",
		int64(31),
	)

	mock.ExpectQuery(`WITH event_errors AS`).
		WithArgs(start.UTC(), end.UTC(), 5, 3, int64(0), "", false).
		WillReturnRows(rows)

	got, err := repo.ListAccountErrorAlertTopUsers(t.Context(), &service.OpsAccountErrorAlertTopUserFilter{
		StartTime:     start,
		EndTime:       end,
		MinErrorCount: 5,
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("ListAccountErrorAlertTopUsers() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0: %+v", len(got), got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListAccountErrorAlertCandidates_PassesKeywordFilters(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 7, 2, 15, 32, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"account_id",
		"account_name",
		"status_code",
		"error_count",
		"latest_at",
		"error_message",
	}).AddRow(
		int64(7),
		"CPA-Pro20-817",
		int64(503),
		int64(3),
		end.Add(-10*time.Second),
		"Our servers are currently overloaded",
	)

	mock.ExpectQuery(`WITH event_errors AS`).
		WithArgs(start.UTC(), end.UTC(), 1, 10, int64(7), "overloaded", false).
		WillReturnRows(rows)

	got, err := repo.ListAccountErrorAlertCandidates(t.Context(), &service.OpsAccountErrorAlertCandidateFilter{
		StartTime:     start,
		EndTime:       end,
		MinErrorCount: 1,
		Limit:         10,
		AccountID:     7,
		Keyword:       "overloaded",
	})
	if err != nil {
		t.Fatalf("ListAccountErrorAlertCandidates() error = %v", err)
	}
	if len(got) != 1 || got[0].AccountID != 7 {
		t.Fatalf("unexpected candidate: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListAccountIDsWithErrorAlertRules(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(7)).AddRow(int64(9))
	mock.ExpectQuery(`SELECT id`).WillReturnRows(rows)

	got, err := repo.ListAccountIDsWithErrorAlertRules(t.Context())
	if err != nil {
		t.Fatalf("ListAccountIDsWithErrorAlertRules() error = %v", err)
	}
	if len(got) != 2 || got[0] != 7 || got[1] != 9 {
		t.Fatalf("got = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetAccountErrorAlertSettings(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	rows := sqlmock.NewRows([]string{"id", "extra"}).
		AddRow(int64(7), []byte(`{"error_alert":{"enabled":false}}`)).
		AddRow(int64(8), []byte(`{}`))
	mock.ExpectQuery(`SELECT id, extra`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	got, err := repo.GetAccountErrorAlertSettings(t.Context(), []int64{7, 8})
	if err != nil {
		t.Fatalf("GetAccountErrorAlertSettings() error = %v", err)
	}
	if got[7].Enabled {
		t.Fatal("account 7 should be disabled")
	}
	if !got[8].Enabled {
		t.Fatal("account 8 should default to enabled")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
