package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SubscriptionCreditPurchaseService fulfills paid subscription-credit orders.
type SubscriptionCreditPurchaseService struct {
	tx                                   SQLTxBeginner
	repo                                 UserSubscriptionRepository
	ledgerRepo                           SubscriptionCreditLedgerRepository
	now                                  func() time.Time
	subscriptionMultiplePurchasesEnabled func(context.Context) bool
}

type SubscriptionCreditPurchaseFulfiller interface {
	FulfillOrder(ctx context.Context, order *dbent.PaymentOrder) error
}

func NewSubscriptionCreditPurchaseService(tx SQLTxBeginner, repo UserSubscriptionRepository, ledgerRepo SubscriptionCreditLedgerRepository) *SubscriptionCreditPurchaseService {
	return &SubscriptionCreditPurchaseService{
		tx:         tx,
		repo:       repo,
		ledgerRepo: ledgerRepo,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (s *SubscriptionCreditPurchaseService) SetSubscriptionMultiplePurchasesEnabledReader(fn func(context.Context) bool) *SubscriptionCreditPurchaseService {
	if s != nil {
		s.subscriptionMultiplePurchasesEnabled = fn
	}
	return s
}

func (s *SubscriptionCreditPurchaseService) FulfillOrder(ctx context.Context, order *dbent.PaymentOrder) error {
	if s == nil || s.tx == nil || s.repo == nil || s.ledgerRepo == nil {
		return infraerrors.InternalServer("SUBSCRIPTION_PURCHASE_SERVICE_UNAVAILABLE", "subscription purchase service is not configured")
	}
	sub, err := subscriptionFromOrderSnapshot(order, s.now())
	if err != nil {
		return err
	}

	tx, err := s.tx.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin subscription purchase tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := s.repo.LockUserForSubscriptionWrite(ctx, tx, order.UserID); err != nil {
		return err
	}
	if !s.multiplePurchasesEnabled(ctx) {
		hasUsable, err := s.repo.HasUsableCreditSubscription(ctx, order.UserID)
		if err != nil {
			return err
		}
		if hasUsable {
			return ErrAlreadyHasUsableSubscription
		}
	}

	created, err := s.repo.InsertCreditSubscription(ctx, tx, sub)
	if err != nil {
		return err
	}
	orderID := order.ID
	if err := s.ledgerRepo.Create(ctx, tx, &SubscriptionCreditLedgerEntry{
		UserID:            order.UserID,
		SubscriptionID:    created.ID,
		OrderID:           &orderID,
		Type:              SubscriptionCreditLedgerPurchase,
		DeltaUSD:          sub.QuotaLimitUSD,
		RemainingAfterUSD: sub.QuotaLimitUSD,
		Reason:            fmt.Sprintf("payment order %d", order.ID),
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit subscription purchase tx: %w", err)
	}
	committed = true
	return nil
}

func (s *SubscriptionCreditPurchaseService) multiplePurchasesEnabled(ctx context.Context) bool {
	return s != nil && s.subscriptionMultiplePurchasesEnabled != nil && s.subscriptionMultiplePurchasesEnabled(ctx)
}

func subscriptionFromOrderSnapshot(order *dbent.PaymentOrder, now time.Time) (*UserSubscription, error) {
	if order == nil {
		return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_ORDER", "subscription order is required")
	}
	if order.OrderType != payment.OrderTypeSubscription {
		return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_ORDER", "order is not a subscription order")
	}
	if order.PlanID == nil || order.SubscriptionQuotaUsd == nil || order.SubscriptionScopeType == nil || order.SubscriptionValidityDays == nil {
		return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_ORDER", "subscription order snapshot is incomplete")
	}
	if *order.SubscriptionQuotaUsd <= 0 || *order.SubscriptionValidityDays <= 0 {
		return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_ORDER", "subscription order snapshot is invalid")
	}
	scopeType := *order.SubscriptionScopeType
	if scopeType == "" {
		scopeType = SubscriptionScopeAllAvailableGroups
	}
	scopeConfig := cloneSubscriptionScopeConfig(order.SubscriptionScopeConfig)
	if scopeConfig == nil {
		scopeConfig = map[string]any{}
	}
	now = now.UTC()
	return &UserSubscription{
		UserID:         order.UserID,
		PlanID:         order.PlanID,
		ScopeType:      scopeType,
		ScopeConfig:    scopeConfig,
		QuotaLimitUSD:  *order.SubscriptionQuotaUsd,
		QuotaUsedUSD:   0,
		DailyLimitUSD:  cloneFloat64Ptr(order.SubscriptionDailyLimitUsd),
		WeeklyLimitUSD: cloneFloat64Ptr(order.SubscriptionWeeklyLimitUsd),
		StartsAt:       now,
		ExpiresAt:      now.AddDate(0, 0, *order.SubscriptionValidityDays),
		Status:         SubscriptionStatusActive,
		AssignedAt:     now,
		Notes:          fmt.Sprintf("payment order %d", order.ID),
	}, nil
}

func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneSubscriptionScopeConfig(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isAlreadyHasUsableSubscription(err error) bool {
	return errors.Is(err, ErrAlreadyHasUsableSubscription)
}
