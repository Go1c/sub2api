package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type adminUpdateUserSubRepoStub struct {
	sub     *UserSubscription
	updated *UserSubscription
}

func (r *adminUpdateUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *adminUpdateUserSubRepoStub) Update(_ context.Context, sub *UserSubscription) error {
	cp := *sub
	r.updated = &cp
	r.sub = &cp
	return nil
}

func (r *adminUpdateUserSubRepoStub) Create(context.Context, *UserSubscription) error {
	panic("unexpected Create")
}
func (r *adminUpdateUserSubRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetByUserIDAndGroupID")
}
func (r *adminUpdateUserSubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetActiveByUserIDAndGroupID")
}
func (r *adminUpdateUserSubRepoStub) Delete(context.Context, int64) error { panic("unexpected Delete") }
func (r *adminUpdateUserSubRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListByUserID")
}
func (r *adminUpdateUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListActiveByUserID")
}
func (r *adminUpdateUserSubRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID")
}
func (r *adminUpdateUserSubRepoStub) List(context.Context, pagination.PaginationParams, UserSubscriptionListFilters) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (r *adminUpdateUserSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	panic("unexpected ExistsByUserIDAndGroupID")
}
func (r *adminUpdateUserSubRepoStub) ExtendExpiry(context.Context, int64, time.Time) error {
	panic("unexpected ExtendExpiry")
}
func (r *adminUpdateUserSubRepoStub) UpdateStatus(context.Context, int64, string) error {
	panic("unexpected UpdateStatus")
}
func (r *adminUpdateUserSubRepoStub) UpdateNotes(context.Context, int64, string) error {
	panic("unexpected UpdateNotes")
}
func (r *adminUpdateUserSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error {
	panic("unexpected ActivateWindows")
}
func (r *adminUpdateUserSubRepoStub) ResetUsageWindows(context.Context, int64, bool, bool, bool, time.Time) error {
	panic("unexpected ResetUsageWindows")
}
func (r *adminUpdateUserSubRepoStub) UserResetWeeklyLimit(context.Context, int64, int64, time.Time, time.Time) (int, error) {
	panic("unexpected UserResetWeeklyLimit")
}
func (r *adminUpdateUserSubRepoStub) ResetDailyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetDailyUsage")
}
func (r *adminUpdateUserSubRepoStub) ResetWeeklyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetWeeklyUsage")
}
func (r *adminUpdateUserSubRepoStub) ResetMonthlyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetMonthlyUsage")
}
func (r *adminUpdateUserSubRepoStub) IncrementUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementUsage")
}
func (r *adminUpdateUserSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	panic("unexpected BatchUpdateExpiredStatus")
}
func (r *adminUpdateUserSubRepoStub) GetUsableCreditSubscription(context.Context, int64) (*UserSubscription, error) {
	panic("unexpected GetUsableCreditSubscription")
}
func (r *adminUpdateUserSubRepoStub) ListUsableCreditSubscriptions(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListUsableCreditSubscriptions")
}
func (r *adminUpdateUserSubRepoStub) HasUsableCreditSubscription(context.Context, int64) (bool, error) {
	panic("unexpected HasUsableCreditSubscription")
}
func (r *adminUpdateUserSubRepoStub) GetRenewalEligibility(context.Context, int64) (RenewalEligibility, error) {
	panic("unexpected GetRenewalEligibility")
}
func (r *adminUpdateUserSubRepoStub) LockUserForSubscriptionWrite(context.Context, *sql.Tx, int64) error {
	panic("unexpected LockUserForSubscriptionWrite")
}
func (r *adminUpdateUserSubRepoStub) InsertCreditSubscription(context.Context, *sql.Tx, *UserSubscription) (*UserSubscription, error) {
	panic("unexpected InsertCreditSubscription")
}
func (r *adminUpdateUserSubRepoStub) ExpireCreditSubscriptions(context.Context) (int64, error) {
	panic("unexpected ExpireCreditSubscriptions")
}
func (r *adminUpdateUserSubRepoStub) MarkExpiredCreditLogged(context.Context, int64, time.Time) error {
	panic("unexpected MarkExpiredCreditLogged")
}

type adminUpdateLedgerRepoStub struct {
	entries []SubscriptionCreditLedgerEntry
	err     error
}

func (r *adminUpdateLedgerRepoStub) Create(_ context.Context, _ SQLExecer, entry *SubscriptionCreditLedgerEntry) error {
	if r.err != nil {
		return r.err
	}
	cp := *entry
	r.entries = append(r.entries, cp)
	return nil
}

func (r *adminUpdateLedgerRepoStub) CreateLimitReachedEvent(context.Context, SQLExecer, *SubscriptionCreditLedgerEntry) (bool, error) {
	panic("unexpected CreateLimitReachedEvent")
}

func (r *adminUpdateLedgerRepoStub) ListByUserID(context.Context, int64, string, pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID")
}

func (r *adminUpdateLedgerRepoStub) ListBySubscriptionID(context.Context, int64, string, pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	panic("unexpected ListBySubscriptionID")
}

func TestAdminSubscriptionUpdateQuotaCreatesDeltaLedger(t *testing.T) {
	repo := &adminUpdateUserSubRepoStub{sub: &UserSubscription{
		ID:            42,
		UserID:        7,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  4,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}}
	ledger := &adminUpdateLedgerRepoStub{}
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	svc.SetSubscriptionCreditLedgerRepository(ledger)
	nextLimit := 15.0

	got, err := svc.AdminUpdateSubscription(context.Background(), 42, AdminUpdateSubscriptionInput{
		QuotaLimitUSD: &nextLimit,
		Reason:        "manual correction",
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.InDelta(t, 15.0, got.QuotaLimitUSD, 1e-9)
	require.Len(t, ledger.entries, 1)
	require.Equal(t, SubscriptionCreditLedgerAdminAdjust, ledger.entries[0].Type)
	require.InDelta(t, 5.0, ledger.entries[0].DeltaUSD, 1e-9)
	require.InDelta(t, 11.0, ledger.entries[0].RemainingAfterUSD, 1e-9)
	require.Equal(t, "manual correction", ledger.entries[0].Reason)
}

func TestAdminSubscriptionUpdateRejectsInvalidQuotaBeforeMutation(t *testing.T) {
	repo := &adminUpdateUserSubRepoStub{sub: &UserSubscription{
		ID:            42,
		UserID:        7,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  4,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}}
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	nextLimit := -1.0

	got, err := svc.AdminUpdateSubscription(context.Background(), 42, AdminUpdateSubscriptionInput{
		QuotaLimitUSD: &nextLimit,
		Reason:        "bad correction",
	})

	require.Error(t, err)
	require.Nil(t, got)
	require.Nil(t, repo.updated)
}

func TestAdminSubscriptionUpdateLedgerFailureDoesNotMutateQuota(t *testing.T) {
	repo := &adminUpdateUserSubRepoStub{sub: &UserSubscription{
		ID:            42,
		UserID:        7,
		QuotaLimitUSD: 10,
		QuotaUsedUSD:  4,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}}
	ledger := &adminUpdateLedgerRepoStub{err: errors.New("ledger unavailable")}
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	svc.SetSubscriptionCreditLedgerRepository(ledger)
	nextLimit := 15.0

	got, err := svc.AdminUpdateSubscription(context.Background(), 42, AdminUpdateSubscriptionInput{
		QuotaLimitUSD: &nextLimit,
		Reason:        "manual correction",
	})

	require.Error(t, err)
	require.Nil(t, got)
	require.Nil(t, repo.updated)
	require.InDelta(t, 10.0, repo.sub.QuotaLimitUSD, 1e-9)
}

func TestProvideSubscriptionServiceWiresLedgerRepository(t *testing.T) {
	repo := &adminUpdateUserSubRepoStub{}
	ledger := &adminUpdateLedgerRepoStub{}

	svc := ProvideSubscriptionService(nil, repo, ledger, nil, nil, nil, nil)

	require.NotNil(t, svc)
	require.Same(t, ledger, svc.creditLedgerRepo)
}
