//go:build unit

package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type redeemServiceRepositoryStub struct {
	RedeemCodeRepository
	code         *RedeemCode
	getByCodeErr error
	useCalls     int
}

func (r *redeemServiceRepositoryStub) GetByCode(context.Context, string) (*RedeemCode, error) {
	if r.getByCodeErr != nil {
		return nil, r.getByCodeErr
	}
	if r.code == nil {
		return nil, ErrRedeemCodeNotFound
	}
	cloned := *r.code
	return &cloned, nil
}

func (r *redeemServiceRepositoryStub) GetByID(_ context.Context, id int64) (*RedeemCode, error) {
	if r.code == nil || r.code.ID != id {
		return nil, ErrRedeemCodeNotFound
	}
	cloned := *r.code
	return &cloned, nil
}

func (r *redeemServiceRepositoryStub) Use(_ context.Context, id, userID int64) error {
	r.useCalls++
	if r.code == nil || r.code.ID != id {
		return ErrRedeemCodeNotFound
	}
	if r.code.Status != StatusUnused {
		return ErrRedeemCodeUsed
	}
	now := time.Now().UTC()
	r.code.Status = StatusUsed
	r.code.UsedBy = &userID
	r.code.UsedAt = &now
	return nil
}

type redeemServiceCacheStub struct {
	count          int
	getErr         error
	incrementErr   error
	acquireOK      bool
	acquireErr     error
	releaseErr     error
	incrementCalls int
	acquireCalls   int
	releaseCalls   int
}

func (c *redeemServiceCacheStub) GetRedeemAttemptCount(context.Context, int64) (int, error) {
	return c.count, c.getErr
}

func (c *redeemServiceCacheStub) IncrementRedeemAttemptCount(context.Context, int64) error {
	c.incrementCalls++
	if c.incrementErr == nil {
		c.count++
	}
	return c.incrementErr
}

func (c *redeemServiceCacheStub) AcquireRedeemLock(context.Context, string, time.Duration) (bool, error) {
	c.acquireCalls++
	return c.acquireOK, c.acquireErr
}

func (c *redeemServiceCacheStub) ReleaseRedeemLock(context.Context, string) error {
	c.releaseCalls++
	return c.releaseErr
}

func TestRedeemServiceRedeemsSingleUnusedBalanceCode(t *testing.T) {
	repo := &redeemServiceRepositoryStub{code: &RedeemCode{
		ID:     1,
		Code:   "SINGLE-UNUSED",
		Type:   RedeemTypeBalance,
		Value:  25,
		Status: StatusUnused,
	}}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: 42, Balance: 10}}
	userRepo.updateBalanceFn = func(_ context.Context, id int64, amount float64) error {
		require.Equal(t, int64(42), id)
		userRepo.getByIDUser.Balance += amount
		return nil
	}
	svc := NewRedeemService(repo, userRepo, nil, nil, nil, newRedeemServiceTestClient(t), nil, nil)

	result, err := svc.Redeem(context.Background(), 42, "SINGLE-UNUSED")

	require.NoError(t, err)
	require.Equal(t, StatusUsed, result.Status)
	require.Equal(t, 35.0, userRepo.getByIDUser.Balance)
	require.Equal(t, 1, repo.useCalls)
}

func TestRedeemServiceReturnsNotFound(t *testing.T) {
	cache := &redeemServiceCacheStub{acquireOK: true}
	svc := NewRedeemService(
		&redeemServiceRepositoryStub{getByCodeErr: ErrRedeemCodeNotFound},
		nil,
		nil,
		cache,
		nil,
		nil,
		nil,
		nil,
	)

	_, err := svc.Redeem(context.Background(), 42, "MISSING")

	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, 1, cache.incrementCalls)
}

func TestRedeemServiceReturnsUsed(t *testing.T) {
	cache := &redeemServiceCacheStub{acquireOK: true}
	repo := &redeemServiceRepositoryStub{code: &RedeemCode{
		ID:     1,
		Code:   "ALREADY-USED",
		Type:   RedeemTypeBalance,
		Value:  25,
		Status: StatusUsed,
	}}
	svc := NewRedeemService(repo, nil, nil, cache, nil, nil, nil, nil)

	_, err := svc.Redeem(context.Background(), 42, "ALREADY-USED")

	require.ErrorIs(t, err, ErrRedeemCodeUsed)
	require.Equal(t, 1, cache.incrementCalls)
	require.Zero(t, repo.useCalls)
}

func TestRedeemServiceReturnsDataConflictWithoutSideEffects(t *testing.T) {
	var logOutput bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	code := &RedeemCode{
		ID:     1,
		Code:   "DUPLICATE-CODE",
		Type:   RedeemTypeBalance,
		Value:  25,
		Status: StatusUnused,
	}
	repo := &redeemServiceRepositoryStub{
		code:         code,
		getByCodeErr: ErrRedeemCodeDataConflict,
	}
	cache := &redeemServiceCacheStub{acquireOK: true}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: 42, Balance: 10, Concurrency: 3}}
	userRepo.updateBalanceFn = func(context.Context, int64, float64) error {
		t.Fatal("balance must not be modified for conflicting redeem data")
		return nil
	}
	svc := NewRedeemService(repo, userRepo, nil, cache, nil, nil, nil, nil)

	_, err := svc.Redeem(context.Background(), 42, "DUPLICATE-CODE")

	require.ErrorIs(t, err, ErrRedeemCodeDataConflict)
	require.Equal(t, http.StatusConflict, infraerrors.Code(err))
	require.Equal(t, "REDEEM_CODE_DATA_CONFLICT", infraerrors.Reason(err))
	require.Equal(t, "redeem code data conflict, please contact support", infraerrors.Message(err))
	require.Equal(t, StatusUnused, code.Status)
	require.Equal(t, 10.0, userRepo.getByIDUser.Balance)
	require.Equal(t, 3, userRepo.getByIDUser.Concurrency)
	require.Zero(t, repo.useCalls)
	require.Zero(t, cache.incrementCalls)
	require.Contains(t, logOutput.String(), "user_id=42")
	require.Contains(t, logOutput.String(), "code_hash=")
	require.NotContains(t, logOutput.String(), code.Code)
}

func TestRedeemServiceReturnsTooManyRequestsAfterTwentyFailures(t *testing.T) {
	cache := &redeemServiceCacheStub{count: 20, acquireOK: true}
	svc := NewRedeemService(&redeemServiceRepositoryStub{}, nil, nil, cache, nil, nil, nil, nil)

	_, err := svc.Redeem(context.Background(), 42, "ANY-CODE")

	require.ErrorIs(t, err, ErrRedeemRateLimited)
	require.Equal(t, http.StatusTooManyRequests, infraerrors.Code(err))
	require.Equal(t, "REDEEM_RATE_LIMITED", infraerrors.Reason(err))
	require.Zero(t, cache.acquireCalls)
}

func TestRedeemServiceFailsOpenWhenRedisIsUnavailable(t *testing.T) {
	redisErr := errors.New("redis unavailable")
	cache := &redeemServiceCacheStub{
		getErr:     redisErr,
		acquireErr: redisErr,
		releaseErr: redisErr,
	}
	repo := &redeemServiceRepositoryStub{code: &RedeemCode{
		ID:     1,
		Code:   "REDIS-FAIL-OPEN",
		Type:   RedeemTypeBalance,
		Value:  5,
		Status: StatusUnused,
	}}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: 42, Balance: 10}}
	userRepo.updateBalanceFn = func(_ context.Context, _ int64, amount float64) error {
		userRepo.getByIDUser.Balance += amount
		return nil
	}
	svc := NewRedeemService(repo, userRepo, nil, cache, nil, newRedeemServiceTestClient(t), nil, nil)

	result, err := svc.Redeem(context.Background(), 42, "REDIS-FAIL-OPEN")

	require.NoError(t, err)
	require.Equal(t, StatusUsed, result.Status)
	require.Equal(t, 15.0, userRepo.getByIDUser.Balance)
	require.Equal(t, 1, cache.acquireCalls)
	require.Equal(t, 1, cache.releaseCalls)
}

func TestRedeemServiceDoesNotTriggerAffiliateRebateForManualCodes(t *testing.T) {
	source, err := os.ReadFile("redeem_service.go")
	require.NoError(t, err)
	content := string(source)

	require.NotContains(t, content, "tryAccrueAffiliateRebateForRedeem")
	require.NotContains(t, content, "AccrueInviteRebate(ctx, userID")
}

func newRedeemServiceTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open(
		"sqlite",
		fmt.Sprintf("file:redeem_service_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
