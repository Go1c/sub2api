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
		WithArgs(start.UTC(), end.UTC(), 5, 10).
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
		WithArgs(start.UTC(), end.UTC(), 5, 3).
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
		WithArgs(start.UTC(), end.UTC(), 5, 3).
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
