//go:build unit

package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type paymentFulfillmentTestProvider struct {
	key            string
	supportedTypes []payment.PaymentType
}

func (p paymentFulfillmentTestProvider) Name() string        { return p.key }
func (p paymentFulfillmentTestProvider) ProviderKey() string { return p.key }
func (p paymentFulfillmentTestProvider) SupportedTypes() []payment.PaymentType {
	return p.supportedTypes
}
func (p paymentFulfillmentTestProvider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	panic("unexpected call")
}
func (p paymentFulfillmentTestProvider) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	panic("unexpected call")
}
func (p paymentFulfillmentTestProvider) VerifyNotification(ctx context.Context, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	panic("unexpected call")
}
func (p paymentFulfillmentTestProvider) Refund(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	panic("unexpected call")
}

// ---------------------------------------------------------------------------
// resolveRedeemAction — pure idempotency decision logic
// ---------------------------------------------------------------------------

func TestResolveRedeemAction_CodeNotFound(t *testing.T) {
	t.Parallel()
	action := resolveRedeemAction(nil, nil)
	assert.Equal(t, redeemActionCreate, action, "nil code with nil error should create")
}

func TestResolveRedeemAction_LookupError(t *testing.T) {
	t.Parallel()
	action := resolveRedeemAction(nil, errors.New("db connection lost"))
	assert.Equal(t, redeemActionCreate, action, "lookup error should fall back to create")
}

func TestResolveRedeemAction_LookupErrorWithNonNilCode(t *testing.T) {
	t.Parallel()
	// Edge case: both code and error are non-nil (shouldn't happen in practice,
	// but the function should still treat error as authoritative)
	code := &RedeemCode{Status: StatusUnused}
	action := resolveRedeemAction(code, errors.New("partial error"))
	assert.Equal(t, redeemActionCreate, action, "non-nil error should always result in create regardless of code")
}

func TestResolveRedeemAction_CodeExistsAndUsed(t *testing.T) {
	t.Parallel()
	code := &RedeemCode{
		Code:   "test-code-123",
		Status: StatusUsed,
		Type:   RedeemTypeBalance,
		Value:  10.0,
	}
	action := resolveRedeemAction(code, nil)
	assert.Equal(t, redeemActionSkipCompleted, action, "used code should skip to completed")
}

func TestResolveRedeemAction_CodeExistsAndUnused(t *testing.T) {
	t.Parallel()
	code := &RedeemCode{
		Code:   "test-code-456",
		Status: StatusUnused,
		Type:   RedeemTypeBalance,
		Value:  25.0,
	}
	action := resolveRedeemAction(code, nil)
	assert.Equal(t, redeemActionRedeem, action, "unused code should skip creation and proceed to redeem")
}

func TestResolveRedeemAction_CodeExistsWithExpiredStatus(t *testing.T) {
	t.Parallel()
	// A code with a non-standard status (neither "unused" nor "used")
	// should NOT be treated as used, so it falls through to redeemActionRedeem.
	code := &RedeemCode{
		Code:   "expired-code",
		Status: StatusExpired,
	}
	action := resolveRedeemAction(code, nil)
	assert.Equal(t, redeemActionRedeem, action, "expired-status code is not IsUsed(), should redeem")
}

// ---------------------------------------------------------------------------
// Table-driven comprehensive test
// ---------------------------------------------------------------------------

func TestResolveRedeemAction_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     *RedeemCode
		err      error
		expected redeemAction
	}{
		{
			name:     "nil code, nil error — first run",
			code:     nil,
			err:      nil,
			expected: redeemActionCreate,
		},
		{
			name:     "nil code, lookup error — treat as not found",
			code:     nil,
			err:      ErrRedeemCodeNotFound,
			expected: redeemActionCreate,
		},
		{
			name:     "nil code, generic DB error — treat as not found",
			code:     nil,
			err:      errors.New("connection refused"),
			expected: redeemActionCreate,
		},
		{
			name:     "code exists, used — previous run completed redeem",
			code:     &RedeemCode{Status: StatusUsed},
			err:      nil,
			expected: redeemActionSkipCompleted,
		},
		{
			name:     "code exists, unused — previous run created code but crashed before redeem",
			code:     &RedeemCode{Status: StatusUnused},
			err:      nil,
			expected: redeemActionRedeem,
		},
		{
			name:     "code exists but error also set — error takes precedence",
			code:     &RedeemCode{Status: StatusUsed},
			err:      errors.New("unexpected"),
			expected: redeemActionCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveRedeemAction(tt.code, tt.err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// redeemAction enum value sanity
// ---------------------------------------------------------------------------

func TestRedeemAction_DistinctValues(t *testing.T) {
	t.Parallel()
	// Ensure the three actions have distinct values (iota correctness)
	assert.NotEqual(t, redeemActionCreate, redeemActionRedeem)
	assert.NotEqual(t, redeemActionCreate, redeemActionSkipCompleted)
	assert.NotEqual(t, redeemActionRedeem, redeemActionSkipCompleted)
}

// ---------------------------------------------------------------------------
// RedeemCode.IsUsed / CanUse interaction with resolveRedeemAction
// ---------------------------------------------------------------------------

func TestResolveRedeemAction_IsUsedCanUseConsistency(t *testing.T) {
	t.Parallel()

	usedCode := &RedeemCode{Status: StatusUsed}
	unusedCode := &RedeemCode{Status: StatusUnused}

	// Verify our decision function is consistent with the domain model methods
	assert.True(t, usedCode.IsUsed())
	assert.False(t, usedCode.CanUse())
	assert.Equal(t, redeemActionSkipCompleted, resolveRedeemAction(usedCode, nil))

	assert.False(t, unusedCode.IsUsed())
	assert.True(t, unusedCode.CanUse())
	assert.Equal(t, redeemActionRedeem, resolveRedeemAction(unusedCode, nil))
}

func TestExpectedNotificationProviderKeyPrefersOrderInstanceProvider(t *testing.T) {
	t.Parallel()

	registry := payment.NewRegistry()
	registry.Register(paymentFulfillmentTestProvider{
		key:            payment.TypeAlipay,
		supportedTypes: []payment.PaymentType{payment.TypeAlipay},
	})

	assert.Equal(t,
		payment.TypeEasyPay,
		expectedNotificationProviderKey(registry, payment.TypeAlipay, "", payment.TypeEasyPay),
	)
}

func TestExpectedNotificationProviderKeyUsesRegistryMappingForLegacyOrders(t *testing.T) {
	t.Parallel()

	registry := payment.NewRegistry()
	registry.Register(paymentFulfillmentTestProvider{
		key:            payment.TypeEasyPay,
		supportedTypes: []payment.PaymentType{payment.TypeAlipay},
	})

	assert.Equal(t,
		payment.TypeEasyPay,
		expectedNotificationProviderKey(registry, payment.TypeAlipay, "", ""),
	)
}

func TestExpectedNotificationProviderKeyFallsBackToPaymentType(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		payment.TypeWxpay,
		expectedNotificationProviderKey(nil, payment.TypeWxpay, "", ""),
	)
}

func TestExpectedNotificationProviderKeyPrefersOrderSnapshotProviderKey(t *testing.T) {
	t.Parallel()

	registry := payment.NewRegistry()
	registry.Register(paymentFulfillmentTestProvider{
		key:            payment.TypeAlipay,
		supportedTypes: []payment.PaymentType{payment.TypeAlipay},
	})

	assert.Equal(t,
		payment.TypeEasyPay,
		expectedNotificationProviderKey(registry, payment.TypeAlipay, payment.TypeEasyPay, ""),
	)
}

func TestExpectedNotificationProviderKeyForOrderUsesSnapshotProviderKey(t *testing.T) {
	t.Parallel()

	registry := payment.NewRegistry()
	registry.Register(paymentFulfillmentTestProvider{
		key:            payment.TypeAlipay,
		supportedTypes: []payment.PaymentType{payment.TypeAlipay},
	})

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeAlipay,
		ProviderSnapshot: map[string]any{
			"schema_version": 1,
			"provider_key":   payment.TypeEasyPay,
		},
	}

	assert.Equal(t,
		payment.TypeEasyPay,
		expectedNotificationProviderKeyForOrder(registry, order, ""),
	)
}

func TestValidateProviderNotificationMetadataRejectsWxpaySnapshotMismatch(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeWxpay,
		ProviderSnapshot: map[string]any{
			"schema_version":  1,
			"merchant_app_id": "wx-app-expected",
			"merchant_id":     "mch-expected",
			"currency":        "CNY",
		},
	}

	err := validateProviderNotificationMetadata(order, payment.TypeWxpay, map[string]string{
		"appid":       "wx-app-other",
		"mchid":       "mch-expected",
		"currency":    "CNY",
		"trade_state": "SUCCESS",
	})
	assert.ErrorContains(t, err, "wxpay appid mismatch")
}

func TestValidateProviderNotificationMetadataAllowsLegacyOrdersWithoutSnapshotFields(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeWxpay,
		ProviderSnapshot: map[string]any{
			"schema_version":       1,
			"provider_instance_id": "9",
			"provider_key":         payment.TypeWxpay,
		},
	}

	err := validateProviderNotificationMetadata(order, payment.TypeWxpay, map[string]string{
		"appid":       "wx-app-runtime",
		"mchid":       "mch-runtime",
		"currency":    "CNY",
		"trade_state": "SUCCESS",
	})
	assert.NoError(t, err)
}

func TestParseLegacyPaymentOrderID(t *testing.T) {
	t.Parallel()

	oid, ok := parseLegacyPaymentOrderID("sub2_42", &dbent.NotFoundError{})
	assert.True(t, ok)
	assert.EqualValues(t, 42, oid)

	_, ok = parseLegacyPaymentOrderID("42", &dbent.NotFoundError{})
	assert.False(t, ok)

	_, ok = parseLegacyPaymentOrderID("sub2_42", errors.New("db down"))
	assert.False(t, ok)
}

func TestIsValidProviderAmount(t *testing.T) {
	t.Parallel()

	assert.True(t, isValidProviderAmount(0.01))
	assert.False(t, isValidProviderAmount(0))
	assert.False(t, isValidProviderAmount(-1))
	assert.False(t, isValidProviderAmount(math.NaN()))
	assert.False(t, isValidProviderAmount(math.Inf(1)))
}

func TestResolveNotificationPaymentAmountAllowsMapayCallbackOrderAmount(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		Amount:    1.13,
		PayAmount: 1.14,
	}

	amount, ok := resolveNotificationPaymentAmount(order, payment.TypeMapay, 1.13)

	assert.True(t, ok)
	assert.InDelta(t, 1.14, amount, amountToleranceCNY)
}

func TestResolveNotificationPaymentAmountRejectsNonMapayOrderAmountMismatch(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		Amount:    1.13,
		PayAmount: 1.14,
	}

	_, ok := resolveNotificationPaymentAmount(order, payment.TypeEasyPay, 1.13)

	assert.False(t, ok)
}

func TestIsAffiliateRebateEligibleOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		order *dbent.PaymentOrder
		want  bool
	}{
		{
			name:  "balance recharge is eligible",
			order: &dbent.PaymentOrder{OrderType: payment.OrderTypeBalance, PaymentType: payment.TypeAlipay},
			want:  true,
		},
		{
			name:  "external subscription payment is eligible",
			order: &dbent.PaymentOrder{OrderType: payment.OrderTypeSubscription, PaymentType: payment.TypeWxpay},
			want:  true,
		},
		{
			name:  "balance paid subscription is not eligible",
			order: &dbent.PaymentOrder{OrderType: payment.OrderTypeSubscription, PaymentType: payment.TypeBalance},
			want:  false,
		},
		{
			name:  "unknown order type is not eligible",
			order: &dbent.PaymentOrder{OrderType: "other", PaymentType: payment.TypeAlipay},
			want:  false,
		},
		{
			name:  "nil order is not eligible",
			order: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isAffiliateRebateEligibleOrder(tt.order))
		})
	}
}

func TestAffiliateRebateAuditClaimSQLCastsParametersAsText(t *testing.T) {
	t.Parallel()

	sql := affiliateRebateAuditClaimSQL()

	require.Contains(t, sql, "CAST($1 AS TEXT)")
	require.Contains(t, sql, "CAST($2 AS TEXT)")
}

func TestValidateProviderNotificationMetadataRejectsAlipaySnapshotMismatch(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeAlipay,
		ProviderSnapshot: map[string]any{
			"schema_version":  2,
			"merchant_app_id": "alipay-app-expected",
		},
	}

	err := validateProviderNotificationMetadata(order, payment.TypeAlipay, map[string]string{
		"app_id": "alipay-app-other",
	})
	assert.ErrorContains(t, err, "alipay app_id mismatch")
}

func TestValidateProviderNotificationMetadataRejectsEasyPaySnapshotMismatch(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeAlipay,
		ProviderSnapshot: map[string]any{
			"schema_version": 2,
			"merchant_id":    "pid-expected",
		},
	}

	err := validateProviderNotificationMetadata(order, payment.TypeEasyPay, map[string]string{
		"pid": "pid-other",
	})
	assert.ErrorContains(t, err, "easypay pid mismatch")
}

func TestValidateProviderNotificationMetadataRejectsMapaySnapshotMismatch(t *testing.T) {
	t.Parallel()

	order := &dbent.PaymentOrder{
		PaymentType: payment.TypeAlipay,
		ProviderSnapshot: map[string]any{
			"schema_version": 2,
			"provider_key":   payment.TypeMapay,
			"merchant_id":    "pid-expected",
		},
	}

	err := validateProviderNotificationMetadata(order, payment.TypeMapay, map[string]string{
		"pid": "pid-other",
	})
	assert.ErrorContains(t, err, "mapay pid mismatch")
}

type subscriptionCreditPurchaseFulfillerStub struct {
	order *dbent.PaymentOrder
	err   error
}

func (s *subscriptionCreditPurchaseFulfillerStub) FulfillOrder(ctx context.Context, order *dbent.PaymentOrder) error {
	s.order = order
	return s.err
}

type paymentFulfillmentAffiliateRepoStub struct {
	*affiliateSignupBonusRepoStub
	accrueCalls []paymentFulfillmentAffiliateAccrueCall
	accrueErr   error
}

type paymentFulfillmentAffiliateAccrueCall struct {
	inviterID     int64
	inviteeUserID int64
	amount        float64
	sourceOrderID *int64
}

func newPaymentFulfillmentAffiliateRepoStub(inviteeUserID int64) *paymentFulfillmentAffiliateRepoStub {
	inviterID := int64(1)
	base := newAffiliateSignupBonusRepoStub()
	now := time.Now()
	base.profiles = map[int64]*AffiliateSummary{
		inviterID: {
			UserID:    inviterID,
			AffCode:   "INVITER",
			CreatedAt: now,
		},
		inviteeUserID: {
			UserID:    inviteeUserID,
			AffCode:   "INVITEE",
			InviterID: &inviterID,
			CreatedAt: now,
		},
	}
	return &paymentFulfillmentAffiliateRepoStub{affiliateSignupBonusRepoStub: base}
}

func (r *paymentFulfillmentAffiliateRepoStub) AccrueQuota(_ context.Context, inviterID, inviteeUserID int64, amount float64, _ int, sourceOrderID *int64) (bool, error) {
	if r.accrueErr != nil {
		return false, r.accrueErr
	}
	var copiedOrderID *int64
	if sourceOrderID != nil {
		v := *sourceOrderID
		copiedOrderID = &v
	}
	r.accrueCalls = append(r.accrueCalls, paymentFulfillmentAffiliateAccrueCall{
		inviterID:     inviterID,
		inviteeUserID: inviteeUserID,
		amount:        amount,
		sourceOrderID: copiedOrderID,
	})
	return true, nil
}

func (r *paymentFulfillmentAffiliateRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}

func ensurePaymentAuditOrderActionUniqueIndex(t *testing.T, ctx context.Context, client *dbent.Client) {
	t.Helper()
	_, err := client.ExecContext(ctx, `
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_audit_logs_order_action_uniq
ON payment_audit_logs(order_id, action)`)
	require.NoError(t, err)
}

func TestExecuteSubscriptionFulfillmentUsesCreditPurchaseSnapshot(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	fulfiller := &subscriptionCreditPurchaseFulfillerStub{}
	svc := &PaymentService{entClient: client, subscriptionCreditPurchaseSvc: fulfiller}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.NoError(t, err)
	require.NotNil(t, fulfiller.order)
	require.Equal(t, order.ID, fulfiller.order.ID)
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, got.Status)
	require.NotNil(t, got.CompletedAt)
}

func TestExecuteSubscriptionFulfillmentRequiresPurchaseService(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	svc := &PaymentService{entClient: client}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.Error(t, err)
	require.Equal(t, "SUBSCRIPTION_PURCHASE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))
	require.Contains(t, err.Error(), "subscription purchase service is not configured")
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusFulfillmentFailed, got.Status)
	require.NotNil(t, got.FailedReason)
	require.Contains(t, *got.FailedReason, "SUBSCRIPTION_PURCHASE_SERVICE_UNAVAILABLE")
}

func TestProvidePaymentServiceInjectsSubscriptionCreditPurchase(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	purchaseSvc := &SubscriptionCreditPurchaseService{}
	notificationEmail := &NotificationEmailService{}

	svc := ProvidePaymentService(
		client,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		purchaseSvc,
		notificationEmail,
		nil,
	)

	require.Same(t, purchaseSvc, svc.subscriptionCreditPurchaseSvc)
	require.Same(t, notificationEmail, svc.notificationEmailService)

	// Inject a stub fulfiller through the same field ProvidePaymentService sets,
	// so ExecuteSubscriptionFulfillment no longer hits the nil SERVICE_UNAVAILABLE path.
	fulfiller := &subscriptionCreditPurchaseFulfillerStub{}
	svc.subscriptionCreditPurchaseSvc = fulfiller

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, fulfiller.order)
	require.Equal(t, order.ID, fulfiller.order.ID)
}

func TestExecuteSubscriptionFulfillmentAccruesAffiliateRebateForExternalPayment(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	fulfiller := &subscriptionCreditPurchaseFulfillerStub{}
	affiliateRepo := newPaymentFulfillmentAffiliateRepoStub(order.UserID)
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:     "true",
		SettingKeyAffiliateRebateTiers: `[{"level":"L1","min_invitees":0,"min_recharge":0,"rebate_rate_percent":10}]`,
	}}, nil)
	svc := &PaymentService{
		entClient:                     client,
		subscriptionCreditPurchaseSvc: fulfiller,
		affiliateService:              NewAffiliateService(affiliateRepo, settingSvc, nil, nil),
	}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.NoError(t, err)
	require.Len(t, affiliateRepo.accrueCalls, 1)
	call := affiliateRepo.accrueCalls[0]
	require.Equal(t, int64(1), call.inviterID)
	require.Equal(t, order.UserID, call.inviteeUserID)
	require.InDelta(t, order.Amount*0.10, call.amount, 1e-9)
	require.NotNil(t, call.sourceOrderID)
	require.Equal(t, order.ID, *call.sourceOrderID)
}

func TestExecuteSubscriptionFulfillmentCompletesWhenAffiliateRebateFails(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	fulfiller := &subscriptionCreditPurchaseFulfillerStub{}
	affiliateRepo := newPaymentFulfillmentAffiliateRepoStub(order.UserID)
	affiliateRepo.accrueErr = errors.New("affiliate ledger unavailable")
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:     "true",
		SettingKeyAffiliateRebateTiers: `[{"level":"L1","min_invitees":0,"min_recharge":0,"rebate_rate_percent":10}]`,
	}}, nil)
	svc := &PaymentService{
		entClient:                     client,
		subscriptionCreditPurchaseSvc: fulfiller,
		affiliateService:              NewAffiliateService(affiliateRepo, settingSvc, nil, nil),
	}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.NoError(t, err)
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, got.Status)
	require.Nil(t, got.FailedReason)
	requirePaymentAuditAction(t, ctx, svc, order.ID, "AFFILIATE_REBATE_FAILED")
	requireNoPaymentAuditAction(t, ctx, svc, order.ID, "FULFILLMENT_FAILED")
}

func TestExecuteBalanceFulfillmentCompletesWhenAffiliateRebateFails(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createPaidBalanceOrderForFulfillmentTest(t, ctx, client)
	affiliateRepo := newPaymentFulfillmentAffiliateRepoStub(order.UserID)
	affiliateRepo.accrueErr = errors.New("affiliate ledger unavailable")
	settingSvc := NewSettingService(&affiliateSignupBonusSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled:     "true",
		SettingKeyAffiliateRebateTiers: `[{"level":"L1","min_invitees":0,"min_recharge":0,"rebate_rate_percent":10}]`,
	}}, nil)
	redeemRepo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{
		order.RechargeCode: {
			ID:     1001,
			Code:   order.RechargeCode,
			Type:   RedeemTypeBalance,
			Value:  order.Amount,
			Status: StatusUsed,
		},
	}}
	svc := &PaymentService{
		entClient:        client,
		redeemService:    NewRedeemService(redeemRepo, nil, nil, nil, nil, client, nil, nil),
		affiliateService: NewAffiliateService(affiliateRepo, settingSvc, nil, nil),
	}

	err := svc.ExecuteBalanceFulfillment(ctx, order.ID)

	require.NoError(t, err)
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, got.Status)
	require.Nil(t, got.FailedReason)
	requirePaymentAuditAction(t, ctx, svc, order.ID, "AFFILIATE_REBATE_FAILED")
	requireNoPaymentAuditAction(t, ctx, svc, order.ID, "FULFILLMENT_FAILED")
}

func TestExecuteSubscriptionFulfillmentMarksCreditPurchaseBlockAsFulfillmentFailed(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)
	fulfiller := &subscriptionCreditPurchaseFulfillerStub{err: ErrAlreadyHasUsableSubscription}
	svc := &PaymentService{entClient: client, subscriptionCreditPurchaseSvc: fulfiller}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.ErrorIs(t, err, ErrAlreadyHasUsableSubscription)
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusFulfillmentFailed, got.Status)
	require.NotNil(t, got.FailedReason)
	require.Contains(t, *got.FailedReason, "ALREADY_HAS_USABLE_SUBSCRIPTION")
}

func TestExecuteSubscriptionFulfillmentRecoversCreditPurchaseFromLedger(t *testing.T) {
	ctx := context.Background()
	client := newOrderNotFoundTestClient(t)
	order := createPaidCreditSubscriptionOrderForFulfillmentTest(t, ctx, client)

	sub := client.UserSubscription.Create().
		SetUserID(order.UserID).
		SetPlanID(*order.PlanID).
		SetScopeType(*order.SubscriptionScopeType).
		SetScopeConfig(order.SubscriptionScopeConfig).
		SetQuotaLimitUsd(*order.SubscriptionQuotaUsd).
		SetQuotaUsedUsd(0).
		SetNillableDailyLimitUsd(order.SubscriptionDailyLimitUsd).
		SetNillableWeeklyLimitUsd(order.SubscriptionWeeklyLimitUsd).
		SetStartsAt(time.Now().UTC()).
		SetExpiresAt(time.Now().UTC().AddDate(0, 0, *order.SubscriptionValidityDays)).
		SetStatus(SubscriptionStatusActive).
		SetAssignedAt(time.Now().UTC()).
		SetNotes("payment order 1").
		SaveX(ctx)
	client.SubscriptionCreditLedger.Create().
		SetUserID(order.UserID).
		SetSubscriptionID(sub.ID).
		SetOrderID(order.ID).
		SetType(SubscriptionCreditLedgerPurchase).
		SetDeltaUsd(*order.SubscriptionQuotaUsd).
		SetBalanceDeltaUsd(0).
		SetRemainingAfterUsd(*order.SubscriptionQuotaUsd).
		SetReason("payment order 1").
		ExecX(ctx)

	fulfiller := &subscriptionCreditPurchaseFulfillerStub{err: ErrAlreadyHasUsableSubscription}
	svc := &PaymentService{entClient: client, subscriptionCreditPurchaseSvc: fulfiller}

	err := svc.ExecuteSubscriptionFulfillment(ctx, order.ID)

	require.NoError(t, err)
	require.Nil(t, fulfiller.order, "purchase ledger anchor should skip the second fulfillment attempt")
	got, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, got.Status)
	require.NotNil(t, got.CompletedAt)
}

func requirePaymentAuditAction(t *testing.T, ctx context.Context, svc *PaymentService, orderID int64, action string) {
	t.Helper()
	logs, err := svc.GetOrderAuditLogs(ctx, orderID)
	require.NoError(t, err)
	for _, log := range logs {
		if log.Action == action {
			return
		}
	}
	require.Failf(t, "missing audit action", "order %d missing audit action %s", orderID, action)
}

func requireNoPaymentAuditAction(t *testing.T, ctx context.Context, svc *PaymentService, orderID int64, action string) {
	t.Helper()
	logs, err := svc.GetOrderAuditLogs(ctx, orderID)
	require.NoError(t, err)
	for _, log := range logs {
		require.NotEqual(t, action, log.Action)
	}
}

func createPaidBalanceOrderForFulfillmentTest(t *testing.T, ctx context.Context, client *dbent.Client) *dbent.PaymentOrder {
	t.Helper()
	user := client.User.Create().
		SetEmail("balance-buyer@example.com").
		SetPasswordHash("hash").
		SetUsername("balance-buyer").
		SaveX(ctx)
	return client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetRechargeCode("BALANCE-ORDER").
		SetOutTradeNo("sub2_balance_order").
		SetPaymentType(payment.TypeWxpay).
		SetPaymentTradeNo("trade-balance").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test").
		SaveX(ctx)
}

func createPaidCreditSubscriptionOrderForFulfillmentTest(t *testing.T, ctx context.Context, client *dbent.Client) *dbent.PaymentOrder {
	t.Helper()
	user := client.User.Create().
		SetEmail("credit-buyer@example.com").
		SetPasswordHash("hash").
		SetUsername("credit-buyer").
		SaveX(ctx)
	quota := 25.0
	daily := 3.5
	weekly := 10.0
	scopeType := SubscriptionScopeAllAvailableGroups
	validityDays := 30
	return client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetRechargeCode("SUB-CREDIT-ORDER").
		SetOutTradeNo("sub2_credit_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-credit").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test").
		SetPlanID(77).
		SetSubscriptionQuotaUsd(quota).
		SetSubscriptionDailyLimitUsd(daily).
		SetSubscriptionWeeklyLimitUsd(weekly).
		SetSubscriptionScopeType(scopeType).
		SetSubscriptionScopeConfig(map[string]any{}).
		SetSubscriptionValidityDays(validityDays).
		SaveX(ctx)
}
