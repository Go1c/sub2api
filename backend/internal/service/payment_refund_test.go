//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type refundCreditSubscriptionRepoStub struct {
	userSubRepoNoop
	byID map[int64]*UserSubscription
}

func (r *refundCreditSubscriptionRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	sub, ok := r.byID[id]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	cp := *sub
	return &cp, nil
}

func (r *refundCreditSubscriptionRepoStub) Update(_ context.Context, sub *UserSubscription) error {
	if sub == nil {
		return ErrSubscriptionNilInput
	}
	cp := *sub
	r.byID[sub.ID] = &cp
	return nil
}

func createCreditSubscriptionRefundFixture(t *testing.T, ctx context.Context, client *dbent.Client, paymentTradeNo string) (*dbent.PaymentOrder, *UserSubscription) {
	t.Helper()

	user := client.User.Create().
		SetEmail("credit-refund@example.com").
		SetPasswordHash("hash").
		SetUsername("credit-refund-user").
		SaveX(ctx)

	quota := 25.0
	scopeType := SubscriptionScopeAllAvailableGroups
	validityDays := 30
	now := time.Now().UTC()

	order := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetRechargeCode("REFUND-CREDIT-ORDER").
		SetOutTradeNo("sub2_credit_refund_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo(paymentTradeNo).
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusCompleted).
		SetPaidAt(now).
		SetCompletedAt(now).
		SetExpiresAt(now.Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test").
		SetPlanID(42).
		SetSubscriptionQuotaUsd(quota).
		SetSubscriptionScopeType(scopeType).
		SetSubscriptionScopeConfig(map[string]any{}).
		SetSubscriptionValidityDays(validityDays).
		SaveX(ctx)

	subEnt := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetPlanID(42).
		SetScopeType(scopeType).
		SetScopeConfig(map[string]any{}).
		SetQuotaLimitUsd(quota).
		SetQuotaUsedUsd(0).
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, validityDays)).
		SetStatus(SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("payment order fixture").
		SaveX(ctx)

	client.SubscriptionCreditLedger.Create().
		SetUserID(user.ID).
		SetSubscriptionID(subEnt.ID).
		SetOrderID(order.ID).
		SetType(SubscriptionCreditLedgerPurchase).
		SetDeltaUsd(quota).
		SetBalanceDeltaUsd(0).
		SetRemainingAfterUsd(quota).
		SetReason("payment order fixture").
		ExecX(ctx)

	return order, &UserSubscription{
		ID:            subEnt.ID,
		UserID:        user.ID,
		PlanID:        order.PlanID,
		ScopeType:     scopeType,
		ScopeConfig:   map[string]any{},
		QuotaLimitUSD: quota,
		QuotaUsedUSD:  0,
		StartsAt:      now,
		ExpiresAt:     now.AddDate(0, 0, validityDays),
		Status:        SubscriptionStatusActive,
		AssignedAt:    now,
		Notes:         "payment order fixture",
	}
}

func TestValidateRefundRequestRejectsLegacyGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-legacy@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-legacy-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-LEGACY-ORDER").
		SetOutTradeNo("sub2_refund_legacy_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-legacy-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	_, err = svc.validateRefundRequest(ctx, order.ID, user.ID)
	require.Error(t, err)
	require.Equal(t, "USER_REFUND_DISABLED", infraerrors.Reason(err))
}

func TestPrepareRefundRejectsLegacyGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-legacy-admin@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-legacy-admin-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-admin-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(188).
		SetPayAmount(188).
		SetFeeRate(0).
		SetRechargeCode("REFUND-LEGACY-ADMIN-ORDER").
		SetOutTradeNo("sub2_refund_legacy_admin_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-legacy-admin-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	plan, result, err := svc.PrepareRefund(ctx, order.ID, 0, "", false, false)
	require.Nil(t, plan)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "REFUND_DISABLED", infraerrors.Reason(err))
}

func TestExecuteRefundRevokesCreditedSubscriptionByOrderID(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order, sub := createCreditSubscriptionRefundFixture(t, ctx, client, "")
	repo := &refundCreditSubscriptionRepoStub{
		byID: map[int64]*UserSubscription{sub.ID: sub},
	}
	subSvc := NewSubscriptionService(nil, repo, nil, nil, nil)
	svc := &PaymentService{
		entClient:       client,
		subscriptionSvc: subSvc,
	}

	plan := &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.Amount,
		Reason:        "credit refund",
	}
	require.Nil(t, svc.prepDeduct(ctx, order, plan, false))

	result, err := svc.ExecuteRefund(ctx, plan)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "revoked", repo.byID[sub.ID].Status)
	gotOrder := client.PaymentOrder.GetX(ctx, order.ID)
	require.Equal(t, OrderStatusRefunded, gotOrder.Status)
}

func TestExecuteRefundRestoresCreditedSubscriptionWhenGatewayFails(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order, sub := createCreditSubscriptionRefundFixture(t, ctx, client, "trade-credit-refund")
	repo := &refundCreditSubscriptionRepoStub{
		byID: map[int64]*UserSubscription{sub.ID: sub},
	}
	subSvc := NewSubscriptionService(nil, repo, nil, nil, nil)
	svc := &PaymentService{
		entClient:       client,
		subscriptionSvc: subSvc,
	}

	plan := &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.Amount,
		Reason:        "credit refund rollback",
	}
	require.Nil(t, svc.prepDeduct(ctx, order, plan, false))

	result, err := svc.ExecuteRefund(ctx, plan)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Warning, "rolled back")
	require.Equal(t, SubscriptionStatusActive, repo.byID[sub.ID].Status)
	gotOrder := client.PaymentOrder.GetX(ctx, order.ID)
	require.Equal(t, OrderStatusCompleted, gotOrder.Status)
}

func TestGwRefundRejectsAlipayMerchantIdentitySnapshotMismatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-snapshot-mismatch@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-snapshot-mismatch-user").
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-mismatch-instance").
		SetConfig(encryptWebhookProviderConfig(t, map[string]string{
			"appId":      "runtime-alipay-app",
			"privateKey": "runtime-private-key",
		})).
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	instID := strconv.FormatInt(inst.ID, 10)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-SNAPSHOT-MISMATCH-ORDER").
		SetOutTradeNo("sub2_refund_snapshot_mismatch_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-refund-snapshot-mismatch").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(instID).
		SetProviderKey(payment.TypeAlipay).
		SetProviderSnapshot(map[string]any{
			"schema_version":       2,
			"provider_instance_id": instID,
			"provider_key":         payment.TypeAlipay,
			"merchant_app_id":      "expected-alipay-app",
		}).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient:    client,
		loadBalancer: newWebhookProviderTestLoadBalancer(client),
	}

	_, err = svc.gwRefund(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.Amount,
		Reason:        "snapshot mismatch",
	})
	require.ErrorContains(t, err, "alipay app_id mismatch")
}
