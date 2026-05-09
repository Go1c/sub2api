package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func newOpsRepoSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func TestOpsRepositoryCreateUserRequestMonitorRejectsExistingActiveMonitor(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	now := time.Date(2026, 5, 9, 10, 11, 12, 0, time.UTC)
	input := &service.OpsCreateUserRequestMonitorRecord{
		UserID:               42,
		TargetEmail:          "target@example.com",
		DurationSeconds:      300,
		MaxCapturesPerMinute: 10,
		SampleRatePercent:    100,
		RetentionDays:        7,
		CreatedBy:            1,
		CreatedAt:            now,
		StartsAt:             now,
		EndsAt:               now.Add(5 * time.Minute),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 FOR UPDATE`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectQuery(`SELECT id\s+FROM ops_user_request_monitors`).
		WithArgs(int64(42), now.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))
	mock.ExpectRollback()

	_, err := repo.CreateUserRequestMonitor(t.Context(), input)
	if !errors.Is(err, service.ErrOpsUserRequestMonitorAlreadyActive) {
		t.Fatalf("err = %v, want ErrOpsUserRequestMonitorAlreadyActive", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOpsRepositoryCreateUserRequestMonitorLocksUserThenInserts(t *testing.T) {
	db, mock := newOpsRepoSQLMock(t)
	repo := &opsRepository{db: db}
	now := time.Date(2026, 5, 9, 10, 11, 12, 0, time.UTC)
	endsAt := now.Add(5 * time.Minute)
	input := &service.OpsCreateUserRequestMonitorRecord{
		UserID:               42,
		TargetEmail:          "target@example.com",
		DurationSeconds:      300,
		MaxCapturesPerMinute: 10,
		SampleRatePercent:    100,
		RetentionDays:        7,
		CreatedBy:            1,
		CreatedAt:            now,
		StartsAt:             now,
		EndsAt:               endsAt,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 FOR UPDATE`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectQuery(`SELECT id\s+FROM ops_user_request_monitors`).
		WithArgs(int64(42), now.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`INSERT INTO ops_user_request_monitors`).
		WithArgs(
			int64(42),
			"target@example.com",
			300,
			10,
			100,
			7,
			int64(1),
			now.UTC(),
			now.UTC(),
			endsAt.UTC(),
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"user_id",
			"target_email",
			"status",
			"duration_seconds",
			"max_captures_per_minute",
			"sample_rate_percent",
			"retention_days",
			"created_by",
			"created_at",
			"starts_at",
			"ends_at",
			"stopped_at",
			"last_capture_at",
			"capture_count",
		}).AddRow(
			int64(11),
			int64(42),
			"target@example.com",
			"active",
			300,
			10,
			100,
			7,
			int64(1),
			now.UTC(),
			now.UTC(),
			endsAt.UTC(),
			nil,
			nil,
			int64(0),
		))
	mock.ExpectCommit()

	monitor, err := repo.CreateUserRequestMonitor(t.Context(), input)
	if err != nil {
		t.Fatalf("CreateUserRequestMonitor returned error: %v", err)
	}
	if monitor == nil || monitor.ID != 11 {
		t.Fatalf("monitor = %#v, want id=11", monitor)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
