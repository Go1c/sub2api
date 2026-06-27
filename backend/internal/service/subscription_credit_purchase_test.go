//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type subscriptionPurchaseRepoStub struct {
	userSubRepoNoop

	lockUserID int64
	hasUsable  bool
	inserted   *UserSubscription
}

func (r *subscriptionPurchaseRepoStub) LockUserForSubscriptionWrite(ctx context.Context, tx *sql.Tx, userID int64) error {
	r.lockUserID = userID
	return nil
}

func (r *subscriptionPurchaseRepoStub) HasUsableCreditSubscription(ctx context.Context, userID int64) (bool, error) {
	return r.hasUsable, nil
}

func (r *subscriptionPurchaseRepoStub) InsertCreditSubscription(ctx context.Context, tx *sql.Tx, sub *UserSubscription) (*UserSubscription, error) {
	cp := *sub
	cp.ID = 9001
	r.inserted = &cp
	return &cp, nil
}

type subscriptionPurchaseLedgerRepoStub struct {
	entry *SubscriptionCreditLedgerEntry
}

func (r *subscriptionPurchaseLedgerRepoStub) Create(ctx context.Context, exec SQLExecer, entry *SubscriptionCreditLedgerEntry) error {
	cp := *entry
	r.entry = &cp
	return nil
}

func (r *subscriptionPurchaseLedgerRepoStub) CreateLimitReachedEvent(ctx context.Context, exec SQLExecer, entry *SubscriptionCreditLedgerEntry) (bool, error) {
	panic("unexpected CreateLimitReachedEvent call")
}

func (r *subscriptionPurchaseLedgerRepoStub) ListByUserID(ctx context.Context, userID int64, ledgerType string, params pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (r *subscriptionPurchaseLedgerRepoStub) ListBySubscriptionID(ctx context.Context, subscriptionID int64, ledgerType string, params pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	panic("unexpected ListBySubscriptionID call")
}

func TestSubscriptionCreditPurchaseFulfillOrderCreatesSubscriptionAndLedger(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectCommit()

	daily := 2.5
	weekly := 8.5
	quota := 25.0
	scopeType := SubscriptionScopePlatforms
	validityDays := 14
	planID := int64(77)
	fixedNow := time.Date(2026, 5, 28, 9, 30, 0, 0, time.UTC)
	order := &dbent.PaymentOrder{
		ID:                         88,
		UserID:                     7,
		OrderType:                  payment.OrderTypeSubscription,
		PlanID:                     &planID,
		SubscriptionQuotaUsd:       &quota,
		SubscriptionDailyLimitUsd:  &daily,
		SubscriptionWeeklyLimitUsd: &weekly,
		SubscriptionScopeType:      &scopeType,
		SubscriptionScopeConfig:    map[string]any{"platforms": []any{PlatformAnthropic}},
		SubscriptionValidityDays:   &validityDays,
	}
	repo := &subscriptionPurchaseRepoStub{}
	ledger := &subscriptionPurchaseLedgerRepoStub{}
	svc := NewSubscriptionCreditPurchaseService(db, repo, ledger)
	svc.now = func() time.Time { return fixedNow }

	err = svc.FulfillOrder(context.Background(), order)

	require.NoError(t, err)
	require.Equal(t, int64(7), repo.lockUserID)
	require.NotNil(t, repo.inserted)
	require.Equal(t, int64(7), repo.inserted.UserID)
	require.Equal(t, &planID, repo.inserted.PlanID)
	require.Equal(t, SubscriptionScopePlatforms, repo.inserted.ScopeType)
	require.Equal(t, order.SubscriptionScopeConfig, repo.inserted.ScopeConfig)
	require.InDelta(t, 25.0, repo.inserted.QuotaLimitUSD, 1e-9)
	require.InDelta(t, 0.0, repo.inserted.QuotaUsedUSD, 1e-9)
	require.Equal(t, &daily, repo.inserted.DailyLimitUSD)
	require.Equal(t, &weekly, repo.inserted.WeeklyLimitUSD)
	require.Equal(t, fixedNow, repo.inserted.StartsAt)
	require.Equal(t, fixedNow.AddDate(0, 0, validityDays), repo.inserted.ExpiresAt)
	require.Equal(t, SubscriptionStatusActive, repo.inserted.Status)
	require.Nil(t, repo.inserted.ExhaustedAt)
	require.NotNil(t, ledger.entry)
	require.Equal(t, SubscriptionCreditLedgerPurchase, ledger.entry.Type)
	require.Equal(t, int64(9001), ledger.entry.SubscriptionID)
	require.Equal(t, &order.ID, ledger.entry.OrderID)
	require.InDelta(t, 25.0, ledger.entry.DeltaUSD, 1e-9)
	require.InDelta(t, 25.0, ledger.entry.RemainingAfterUSD, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionCreditPurchaseFulfillOrderBlocksWhenUsableSubscriptionExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectRollback()

	quota := 10.0
	scopeType := SubscriptionScopeAllAvailableGroups
	validityDays := 30
	planID := int64(9)
	order := &dbent.PaymentOrder{
		ID:                       99,
		UserID:                   7,
		OrderType:                payment.OrderTypeSubscription,
		PlanID:                   &planID,
		SubscriptionQuotaUsd:     &quota,
		SubscriptionScopeType:    &scopeType,
		SubscriptionScopeConfig:  map[string]any{},
		SubscriptionValidityDays: &validityDays,
	}
	repo := &subscriptionPurchaseRepoStub{hasUsable: true}
	ledger := &subscriptionPurchaseLedgerRepoStub{}
	svc := NewSubscriptionCreditPurchaseService(db, repo, ledger)

	err = svc.FulfillOrder(context.Background(), order)

	require.ErrorIs(t, err, ErrAlreadyHasUsableSubscription)
	require.Nil(t, repo.inserted)
	require.Nil(t, ledger.entry)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionCreditPurchaseFulfillOrderAllowsUsableSubscriptionWhenMultiplePurchasesEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectCommit()

	quota := 10.0
	scopeType := SubscriptionScopeAllAvailableGroups
	validityDays := 30
	planID := int64(9)
	order := &dbent.PaymentOrder{
		ID:                       100,
		UserID:                   7,
		OrderType:                payment.OrderTypeSubscription,
		PlanID:                   &planID,
		SubscriptionQuotaUsd:     &quota,
		SubscriptionScopeType:    &scopeType,
		SubscriptionScopeConfig:  map[string]any{},
		SubscriptionValidityDays: &validityDays,
	}
	repo := &subscriptionPurchaseRepoStub{hasUsable: true}
	ledger := &subscriptionPurchaseLedgerRepoStub{}
	svc := NewSubscriptionCreditPurchaseService(db, repo, ledger)
	svc.subscriptionMultiplePurchasesEnabled = func(context.Context) bool { return true }

	err = svc.FulfillOrder(context.Background(), order)

	require.NoError(t, err)
	require.NotNil(t, repo.inserted)
	require.NotNil(t, ledger.entry)
	require.NoError(t, mock.ExpectationsWereMet())
}
