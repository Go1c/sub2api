package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const maxManualCompleteReasonLength = 500

// ManualCompleteOrder lets an admin recover an expired order after confirming
// the provider already collected the payment. It deliberately does not widen
// webhook recovery beyond the grace period.
func (s *PaymentService) ManualCompleteOrder(ctx context.Context, oid int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return infraerrors.BadRequest("MANUAL_REASON_REQUIRED", "manual completion reason is required")
	}
	if len([]rune(reason)) > maxManualCompleteReasonLength {
		return infraerrors.BadRequest("MANUAL_REASON_TOO_LONG", "manual completion reason is too long")
	}

	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.Status != OrderStatusExpired {
		return infraerrors.BadRequest("INVALID_STATUS", "only expired orders can be manually completed")
	}

	now := time.Now()
	updated, err := s.entClient.PaymentOrder.Update().
		Where(paymentorder.IDEQ(oid), paymentorder.StatusEQ(OrderStatusExpired)).
		SetStatus(OrderStatusPaid).
		SetPaidAt(now).
		ClearFailedAt().
		ClearFailedReason().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark manual-completed order paid: %w", err)
	}
	if updated == 0 {
		return infraerrors.Conflict("CONFLICT", "order status changed")
	}

	s.writeAuditLog(ctx, oid, "ORDER_MANUAL_SUPPLEMENTED", "admin", map[string]any{
		"previous_status": o.Status,
		"reason":          reason,
		"paidAmount":      o.PayAmount,
		"creditedAmount":  o.Amount,
	})

	return s.executeFulfillment(ctx, oid)
}
