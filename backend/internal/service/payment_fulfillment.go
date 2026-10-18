package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/subscriptioncreditledger"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
)

// ErrOrderNotFound is returned by HandlePaymentNotification when the webhook
// references an out_trade_no that does not exist in our DB. Callers (webhook
// handlers) should treat this as a terminal, non-retryable condition and still
// respond with a 2xx success to the provider — otherwise the provider will keep
// retrying forever (e.g. when a foreign environment's webhook endpoint is
// misconfigured to point at us, or when our orders table has been wiped).
var ErrOrderNotFound = errors.New("payment order not found")

const paymentFulfillmentLeaseDuration = 5 * time.Minute

type paymentFulfillmentLease struct {
	version time.Time
}

// --- Payment Notification & Fulfillment ---

func (s *PaymentService) HandlePaymentNotification(ctx context.Context, n *payment.PaymentNotification, pk string) error {
	if n.Status != payment.NotificationStatusSuccess {
		return nil
	}
	// Look up order by out_trade_no (the external order ID we sent to the provider)
	order, err := s.entClient.PaymentOrder.Query().Where(paymentorder.OutTradeNo(n.OrderID)).Only(ctx)
	if err != nil {
		// Fallback only for true legacy "sub2_N" DB-ID payloads when the
		// current out_trade_no lookup genuinely did not find an order.
		if oid, ok := parseLegacyPaymentOrderID(n.OrderID, err); ok {
			return s.confirmPayment(ctx, oid, n.TradeNo, n.Amount, pk, n.Metadata)
		}
		if dbent.IsNotFound(err) {
			return fmt.Errorf("%w: out_trade_no=%s", ErrOrderNotFound, n.OrderID)
		}
		return fmt.Errorf("lookup order failed for out_trade_no %s: %w", n.OrderID, err)
	}
	return s.confirmPayment(ctx, order.ID, n.TradeNo, n.Amount, pk, n.Metadata)
}

// ConfirmNotificationUpstream asks providers that implement
// payment.NotificationConfirmer to confirm a verified success notification
// against the upstream order, using the same instance that verified it.
// Failures are recorded on the order's audit trail and returned so the webhook
// answers non-success and the gateway retries.
func (s *PaymentService) ConfirmNotificationUpstream(ctx context.Context, prov payment.Provider, n *payment.PaymentNotification) error {
	confirmer, ok := prov.(payment.NotificationConfirmer)
	if !ok || n == nil || n.Status != payment.NotificationStatusSuccess {
		return nil
	}
	notifiedTradeNo := n.TradeNo
	finishProviderCall := servertiming.ObserveDependency(ctx, "payment")
	err := confirmer.ConfirmNotification(ctx, n)
	finishProviderCall()
	if err == nil {
		return nil
	}
	if order, lookupErr := s.entClient.PaymentOrder.Query().Where(paymentorder.OutTradeNo(n.OrderID)).Only(ctx); lookupErr == nil {
		s.writeAuditLog(ctx, order.ID, "PAYMENT_UPSTREAM_UNCONFIRMED", prov.ProviderKey(), map[string]any{
			"detail":  err.Error(),
			"tradeNo": notifiedTradeNo,
			"amount":  n.Amount,
		})
	}
	return err
}

func parseLegacyPaymentOrderID(orderID string, lookupErr error) (int64, bool) {
	if !dbent.IsNotFound(lookupErr) {
		return 0, false
	}
	orderID = strings.TrimSpace(orderID)
	if !strings.HasPrefix(orderID, orderIDPrefix) {
		return 0, false
	}
	trimmed := strings.TrimPrefix(orderID, orderIDPrefix)
	if trimmed == "" || trimmed == orderID {
		return 0, false
	}
	oid, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || oid <= 0 {
		return 0, false
	}
	return oid, true
}

func (s *PaymentService) confirmPayment(ctx context.Context, oid int64, tradeNo string, paid float64, pk string, metadata map[string]string) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		slog.Error("order not found", "orderID", oid)
		return nil
	}
	instanceProviderKey := ""
	if inst, instErr := s.getOrderProviderInstance(ctx, o); instErr == nil && inst != nil {
		instanceProviderKey = inst.ProviderKey
	}
	expectedProviderKey := expectedNotificationProviderKeyForOrder(s.registry, o, instanceProviderKey)
	if expectedProviderKey != "" && strings.TrimSpace(pk) != "" && !strings.EqualFold(expectedProviderKey, strings.TrimSpace(pk)) {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_PROVIDER_MISMATCH", pk, map[string]any{
			"expectedProvider": expectedProviderKey,
			"actualProvider":   pk,
			"tradeNo":          tradeNo,
		})
		return fmt.Errorf("provider mismatch: expected %s, got %s", expectedProviderKey, pk)
	}
	if err := validateProviderNotificationMetadata(o, pk, metadata); err != nil {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_PROVIDER_METADATA_MISMATCH", pk, map[string]any{
			"detail":  err.Error(),
			"tradeNo": tradeNo,
		})
		return err
	}
	if !isValidProviderAmount(paid) {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_INVALID_AMOUNT", pk, map[string]any{
			"expected": o.PayAmount,
			"paid":     paid,
			"tradeNo":  tradeNo,
		})
		return fmt.Errorf("invalid paid amount from provider: %v", paid)
	}
	confirmedPayAmount, ok := resolveNotificationPaymentAmount(o, pk, paid)
	if !ok {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_AMOUNT_MISMATCH", pk, map[string]any{
			"expected":       o.PayAmount,
			"creditedAmount": o.Amount,
			"paid":           paid,
			"tradeNo":        tradeNo,
		})
		return fmt.Errorf("amount mismatch: expected %.2f, got %.2f", o.PayAmount, paid)
	}
	if isEasyPayFamilyProviderKey(pk) || isEasyPayFamilyProviderKey(expectedProviderKey) {
		// Completed/refund-state callbacks are acknowledgements only. They must
		// neither re-credit the user nor depend on a still-available provider.
		if o.Status == OrderStatusCompleted || psIsRefundStatus(o.Status) {
			return nil
		}
		if err := s.verifyEasyPaySettlement(ctx, o, tradeNo, paid); err != nil {
			s.writeAuditLog(ctx, o.ID, "PAYMENT_UPSTREAM_VERIFY_FAILED", pk, map[string]any{
				"reason": err.Error(),
			})
			return err
		}
	}
	return s.toPaid(ctx, o, tradeNo, confirmedPayAmount, pk)
}

func paymentAmountToleranceForCurrency(currency string) float64 {
	minorUnit := payment.CurrencyMinorUnit(currency)
	if minorUnit <= 2 {
		return amountToleranceCNY
	}
	return math.Pow10(-minorUnit) / 2
}

func isValidProviderAmount(amount float64) bool {
	return amount > 0 && !math.IsNaN(amount) && !math.IsInf(amount, 0)
}

func resolveNotificationPaymentAmount(order *dbent.PaymentOrder, providerKey string, paid float64) (float64, bool) {
	if order == nil {
		return 0, false
	}
	if math.Abs(paid-order.PayAmount) <= amountToleranceCNY {
		return paid, true
	}
	if strings.EqualFold(strings.TrimSpace(providerKey), payment.TypeMapay) &&
		math.Abs(paid-order.Amount) <= amountToleranceCNY {
		return order.PayAmount, true
	}
	return 0, false
}

func validateProviderNotificationMetadata(order *dbent.PaymentOrder, providerKey string, metadata map[string]string) error {
	return validateProviderSnapshotMetadata(order, providerKey, metadata)
}

func expectedNotificationProviderKey(registry *payment.Registry, orderPaymentType string, orderProviderKey string, instanceProviderKey string) string {
	if key := strings.TrimSpace(instanceProviderKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(orderProviderKey); key != "" {
		return key
	}
	if registry != nil {
		if key := strings.TrimSpace(registry.GetProviderKey(payment.PaymentType(orderPaymentType))); key != "" {
			return key
		}
	}
	return strings.TrimSpace(orderPaymentType)
}

func (s *PaymentService) toPaid(ctx context.Context, o *dbent.PaymentOrder, tradeNo string, paid float64, pk string) error {
	previousStatus := o.Status
	now := time.Now()
	grace := now.Add(-paymentGraceMinutes * time.Minute)
	c, err := s.entClient.PaymentOrder.Update().Where(
		paymentorder.IDEQ(o.ID),
		paymentorder.Or(
			paymentorder.StatusEQ(OrderStatusPending),
			paymentorder.StatusEQ(OrderStatusCancelled),
			paymentorder.And(
				paymentorder.StatusEQ(OrderStatusExpired),
				paymentorder.UpdatedAtGTE(grace),
			),
		),
	).SetStatus(OrderStatusPaid).SetPayAmount(paid).SetPaymentTradeNo(tradeNo).SetPaidAt(now).ClearFailedAt().ClearFailedReason().Save(ctx)
	if err != nil {
		return fmt.Errorf("update to PAID: %w", err)
	}
	if c == 0 {
		return s.alreadyProcessed(ctx, o)
	}
	if previousStatus == OrderStatusCancelled || previousStatus == OrderStatusExpired {
		slog.Info("order recovered from webhook payment success",
			"orderID", o.ID,
			"previousStatus", previousStatus,
			"tradeNo", tradeNo,
			"provider", pk,
		)
		s.writeAuditLog(ctx, o.ID, "ORDER_RECOVERED", pk, map[string]any{
			"previous_status": previousStatus,
			"tradeNo":         tradeNo,
			"paidAmount":      paid,
			"reason":          "webhook payment success received after order " + previousStatus,
		})
	}
	s.writeAuditLog(ctx, o.ID, "ORDER_PAID", pk, map[string]any{"tradeNo": tradeNo, "paidAmount": paid})
	if err := s.executeFulfillment(ctx, o.ID); err != nil {
		slog.Error("payment fulfillment failed after payment confirmation",
			"orderID", o.ID,
			"provider", pk,
			"tradeNo", tradeNo,
			"error", err,
		)
		s.scheduleFulfillmentRetries(o.ID)
		return nil
	}
	return nil
}

func (s *PaymentService) alreadyProcessed(ctx context.Context, o *dbent.PaymentOrder) error {
	cur, err := s.entClient.PaymentOrder.Get(ctx, o.ID)
	if err != nil {
		return nil
	}
	switch cur.Status {
	case OrderStatusCompleted, OrderStatusRefunded:
		return nil
	case OrderStatusFailed, OrderStatusFulfillmentFailed:
		if err := s.executeFulfillment(ctx, o.ID); err != nil {
			s.scheduleFulfillmentRetries(o.ID)
			return nil
		}
		return nil
	case OrderStatusPaid, OrderStatusRecharging:
		return fmt.Errorf("order %d is being processed", o.ID)
	case OrderStatusExpired:
		slog.Warn("webhook payment success for expired order beyond grace period",
			"orderID", o.ID,
			"status", cur.Status,
			"updatedAt", cur.UpdatedAt,
		)
		s.writeAuditLog(ctx, o.ID, "PAYMENT_AFTER_EXPIRY", "system", map[string]any{
			"status":    cur.Status,
			"updatedAt": cur.UpdatedAt,
			"reason":    "payment arrived after expiry grace period",
		})
		return nil
	default:
		return nil
	}
}

func (s *PaymentService) executeFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	switch o.OrderType {
	case payment.OrderTypeSubscription:
		return s.ExecuteSubscriptionFulfillment(ctx, oid)
	case payment.OrderTypeBalance:
		return s.ExecuteBalanceFulfillment(ctx, oid)
	default:
		// 不支持的订单类型不履约：标记 FAILED 后正常返回，避免支付回调持续 500 重试。
		s.rejectUnsupportedOrderType(ctx, o)
		return nil
	}
}

func isSupportedPaymentOrderType(orderType string) bool {
	return orderType == payment.OrderTypeBalance || orderType == payment.OrderTypeSubscription
}

func (s *PaymentService) rejectUnsupportedOrderType(ctx context.Context, o *dbent.PaymentOrder) {
	reason := "unsupported order type: " + o.OrderType
	c, err := s.entClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(o.ID),
			paymentorder.StatusIn(OrderStatusPaid, OrderStatusRecharging),
		).
		SetStatus(OrderStatusFailed).SetFailedAt(time.Now()).SetFailedReason(reason).Save(ctx)
	if err != nil {
		slog.Error("mark unsupported order type FAILED", "orderID", o.ID, "error", err)
		return
	}
	if c > 0 {
		slog.Warn("payment order with unsupported order type not fulfilled", "orderID", o.ID, "orderType", o.OrderType)
		s.writeAuditLog(ctx, o.ID, "FULFILLMENT_FAILED", "system", map[string]any{"reason": reason})
	}
}

func (s *PaymentService) ExecuteBalanceFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.OrderType != payment.OrderTypeBalance {
		return infraerrors.BadRequest("INVALID_ORDER_TYPE", "order is not a balance order")
	}
	if o.Status == OrderStatusCompleted {
		return nil
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusFailed && o.Status != OrderStatusFulfillmentFailed && o.Status != OrderStatusRecharging {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+o.Status)
	}
	lease, err := s.acquirePaymentFulfillmentLease(ctx, o)
	if err != nil {
		return err
	}
	if lease == nil {
		return nil
	}
	if err := s.doBalance(ctx, o, lease); err != nil {
		s.markFailed(ctx, oid, lease, err)
		return err
	}
	return nil
}

func (s *PaymentService) acquirePaymentFulfillmentLease(ctx context.Context, o *dbent.PaymentOrder) (*paymentFulfillmentLease, error) {
	if o == nil {
		return nil, infraerrors.BadRequest("INVALID_STATUS", "nil payment order")
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	staleBefore := now.Add(-paymentFulfillmentLeaseDuration)
	updated, err := s.entClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(o.ID),
			paymentorder.Or(
				paymentorder.StatusIn(OrderStatusPaid, OrderStatusFailed, OrderStatusFulfillmentFailed),
				paymentorder.And(
					paymentorder.StatusEQ(OrderStatusRecharging),
					paymentorder.UpdatedAtLTE(staleBefore),
				),
			),
		).
		SetStatus(OrderStatusRecharging).
		SetUpdatedAt(now).
		ClearFailedAt().
		ClearFailedReason().
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire fulfillment lease: %w", err)
	}
	if updated == 0 {
		current, getErr := s.entClient.PaymentOrder.Get(ctx, o.ID)
		if getErr != nil {
			return nil, fmt.Errorf("reload fulfillment lease: %w", getErr)
		}
		if current.Status == OrderStatusCompleted {
			return nil, nil
		}
		if current.Status == OrderStatusRecharging {
			return nil, infraerrors.Conflict("CONFLICT", "order is being processed")
		}
		return nil, infraerrors.Conflict("CONFLICT", "order status changed while acquiring fulfillment lease")
	}

	// Reload the persisted timestamp instead of trusting application clock precision.
	claimed, err := s.entClient.PaymentOrder.Get(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("reload acquired fulfillment lease: %w", err)
	}
	if claimed.Status != OrderStatusRecharging {
		return nil, infraerrors.Conflict("CONFLICT", "fulfillment lease was lost")
	}
	return &paymentFulfillmentLease{version: claimed.UpdatedAt}, nil
}

// redeemAction represents the idempotency decision for balance fulfillment.
type redeemAction int

const (
	// redeemActionCreate: code does not exist — create it, then redeem.
	redeemActionCreate redeemAction = iota
	// redeemActionRedeem: code exists but is unused — skip creation, redeem only.
	redeemActionRedeem
	// redeemActionSkipCompleted: code exists and is already used — skip to mark completed.
	redeemActionSkipCompleted
)

// resolveRedeemAction decides the idempotency action based on an existing redeem code lookup.
// existing is the result of GetByCode; lookupErr is the error from that call.
func resolveRedeemAction(existing *RedeemCode, lookupErr error) (redeemAction, error) {
	if lookupErr != nil {
		if errors.Is(lookupErr, ErrRedeemCodeNotFound) {
			return redeemActionCreate, nil
		}
		return redeemActionCreate, fmt.Errorf("lookup payment redeem code: %w", lookupErr)
	}
	if existing == nil {
		return redeemActionCreate, nil
	}
	if existing.IsUsed() {
		return redeemActionSkipCompleted, nil
	}
	return redeemActionRedeem, nil
}

func validatePaymentRedeemCode(o *dbent.PaymentOrder, code *RedeemCode) error {
	if o == nil || code == nil {
		return errors.New("payment redeem code validation requires an order and code")
	}
	if code.Code != o.RechargeCode {
		return fmt.Errorf("payment redeem code mismatch for order %d", o.ID)
	}
	if code.Type != RedeemTypeBalance {
		return fmt.Errorf("payment redeem code type mismatch for order %d: got %s", o.ID, code.Type)
	}
	if math.IsNaN(code.Value) || math.IsInf(code.Value, 0) || math.Abs(code.Value-o.Amount) > 1e-8 {
		return fmt.Errorf("payment redeem code amount mismatch for order %d: expected %.8f, got %.8f", o.ID, o.Amount, code.Value)
	}
	switch code.Status {
	case StatusUnused:
		if code.UsedBy != nil {
			return fmt.Errorf("unused payment redeem code has a user for order %d", o.ID)
		}
	case StatusUsed:
		if code.UsedBy == nil || *code.UsedBy != o.UserID {
			return fmt.Errorf("payment redeem code user mismatch for order %d", o.ID)
		}
	default:
		return fmt.Errorf("payment redeem code has invalid status for order %d: %s", o.ID, code.Status)
	}
	return nil
}

func (s *PaymentService) doBalance(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	// Idempotency: check if redeem code already exists (from a previous partial run)
	existing, lookupErr := s.redeemService.GetByCode(ctx, o.RechargeCode)
	action, err := resolveRedeemAction(existing, lookupErr)
	if err != nil {
		return err
	}
	if existing != nil {
		if err := validatePaymentRedeemCode(o, existing); err != nil {
			return err
		}
	}

	if action == redeemActionCreate {
		rc := &RedeemCode{Code: o.RechargeCode, Type: RedeemTypeBalance, Value: o.Amount, Status: StatusUnused}
		if err := s.redeemService.CreateCode(ctx, rc); err != nil {
			return fmt.Errorf("create redeem code: %w", err)
		}
	}
	// redeemActionRedeem: code exists but unused — skip creation, proceed to redeem
	if action != redeemActionSkipCompleted {
		if _, err := s.redeemService.redeemForPaymentFulfillment(ctx, o.UserID, o.RechargeCode); err != nil {
			return fmt.Errorf("redeem balance: %w", err)
		}
	}
	// 赠送发放放在两个分支的汇合处：主码已入账（SkipCompleted）但赠送上次失败的重试
	// 也能在此补发，且由 "-B" 锚定码保证不重复。
	if o.BonusAmount > 0 {
		if err := s.fulfillRechargeBonus(ctx, o); err != nil {
			// 失败走 markFailed → 既有重试链路
			return fmt.Errorf("recharge bonus: %w", err)
		}
	}
	s.applyAffiliateRebateBestEffort(ctx, o)
	return s.markCompleted(ctx, o, lease, "RECHARGE_SUCCESS")
}

// fulfillRechargeBonus 发放充值满赠赠送余额。
// 与主码同构：幂等锚定为 "{recharge_code}-B"（远小于 recharge_code 的 64 上限），
// GetByCode 不存在则创建 type=recharge_bonus 的兑换码后走可信兑换入账。
func (s *PaymentService) fulfillRechargeBonus(ctx context.Context, o *dbent.PaymentOrder) error {
	bonusCode := o.RechargeCode + "-B"
	existing, lookupErr := s.redeemService.GetByCode(ctx, bonusCode)
	action, err := resolveRedeemAction(existing, lookupErr)
	if err != nil {
		return err
	}
	switch action {
	case redeemActionSkipCompleted:
		// 已发放
		return nil
	case redeemActionCreate:
		rc := &RedeemCode{
			Code:   bonusCode,
			Type:   RedeemTypeRechargeBonus,
			Value:  o.BonusAmount,
			Status: StatusUnused,
			Notes:  "payment_bonus:" + o.RechargeCode,
		}
		if err := s.redeemService.CreateCode(ctx, rc); err != nil {
			return fmt.Errorf("create bonus redeem code: %w", err)
		}
	case redeemActionRedeem:
		// 码已创建但未使用，直接兑换
	}
	if _, err := s.redeemService.redeemForPaymentFulfillment(ctx, o.UserID, bonusCode); err != nil {
		return fmt.Errorf("redeem bonus: %w", err)
	}
	return nil
}

func (s *PaymentService) markCompleted(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease, auditAction string) error {
	if lease == nil {
		return errors.New("missing payment fulfillment lease")
	}
	now := time.Now()
	updated, err := s.entClient.PaymentOrder.Update().Where(
		paymentorder.IDEQ(o.ID),
		paymentorder.StatusEQ(OrderStatusRecharging),
		paymentorder.UpdatedAtEQ(lease.version),
	).SetStatus(OrderStatusCompleted).SetCompletedAt(now).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	if updated == 0 {
		current, getErr := s.entClient.PaymentOrder.Get(ctx, o.ID)
		if getErr == nil && current.Status == OrderStatusCompleted {
			return nil
		}
		return infraerrors.Conflict("CONFLICT", "fulfillment lease was lost before completion")
	}
	if !s.hasAuditLog(ctx, o.ID, auditAction) {
		s.writeAuditLog(ctx, o.ID, auditAction, "system", map[string]any{
			"rechargeCode":   o.RechargeCode,
			"creditedAmount": o.Amount,
			"payAmount":      o.PayAmount,
			"bonusAmount":    o.BonusAmount,
		})
	}
	return nil
}

func (s *PaymentService) dispatchPaymentFulfillmentNotification(o *dbent.PaymentOrder, auditAction string) {
	if s == nil || s.notificationEmailService == nil || o == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
		defer cancel()
		var err error
		switch auditAction {
		case "RECHARGE_SUCCESS":
			err = s.sendBalanceRechargeSuccessNotification(ctx, o)
		case "SUBSCRIPTION_SUCCESS":
			err = s.sendSubscriptionPurchaseSuccessNotification(ctx, o)
		default:
			return
		}
		if err != nil {
			slog.Warn("payment fulfillment notification email failed", "order_id", o.ID, "action", auditAction, "err", err.Error())
		}
	}()
}

func (s *PaymentService) sendBalanceRechargeSuccessNotification(ctx context.Context, o *dbent.PaymentOrder) error {
	currentBalance := ""
	if s.userRepo != nil {
		if user, err := s.userRepo.GetByID(ctx, o.UserID); err == nil && user != nil {
			currentBalance = fmt.Sprintf("%.2f", user.Balance)
		}
	}
	return s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
		Event:          NotificationEmailEventBalanceRechargeSuccess,
		RecipientEmail: o.UserEmail,
		RecipientName:  firstNonEmpty(o.UserName, o.UserEmail),
		UserID:         o.UserID,
		SourceType:     "payment_order",
		SourceID:       strconv.FormatInt(o.ID, 10),
		Variables: map[string]string{
			"recharge_amount": fmt.Sprintf("%.2f", o.Amount),
			"current_balance": currentBalance,
			"order_id":        strconv.FormatInt(o.ID, 10),
		},
	})
}

func (s *PaymentService) sendSubscriptionPurchaseSuccessNotification(ctx context.Context, o *dbent.PaymentOrder) error {
	variables := map[string]string{
		"subscription_group": "Subscription",
		"subscription_days":  "",
		"expiry_time":        "",
		"order_id":           strconv.FormatInt(o.ID, 10),
	}
	if o.SubscriptionDays != nil {
		variables["subscription_days"] = strconv.Itoa(*o.SubscriptionDays)
	}
	if o.SubscriptionGroupID != nil {
		if s.groupRepo != nil {
			if group, err := s.groupRepo.GetByID(ctx, *o.SubscriptionGroupID); err == nil && group != nil && strings.TrimSpace(group.Name) != "" {
				variables["subscription_group"] = group.Name
			}
		}
		if s.subscriptionSvc != nil {
			if sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, o.UserID, *o.SubscriptionGroupID); err == nil && sub != nil {
				variables["expiry_time"] = sub.ExpiresAt.Format("2006-01-02 15:04")
			}
		}
	}
	return s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
		Event:          NotificationEmailEventSubscriptionPurchaseSuccess,
		RecipientEmail: o.UserEmail,
		RecipientName:  firstNonEmpty(o.UserName, o.UserEmail),
		UserID:         o.UserID,
		SourceType:     "payment_order",
		SourceID:       strconv.FormatInt(o.ID, 10),
		Variables:      variables,
	})
}

func (s *PaymentService) ExecuteSubscriptionFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.Status == OrderStatusCompleted {
		return nil
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusFailed && o.Status != OrderStatusFulfillmentFailed && o.Status != OrderStatusRecharging {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+o.Status)
	}
	isCreditOrder := paymentOrderHasCreditSubscriptionSnapshot(o)
	if !isCreditOrder && (o.SubscriptionGroupID == nil || o.SubscriptionDays == nil) {
		return infraerrors.BadRequest("INVALID_STATUS", "missing subscription info")
	}
	lease, err := s.acquirePaymentFulfillmentLease(ctx, o)
	if err != nil {
		return err
	}
	if lease == nil {
		return nil
	}
	var fulfillErr error
	if isCreditOrder {
		fulfillErr = s.doCreditSub(ctx, o, lease)
	} else {
		fulfillErr = s.doSub(ctx, o, lease)
	}
	if fulfillErr != nil {
		s.markFailed(ctx, oid, lease, fulfillErr)
		return fulfillErr
	}
	return nil
}

func paymentOrderHasCreditSubscriptionSnapshot(o *dbent.PaymentOrder) bool {
	return o != nil &&
		o.SubscriptionQuotaUsd != nil &&
		o.SubscriptionScopeType != nil &&
		o.SubscriptionValidityDays != nil
}

func (s *PaymentService) findCreditSubscriptionPurchase(ctx context.Context, orderID int64) (*dbent.SubscriptionCreditLedger, error) {
	if s == nil || s.entClient == nil || orderID <= 0 {
		return nil, ErrSubscriptionNotFound
	}
	entry, err := s.entClient.SubscriptionCreditLedger.Query().
		Where(
			subscriptioncreditledger.OrderIDEQ(orderID),
			subscriptioncreditledger.TypeEQ(SubscriptionCreditLedgerPurchase),
		).
		Order(
			dbent.Desc(subscriptioncreditledger.FieldCreatedAt),
			dbent.Desc(subscriptioncreditledger.FieldID),
		).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return entry, nil
}

func (s *PaymentService) findCreditSubscriptionByOrderID(ctx context.Context, orderID int64) (*UserSubscription, error) {
	if s == nil || s.subscriptionSvc == nil {
		return nil, ErrSubscriptionNotFound
	}
	entry, err := s.findCreditSubscriptionPurchase(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return s.subscriptionSvc.GetByID(ctx, entry.SubscriptionID)
}

func (s *PaymentService) doCreditSub(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	if s.hasAuditLog(ctx, o.ID, "SUBSCRIPTION_SUCCESS") {
		slog.Info("subscription credit already fulfilled for order, skipping", "orderID", o.ID)
		if s.subscriptionSvc != nil {
			s.subscriptionSvc.InvalidateSubCache(o.UserID, 0)
		}
		return s.completeSubscriptionOrder(ctx, o, lease)
	}
	if entry, err := s.findCreditSubscriptionPurchase(ctx, o.ID); err == nil {
		slog.Info("subscription credit fulfillment already anchored by purchase ledger, recovering order status",
			"orderID", o.ID,
			"subscriptionID", entry.SubscriptionID,
		)
		if s.subscriptionSvc != nil {
			s.subscriptionSvc.InvalidateSubCache(o.UserID, 0)
		}
		return s.completeSubscriptionOrder(ctx, o, lease)
	} else if !errors.Is(err, ErrSubscriptionNotFound) {
		return fmt.Errorf("lookup subscription credit purchase: %w", err)
	}
	if s.subscriptionCreditPurchaseSvc == nil {
		return infraerrors.InternalServer("SUBSCRIPTION_PURCHASE_SERVICE_UNAVAILABLE", "subscription purchase service is not configured")
	}
	if err := s.subscriptionCreditPurchaseSvc.FulfillOrder(ctx, o); err != nil {
		if isAlreadyHasUsableSubscription(err) {
			slog.Warn("subscription credit purchase blocked by usable subscription", "orderID", o.ID, "userID", o.UserID, "error", err)
		}
		return fmt.Errorf("fulfill subscription credit purchase: %w", err)
	}
	if s.subscriptionSvc != nil {
		s.subscriptionSvc.InvalidateSubCache(o.UserID, 0)
	}
	return s.completeSubscriptionOrder(ctx, o, lease)
}

func (s *PaymentService) doSub(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	gid := *o.SubscriptionGroupID
	days := *o.SubscriptionDays
	g, err := s.groupRepo.GetByID(ctx, gid)
	if err != nil || g.Status != payment.EntityStatusActive {
		return fmt.Errorf("group %d no longer exists or inactive", gid)
	}
	// Idempotency: check audit log to see if subscription was already assigned.
	// Prevents double-extension on retry after markCompleted fails.
	if s.hasAuditLog(ctx, o.ID, "SUBSCRIPTION_SUCCESS") {
		slog.Info("subscription already assigned for order, skipping", "orderID", o.ID, "groupID", gid)
		return s.markCompleted(ctx, o, lease, "SUBSCRIPTION_SUCCESS")
	}
	orderNote := fmt.Sprintf("payment order %d", o.ID)
	_, _, err = s.subscriptionSvc.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{UserID: o.UserID, GroupID: gid, ValidityDays: days, AssignedBy: 0, Notes: orderNote})
	if err != nil {
		return fmt.Errorf("assign subscription: %w", err)
	}
	return s.completeSubscriptionOrder(ctx, o, lease)
}

func (s *PaymentService) completeSubscriptionOrder(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	s.applyAffiliateRebateBestEffort(ctx, o)
	return s.markCompleted(ctx, o, lease, "SUBSCRIPTION_SUCCESS")
}

func (s *PaymentService) applyAffiliateRebateBestEffort(ctx context.Context, o *dbent.PaymentOrder) {
	if err := s.applyAffiliateRebateForOrder(ctx, o); err != nil {
		orderID := int64(0)
		if o != nil {
			orderID = o.ID
		}
		slog.Error("affiliate rebate failed after payment fulfillment; continuing order completion",
			"orderID", orderID,
			"error", err,
		)
	}
}

func isAffiliateRebateEligibleOrder(o *dbent.PaymentOrder) bool {
	if o == nil {
		return false
	}
	switch o.OrderType {
	case payment.OrderTypeBalance:
		return true
	case payment.OrderTypeSubscription:
		return !isInternalBalancePaymentType(o.PaymentType)
	default:
		return false
	}
}

func (s *PaymentService) hasAuditLog(ctx context.Context, orderID int64, action string) bool {
	oid := strconv.FormatInt(orderID, 10)
	c, _ := s.entClient.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(oid), paymentauditlog.ActionEQ(action)).
		Limit(1).Count(ctx)
	return c > 0
}

func (s *PaymentService) applyAffiliateRebateForOrder(ctx context.Context, o *dbent.PaymentOrder) error {
	baseAmount := affiliateRebateBaseAmount(o)
	if o == nil || !isAffiliateRebateEligibleOrder(o) || baseAmount <= 0 {
		return nil
	}
	if s.affiliateService == nil {
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": fmt.Sprintf("begin affiliate rebate tx: %v", err),
		})
		return fmt.Errorf("begin affiliate rebate tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	claimed, err := s.tryClaimAffiliateRebateAudit(txCtx, tx.Client(), o.ID, baseAmount)
	if err != nil {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("claim affiliate rebate audit: %w", err)
	}
	if !claimed {
		return nil
	}

	sourceOrderID := o.ID
	rebateAmount, err := s.affiliateService.AccrueInviteRebateForOrder(txCtx, o.UserID, o.Amount, &sourceOrderID)
	if err != nil {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("accrue affiliate rebate: %w", err)
	}

	if rebateAmount <= 0 {
		if err := s.updateClaimedAffiliateRebateAudit(txCtx, tx.Client(), o.ID, "AFFILIATE_REBATE_SKIPPED", map[string]any{
			"baseAmount": baseAmount,
			"reason":     "no inviter bound or rebate amount <= 0",
		}); err != nil {
			_ = tx.Rollback()
			s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
				"error": err.Error(),
			})
			return fmt.Errorf("update affiliate rebate skipped audit: %w", err)
		}
		if err := tx.Commit(); err != nil {
			s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
				"error": fmt.Sprintf("commit affiliate rebate tx: %v", err),
			})
			return fmt.Errorf("commit affiliate rebate tx: %w", err)
		}
		return nil
	}

	if err := s.updateClaimedAffiliateRebateAudit(txCtx, tx.Client(), o.ID, "AFFILIATE_REBATE_APPLIED", map[string]any{
		"baseAmount":   baseAmount,
		"rebateAmount": rebateAmount,
	}); err != nil {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("update affiliate rebate applied audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": fmt.Sprintf("commit affiliate rebate tx: %v", err),
		})
		return fmt.Errorf("commit affiliate rebate tx: %w", err)
	}
	return nil
}

func affiliateRebateBaseAmount(o *dbent.PaymentOrder) float64 {
	if o == nil {
		return 0
	}
	switch o.OrderType {
	case payment.OrderTypeBalance:
		// 返利只按实充部分计算，赠送额度不参与
		return paymentOrderAmountWithoutBonus(o)
	case payment.OrderTypeSubscription:
		return o.Amount
	default:
		return 0
	}
}

func (s *PaymentService) tryClaimAffiliateRebateAudit(ctx context.Context, client *dbent.Client, orderID int64, baseAmount float64) (bool, error) {
	if client == nil {
		return false, errors.New("nil payment client")
	}
	oid := strconv.FormatInt(orderID, 10)
	detail, _ := json.Marshal(map[string]any{
		"baseAmount": baseAmount,
		"status":     "reserved",
	})
	rows, err := client.QueryContext(ctx, affiliateRebateAuditClaimSQL(), oid, string(detail))
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	var claimID int64
	if err := rows.Scan(&claimID); err != nil {
		return false, err
	}
	return true, nil
}

func affiliateRebateAuditClaimSQL() string {
	return `
INSERT INTO payment_audit_logs (order_id, action, detail, operator, created_at)
SELECT CAST($1 AS TEXT), 'AFFILIATE_REBATE_APPLIED', CAST($2 AS TEXT), 'system', CURRENT_TIMESTAMP
WHERE NOT EXISTS (
	SELECT 1
	FROM payment_audit_logs
	WHERE order_id = CAST($1 AS TEXT)
	  AND action IN ('AFFILIATE_REBATE_APPLIED', 'AFFILIATE_REBATE_SKIPPED')
)
ON CONFLICT (order_id, action) DO NOTHING
RETURNING id`
}

func (s *PaymentService) updateClaimedAffiliateRebateAudit(ctx context.Context, client *dbent.Client, orderID int64, action string, detail map[string]any) error {
	if client == nil {
		return errors.New("nil payment client")
	}
	oid := strconv.FormatInt(orderID, 10)
	detailJSON, _ := json.Marshal(detail)
	updated, err := client.PaymentAuditLog.Update().
		Where(
			paymentauditlog.OrderIDEQ(oid),
			paymentauditlog.ActionEQ("AFFILIATE_REBATE_APPLIED"),
		).
		SetAction(action).
		SetDetail(string(detailJSON)).
		SetOperator("system").
		Save(ctx)
	if err != nil {
		return err
	}
	if updated == 0 {
		return errors.New("affiliate rebate claim log not found")
	}
	return nil
}

func (s *PaymentService) markFailed(ctx context.Context, oid int64, lease *paymentFulfillmentLease, cause error) {
	if lease == nil {
		slog.Error("mark FAILED without fulfillment lease", "orderID", oid)
		return
	}
	now := time.Now()
	r := psErrMsg(cause)
	// Only mark fulfillment failed if still in RECHARGING state — prevents overwriting
	// a COMPLETED order when markCompleted failed but fulfillment succeeded.
	c, e := s.entClient.PaymentOrder.Update().
		Where(
			paymentorder.IDEQ(oid),
			paymentorder.StatusEQ(OrderStatusRecharging),
			paymentorder.UpdatedAtEQ(lease.version),
		).
		SetStatus(OrderStatusFulfillmentFailed).SetFailedAt(now).SetFailedReason(r).Save(ctx)
	if e != nil {
		slog.Error("mark fulfillment failed", "orderID", oid, "error", e)
	}
	if c > 0 {
		s.writeAuditLog(ctx, oid, "FULFILLMENT_FAILED", "system", map[string]any{"reason": r})
	}
}

func (s *PaymentService) scheduleFulfillmentRetries(oid int64) {
	delays := s.effectiveFulfillmentRetryDelays()
	if len(delays) == 0 {
		s.writeAuditLog(context.Background(), oid, "FULFILLMENT_MANUAL_REQUIRED", "system", map[string]any{
			"reason": "fulfillment retry attempts exhausted",
		})
		return
	}
	run := func() {
		s.runFulfillmentRetries(context.Background(), oid, delays)
	}
	if s.fulfillmentRetryAsync != nil {
		s.fulfillmentRetryAsync(run)
		return
	}
	go run()
}

func (s *PaymentService) effectiveFulfillmentRetryDelays() []time.Duration {
	if s != nil && s.fulfillmentRetryDelays != nil {
		return s.fulfillmentRetryDelays
	}
	return defaultFulfillmentRetryDelays
}

func (s *PaymentService) runFulfillmentRetries(ctx context.Context, oid int64, delays []time.Duration) {
	for attempt, delay := range delays {
		if !s.sleepBeforeFulfillmentRetry(ctx, delay) {
			return
		}
		s.writeAuditLog(ctx, oid, "FULFILLMENT_RETRY", "system", map[string]any{
			"attempt": attempt + 2,
			"delay":   delay.String(),
		})
		if err := s.executeFulfillment(ctx, oid); err != nil {
			slog.Warn("payment fulfillment retry failed",
				"orderID", oid,
				"attempt", attempt+2,
				"error", err,
			)
			continue
		}
		return
	}
	s.writeAuditLog(ctx, oid, "FULFILLMENT_MANUAL_REQUIRED", "system", map[string]any{
		"attempts": len(delays) + 1,
		"reason":   "fulfillment retry attempts exhausted",
	})
}

func (s *PaymentService) sleepBeforeFulfillmentRetry(ctx context.Context, delay time.Duration) bool {
	if s != nil && s.fulfillmentRetrySleep != nil {
		return s.fulfillmentRetrySleep(ctx, delay)
	}
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *PaymentService) RetryFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.PaidAt == nil {
		return infraerrors.BadRequest("INVALID_STATUS", "order is not paid")
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot retry")
	}
	if o.Status == OrderStatusCompleted {
		return infraerrors.BadRequest("INVALID_STATUS", "order already completed")
	}
	if o.Status != OrderStatusFailed && o.Status != OrderStatusFulfillmentFailed && o.Status != OrderStatusPaid {
		return infraerrors.BadRequest("INVALID_STATUS", "only paid and fulfillment-failed orders can retry")
	}
	_, err = s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(oid), paymentorder.StatusIn(OrderStatusFailed, OrderStatusFulfillmentFailed, OrderStatusPaid)).SetStatus(OrderStatusPaid).ClearFailedAt().ClearFailedReason().Save(ctx)
	if err != nil {
		return fmt.Errorf("reset for retry: %w", err)
	}
	s.writeAuditLog(ctx, oid, "RECHARGE_RETRY", "admin", map[string]any{"detail": "admin manual retry"})
	return s.executeFulfillment(ctx, oid)
}
