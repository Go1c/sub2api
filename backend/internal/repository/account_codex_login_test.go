package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompleteCodexLoginRecoveryConditionalOutbox(t *testing.T) {
	for _, affected := range []int64{0, 1} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		repo := &accountRepository{sql: db}
		mock.ExpectExec("WITH restored AS").WithArgs(int64(7), "owned-recovery", "new-token", service.SchedulerOutboxEventAccountChanged).WillReturnResult(sqlmock.NewResult(0, affected))
		restored, err := repo.CompleteCodexLoginRecovery(context.Background(), 7, "new-token", "owned-recovery")
		require.NoError(t, err)
		require.Equal(t, affected == 1, restored)
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	}
}
