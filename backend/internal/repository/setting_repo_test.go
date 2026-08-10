package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/lib/pq"
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

func settingBatchSavepointSQLPattern() string {
	return regexp.QuoteMeta(`SAVEPOINT sub2api_settings_batch`)
}

func settingBatchRollbackSQLPattern() string {
	return regexp.QuoteMeta(`ROLLBACK TO SAVEPOINT sub2api_settings_batch`)
}

func settingBatchReleaseSQLPattern() string {
	return regexp.QuoteMeta(`RELEASE SAVEPOINT sub2api_settings_batch`)
}

func TestSettingRepositorySetMultipleUpdatesExistingKeyWithoutInsert(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("updated", sqlmock.AnyArg(), "existing_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
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
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("created", sqlmock.AnyArg(), "new_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("new_key", "created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
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
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("updated", sqlmock.AnyArg(), "existing_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("created", sqlmock.AnyArg(), "new_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("new_key", "created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
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
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("written", sqlmock.AnyArg(), "first_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("fails", sqlmock.AnyArg(), "second_key").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("second_key", "fails", sqlmock.AnyArg()).
		WillReturnError(writeErr)
	mock.ExpectExec(settingBatchRollbackSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"second_key": "fails",
		"first_key":  "written",
	})
	require.ErrorIs(t, err, writeErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingRepositorySetMultipleFallsBackToUpsertOnUniqueViolation 复现生产观察到的怪异错误：
// UPDATE 反报 pq 23505（settings_key_key）。理论上不可能（UPDATE 不动 key 列），
// 但 b161265e 部署后仍然发生。fallback 到 ON CONFLICT upsert，请求必须成功。
func TestSettingRepositorySetMultipleFallsBackToUpsertOnUniqueViolation(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	pgErr := &pq.Error{Code: pgUniqueViolation, Constraint: "settings_key_key"}

	mock.ExpectBegin()
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("recovered", sqlmock.AnyArg(), "custom_menu_items").
		WillReturnError(pgErr)
	mock.ExpectExec(settingBatchRollbackSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingInsertUpsertSQLPattern()).
		WithArgs("custom_menu_items", "recovered", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.SetMultiple(context.Background(), map[string]string{
		"custom_menu_items": "recovered",
	})
	require.NoError(t, err, "unique violation on UPDATE must be recovered via upsert")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingRepositorySetMultipleConsecutiveSavesSucceed 验证管理员连续两次保存设置不会失败。
// 这一覆盖直接映射到生产场景：PUT /api/v1/admin/settings 反复调用必须幂等。
func TestSettingRepositorySetMultipleConsecutiveSavesSucceed(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)

	// 第一次：UPDATE 命中（管理员已经保存过一次）
	mock.ExpectBegin()
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("first", sqlmock.AnyArg(), "consecutive_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	// 第二次：UPDATE 命中
	mock.ExpectBegin()
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("second", sqlmock.AnyArg(), "consecutive_key").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	require.NoError(t, repo.SetMultiple(context.Background(), map[string]string{"consecutive_key": "first"}))
	require.NoError(t, repo.SetMultiple(context.Background(), map[string]string{"consecutive_key": "second"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingRepositorySetMultipleNonUniqueUpdateErrorPropagates 验证 UPDATE 的其它错误不被吞掉。
func TestSettingRepositorySetMultipleNonUniqueUpdateErrorPropagates(t *testing.T) {
	repo, mock := newSettingRepoSQLMock(t)
	deadlock := &pq.Error{Code: "40P01", Message: "deadlock detected"}

	mock.ExpectBegin()
	mock.ExpectExec(settingBatchSavepointSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingUpdateSQLPattern()).
		WithArgs("ignored", sqlmock.AnyArg(), "deadlock_key").
		WillReturnError(deadlock)
	mock.ExpectExec(settingBatchRollbackSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(settingBatchReleaseSQLPattern()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err := repo.SetMultiple(context.Background(), map[string]string{"deadlock_key": "ignored"})
	require.Error(t, err)
	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "40P01", string(pgErr.Code))
	require.NoError(t, mock.ExpectationsWereMet())
}
