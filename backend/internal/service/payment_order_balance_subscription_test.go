//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type subscriptionBalancePaymentFulfillerStub struct {
	order *dbent.PaymentOrder
	err   error
}

func (s *subscriptionBalancePaymentFulfillerStub) FulfillOrder(_ context.Context, order *dbent.PaymentOrder) error {
	if order != nil {
		cp := *order
		s.order = &cp
	}
	return s.err
}

func TestCreateOrderSubscriptionBalancePaymentRecordsLedgerAndAudit(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("balance-subscription@example.com").
		SetPasswordHash("hash").
		SetUsername("balance-user").
		SetStatus(payment.EntityStatusActive).
		SetBalance(40).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetName("Starter").
		SetDescription("starter plan").
		SetPrice(10).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetFeatures("[]").
		SetProductName("Starter Plan").
		SetQuotaUsd(25).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
		configService: &PaymentConfigService{
			entClient: client,
			settingRepo: &paymentConfigSettingRepoStub{
				values: map[string]string{
					SettingPaymentEnabled:                    "true",
					SettingSubscriptionBalancePaymentEnabled: "true",
					SettingBalanceRechargeMult:               "1.5",
				},
			},
		},
		userRepo: &mockUserRepo{
			getByIDUser: &User{
				ID:       user.ID,
				Email:    user.Email,
				Username: user.Username,
				Status:   payment.EntityStatusActive,
				Balance:  40,
			},
		},
		subscriptionCreditPurchaseSvc: &subscriptionBalancePaymentFulfillerStub{},
	}

	resp, err := svc.CreateOrder(ctx, CreateOrderRequest{
		UserID:      user.ID,
		PaymentType: payment.TypeBalance,
		OrderType:   payment.OrderTypeSubscription,
		PlanID:      plan.ID,
	})
	require.NoError(t, err)
	require.Equal(t, payment.TypeBalance, resp.PaymentType)
	require.InDelta(t, 10, resp.Amount, 1e-9)
	require.InDelta(t, 15, resp.PayAmount, 1e-9)
	require.Equal(t, OrderStatusCompleted, resp.Status)

	updatedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 25, updatedUser.Balance, 1e-9)

	order, err := client.PaymentOrder.Get(ctx, resp.OrderID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, order.Status)
	require.Equal(t, payment.TypeBalance, order.PaymentType)
	require.InDelta(t, 10, order.Amount, 1e-9)
	require.InDelta(t, 15, order.PayAmount, 1e-9)

	ledger, err := client.RedeemCode.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, RedeemTypeBalancePayment, ledger.Type)
	require.InDelta(t, -15, ledger.Value, 1e-9)
	require.Equal(t, user.ID, *ledger.UsedBy)
	require.NotNil(t, ledger.Notes)
	require.Contains(t, *ledger.Notes, "Starter Plan")
	require.Contains(t, *ledger.Notes, "扣除 $15.00")
	require.Contains(t, *ledger.Notes, "剩余 $25.00")

	auditEntry, err := client.PaymentAuditLog.Query().
		Where(
			paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)),
			paymentauditlog.ActionEQ("BALANCE_PAYMENT_RECORDED"),
		).
		Only(ctx)
	require.NoError(t, err)

	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(auditEntry.Detail), &detail))
	require.Equal(t, "Starter Plan", detail["item"])
	require.Equal(t, float64(10), detail["plan_price_cny"])
	require.Equal(t, float64(15), detail["balance_spent"])
	require.Equal(t, float64(25), detail["balance_after"])
}

func TestCreateOrderSubscriptionPurchaseForbiddenWhenUserBanned(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("sub-purchase-banned@example.com").
		SetPasswordHash("hash").
		SetUsername("banned-buyer").
		SetStatus(payment.EntityStatusActive).
		SetBalance(40).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetName("Starter").
		SetDescription("starter plan").
		SetPrice(10).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetFeatures("[]").
		SetProductName("Starter Plan").
		SetQuotaUsd(25).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
		configService: &PaymentConfigService{
			entClient: client,
			settingRepo: &paymentConfigSettingRepoStub{
				values: map[string]string{
					SettingPaymentEnabled:                    "true",
					SettingSubscriptionBalancePaymentEnabled: "true",
				},
			},
		},
		userRepo: &mockUserRepo{
			getByIDUser: &User{
				ID:                           user.ID,
				Email:                        user.Email,
				Username:                     user.Username,
				Status:                       payment.EntityStatusActive,
				Balance:                      40,
				SubscriptionPurchaseDisabled: true,
			},
		},
		subscriptionCreditPurchaseSvc: &subscriptionBalancePaymentFulfillerStub{},
	}

	_, err = svc.CreateOrder(ctx, CreateOrderRequest{
		UserID:      user.ID,
		PaymentType: payment.TypeBalance,
		OrderType:   payment.OrderTypeSubscription,
		PlanID:      plan.ID,
	})
	require.Error(t, err)
	require.Equal(t, "SUBSCRIPTION_PURCHASE_FORBIDDEN", infraerrors.FromError(err).Reason)
}

func TestCreateOrderSubscriptionBalancePaymentRequiresToggle(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("balance-toggle@example.com").
		SetPasswordHash("hash").
		SetUsername("toggle-user").
		SetStatus(payment.EntityStatusActive).
		SetBalance(20).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetName("Starter").
		SetDescription("starter plan").
		SetPrice(10).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetFeatures("[]").
		SetProductName("Starter Plan").
		SetQuotaUsd(25).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
		configService: &PaymentConfigService{
			entClient: client,
			settingRepo: &paymentConfigSettingRepoStub{
				values: map[string]string{
					SettingPaymentEnabled:                    "true",
					SettingSubscriptionBalancePaymentEnabled: "false",
				},
			},
		},
		userRepo: &mockUserRepo{
			getByIDUser: &User{
				ID:       user.ID,
				Email:    user.Email,
				Username: user.Username,
				Status:   payment.EntityStatusActive,
				Balance:  20,
			},
		},
	}

	_, err = svc.CreateOrder(ctx, CreateOrderRequest{
		UserID:      user.ID,
		PaymentType: payment.TypeBalance,
		OrderType:   payment.OrderTypeSubscription,
		PlanID:      plan.ID,
	})
	require.Error(t, err)
	require.Equal(t, "SUBSCRIPTION_BALANCE_PAYMENT_DISABLED", infraerrors.FromError(err).Reason)
}

func TestCreateOrderSubscriptionBalancePaymentRequiresEnoughBalance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("balance-not-enough@example.com").
		SetPasswordHash("hash").
		SetUsername("short-user").
		SetStatus(payment.EntityStatusActive).
		SetBalance(14.99).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetName("Starter").
		SetDescription("starter plan").
		SetPrice(10).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetFeatures("[]").
		SetProductName("Starter Plan").
		SetQuotaUsd(25).
		SetScopeType(SubscriptionScopeAllAvailableGroups).
		SetScopeConfig(map[string]any{}).
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
		configService: &PaymentConfigService{
			entClient: client,
			settingRepo: &paymentConfigSettingRepoStub{
				values: map[string]string{
					SettingPaymentEnabled:                    "true",
					SettingSubscriptionBalancePaymentEnabled: "true",
					SettingBalanceRechargeMult:               "1.5",
				},
			},
		},
		userRepo: &mockUserRepo{
			getByIDUser: &User{
				ID:       user.ID,
				Email:    user.Email,
				Username: user.Username,
				Status:   payment.EntityStatusActive,
				Balance:  14.99,
			},
		},
	}

	_, err = svc.CreateOrder(ctx, CreateOrderRequest{
		UserID:      user.ID,
		PaymentType: payment.TypeBalance,
		OrderType:   payment.OrderTypeSubscription,
		PlanID:      plan.ID,
	})
	require.Error(t, err)
	appErr := infraerrors.FromError(err)
	require.Equal(t, "INSUFFICIENT_BALANCE", appErr.Reason)
	require.Equal(t, "15.00", appErr.Metadata["required_balance"])
}
