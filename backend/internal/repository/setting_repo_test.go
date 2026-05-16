package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func newSettingRepoSQLMock(t *testing.T) (*settingRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	repo, ok := NewSettingRepository(client).(*settingRepository)
	require.True(t, ok)
	return repo, mock
}

func settingUpsertSQLPattern() string {
	return regexp.QuoteMeta(`INSERT INTO "settings" ("key", "value", "updated_at") VALUES ($1, $2, $3) ON CONFLICT ("key") DO UPDATE SET "value" = "excluded"."value", "updated_at" = "excluded"."updated_at" RETURNING "id"`)
}

func TestSettingRepositorySetMultipleUsesTransactionalEntUpserts(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(settingUpsertSQLPattern()).
		WithArgs("existing_key", "updated", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(settingUpsertSQLPattern()).
		WithArgs("new_key", "created", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	mock.ExpectCommit()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"new_key":      "created",
		"existing_key": "updated",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettingRepositorySetMultipleRollsBackWhenAnyUpsertFails(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)
	writeErr := errors.New("write failed")

	mock.ExpectBegin()
	mock.ExpectQuery(settingUpsertSQLPattern()).
		WithArgs("first_key", "written", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(settingUpsertSQLPattern()).
		WithArgs("second_key", "fails", sqlmock.AnyArg()).
		WillReturnError(writeErr)
	mock.ExpectRollback()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"second_key": "fails",
		"first_key":  "written",
	})
	require.ErrorIs(t, err, writeErr)
	require.NoError(t, mock.ExpectationsWereMet())
}
