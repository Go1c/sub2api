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

func settingUpdateSQLPattern() string {
	return regexp.QuoteMeta(`UPDATE "settings" SET "value" = $1, "updated_at" = $2 WHERE "key" = $3`)
}

func settingInsertUpsertSQLPattern() string {
	return regexp.QuoteMeta(`INSERT INTO "settings" ("key", "value", "updated_at") VALUES ($1, $2, $3) ON CONFLICT ("key") DO UPDATE SET "value" = EXCLUDED."value", "updated_at" = EXCLUDED."updated_at"`)
}

func TestSettingRepositorySetMultipleUpdatesExistingKeyWithoutInsert(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("updated", sqlmock.AnyArg(), "existing_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"existing_key": "updated",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettingRepositorySetMultipleInsertsNewKeyAfterUpdateMiss(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("created", sqlmock.AnyArg(), "new_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("new_key", "created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"new_key": "created",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettingRepositorySetMultipleUpdatesExistingAndInsertsNewKey(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("updated", sqlmock.AnyArg(), "existing_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("created", sqlmock.AnyArg(), "new_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("new_key", "created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"new_key":      "created",
		"existing_key": "updated",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettingRepositorySetMultipleRollsBackWhenAnyWriteFails(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)
	writeErr := errors.New("write failed")

	mock.ExpectBegin()
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("written", sqlmock.AnyArg(), "first_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("fails", sqlmock.AnyArg(), "second_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
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
