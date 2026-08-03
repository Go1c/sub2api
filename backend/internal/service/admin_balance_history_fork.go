package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type balanceHistorySQLQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *adminServiceImpl) sumUserRechargeHistoryTotal(ctx context.Context, userID int64) (float64, error) {
	total, err := s.redeemCodeRepo.SumPositiveBalanceByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	subscriptionPayments, err := s.sumExternalSubscriptionPaymentAmount(ctx, userID)
	if err != nil {
		return 0, err
	}
	promoBalance, err := s.sumPromoBalanceHistoryAmount(ctx, userID)
	if err != nil {
		return 0, err
	}
	return total + subscriptionPayments + promoBalance, nil
}

func (s *adminServiceImpl) sumExternalSubscriptionPaymentAmount(ctx context.Context, userID int64) (float64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return 0, nil
	}
	return sumExternalSubscriptionPaymentAmount(ctx, s.entClient, userID)
}

func (s *adminServiceImpl) sumPromoBalanceHistoryAmount(ctx context.Context, userID int64) (float64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return 0, nil
	}
	return sumPromoBalanceHistoryAmount(ctx, s.entClient, userID)
}

func (s *adminServiceImpl) listSubscriptionPaymentHistoryForMerge(ctx context.Context, userID int64, needed int) ([]RedeemCode, int64, error) {
	if needed <= 0 {
		return nil, 0, nil
	}

	var (
		out   []RedeemCode
		total int64
	)
	for page := 1; len(out) < needed; page++ {
		params := pagination.PaginationParams{Page: page, PageSize: 1000}
		codes, currentTotal, err := s.listSubscriptionPaymentHistory(ctx, userID, params)
		if err != nil {
			return nil, 0, err
		}
		total = currentTotal
		out = append(out, codes...)
		if len(codes) < params.Limit() || int64(len(out)) >= total {
			break
		}
	}
	if len(out) > needed {
		out = out[:needed]
	}
	return out, total, nil
}

func (s *adminServiceImpl) listPromoBalanceHistoryForMerge(ctx context.Context, userID int64, needed int) ([]RedeemCode, int64, error) {
	if needed <= 0 {
		return nil, 0, nil
	}

	var (
		out   []RedeemCode
		total int64
	)
	for page := 1; len(out) < needed; page++ {
		params := pagination.PaginationParams{Page: page, PageSize: 1000}
		codes, currentTotal, err := s.listPromoBalanceHistory(ctx, userID, params)
		if err != nil {
			return nil, 0, err
		}
		total = currentTotal
		out = append(out, codes...)
		if len(codes) < params.Limit() || int64(len(out)) >= total {
			break
		}
	}
	if len(out) > needed {
		out = out[:needed]
	}
	return out, total, nil
}

func (s *adminServiceImpl) listPromoBalanceHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]RedeemCode, int64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return nil, 0, nil
	}
	return listPromoBalanceHistory(ctx, s.entClient, userID, params)
}

func listPromoBalanceHistory(ctx context.Context, q balanceHistorySQLQueryer, userID int64, params pagination.PaginationParams) ([]RedeemCode, int64, error) {
	if q == nil || userID <= 0 {
		return nil, 0, nil
	}

	rows, err := q.QueryContext(ctx, `
SELECT pcu.id,
       COALESCE(NULLIF(pc.code, ''), 'PROMO-' || pcu.id::text) AS code,
       pcu.bonus_amount::double precision,
       COALESCE(pc.notes, '') AS notes,
       pcu.used_at
FROM promo_code_usages pcu
LEFT JOIN promo_codes pc ON pc.id = pcu.promo_code_id
WHERE pcu.user_id = $1
ORDER BY pcu.used_at DESC, pcu.id DESC
OFFSET $2
LIMIT $3`, userID, params.Offset(), params.Limit())
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	codes := make([]RedeemCode, 0, params.Limit())
	for rows.Next() {
		var (
			id     int64
			code   string
			amount float64
			notes  string
			usedAt time.Time
		)
		if err := rows.Scan(&id, &code, &amount, &notes, &usedAt); err != nil {
			return nil, 0, err
		}
		codes = append(codes, promoBalanceHistoryItem(PromoCodeUsage{
			ID:          id,
			UserID:      userID,
			BonusAmount: amount,
			UsedAt:      usedAt,
			PromoCode:   &PromoCode{Code: code, Notes: notes},
		}))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	total, err := countPromoBalanceHistory(ctx, q, userID)
	if err != nil {
		return nil, 0, err
	}
	return codes, total, nil
}

func (s *adminServiceImpl) listSubscriptionPaymentHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]RedeemCode, int64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return nil, 0, nil
	}
	return listSubscriptionPaymentHistory(ctx, s.entClient, userID, params)
}

func balanceHistorySubscriptionPaymentStatuses() []string {
	return []string{
		OrderStatusCompleted,
		OrderStatusPaid,
		OrderStatusRecharging,
		OrderStatusFulfillmentFailed,
	}
}

func listSubscriptionPaymentHistory(ctx context.Context, q balanceHistorySQLQueryer, userID int64, params pagination.PaginationParams) ([]RedeemCode, int64, error) {
	if q == nil || userID <= 0 {
		return nil, 0, nil
	}

	statuses := balanceHistorySubscriptionPaymentStatuses()
	args := []any{userID, payment.OrderTypeSubscription, payment.TypeBalance}
	statusPlaceholders := make([]string, 0, len(statuses))
	for i, status := range statuses {
		args = append(args, status)
		statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", i+4))
	}
	offsetArg := len(args) + 1
	limitArg := len(args) + 2

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
SELECT po.id,
       COALESCE(NULLIF(po.recharge_code, ''), NULLIF(po.out_trade_no, ''), 'ORDER-' || po.id::text) AS code,
       po.pay_amount::double precision,
       po.status,
       po.payment_type,
       COALESCE(po.subscription_validity_days, po.subscription_days, 0) AS subscription_validity_days,
       COALESCE(NULLIF(sp.product_name, ''), NULLIF(sp.name, ''), '') AS plan_name,
       po.paid_at,
       po.completed_at,
       po.created_at
FROM payment_orders po
LEFT JOIN subscription_plans sp ON sp.id = po.plan_id
WHERE po.user_id = $1
  AND po.order_type = $2
  AND po.payment_type <> $3
  AND po.status IN (%s)
ORDER BY COALESCE(po.paid_at, po.completed_at, po.created_at) DESC, po.id DESC
OFFSET $%d
LIMIT $%d`, strings.Join(statusPlaceholders, ","), offsetArg, limitArg),
		append(args, params.Offset(), params.Limit())...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	codes := make([]RedeemCode, 0, params.Limit())
	for rows.Next() {
		var (
			id           int64
			code         string
			value        float64
			status       string
			paymentType  string
			validityDays int
			planName     sql.NullString
			paidAt       sql.NullTime
			completedAt  sql.NullTime
			createdAt    time.Time
		)
		if err := rows.Scan(&id, &code, &value, &status, &paymentType, &validityDays, &planName, &paidAt, &completedAt, &createdAt); err != nil {
			return nil, 0, err
		}
		codes = append(codes, subscriptionPaymentHistoryItem(id, code, value, status, paymentType, validityDays, planName.String, userID, paidAt, completedAt, createdAt))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	total, err := countSubscriptionPaymentHistory(ctx, q, userID)
	if err != nil {
		return nil, 0, err
	}
	return codes, total, nil
}

func countPromoBalanceHistory(ctx context.Context, q balanceHistorySQLQueryer, userID int64) (int64, error) {
	if q == nil || userID <= 0 {
		return 0, nil
	}
	rows, err := q.QueryContext(ctx, `
SELECT COUNT(*)
FROM promo_code_usages
WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	var total sql.NullInt64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func countSubscriptionPaymentHistory(ctx context.Context, q balanceHistorySQLQueryer, userID int64) (int64, error) {
	if q == nil || userID <= 0 {
		return 0, nil
	}

	statuses := balanceHistorySubscriptionPaymentStatuses()
	args := []any{userID, payment.OrderTypeSubscription, payment.TypeBalance}
	statusPlaceholders := make([]string, 0, len(statuses))
	for i, status := range statuses {
		args = append(args, status)
		statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", i+4))
	}

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
SELECT COUNT(*)
FROM payment_orders po
WHERE po.user_id = $1
  AND po.order_type = $2
  AND po.payment_type <> $3
  AND po.status IN (%s)`, strings.Join(statusPlaceholders, ",")),
		args...,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	var total sql.NullInt64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func sumExternalSubscriptionPaymentAmount(ctx context.Context, q balanceHistorySQLQueryer, userID int64) (float64, error) {
	if q == nil || userID <= 0 {
		return 0, nil
	}

	statuses := balanceHistorySubscriptionPaymentStatuses()
	args := []any{userID, payment.OrderTypeSubscription, payment.TypeBalance}
	statusPlaceholders := make([]string, 0, len(statuses))
	for i, status := range statuses {
		args = append(args, status)
		statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", i+4))
	}

	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
SELECT COALESCE(SUM(po.pay_amount), 0)
FROM payment_orders po
WHERE po.user_id = $1
  AND po.order_type = $2
  AND po.payment_type <> $3
  AND po.status IN (%s)`, strings.Join(statusPlaceholders, ",")),
		args...,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	var total sql.NullFloat64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func sumPromoBalanceHistoryAmount(ctx context.Context, q balanceHistorySQLQueryer, userID int64) (float64, error) {
	if q == nil || userID <= 0 {
		return 0, nil
	}
	rows, err := q.QueryContext(ctx, `
SELECT COALESCE(SUM(bonus_amount), 0)::double precision
FROM promo_code_usages
WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	var total sql.NullFloat64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func affiliateBalanceHistoryItem(id int64, action string, amount float64, userID int64, createdAt time.Time) RedeemCode {
	usedBy := userID
	usedAt := createdAt
	return RedeemCode{
		ID:        -id,
		Code:      fmt.Sprintf("AFF-%d", id),
		Type:      RedeemTypeAffiliateBalance,
		Value:     amount,
		Status:    StatusUsed,
		UsedBy:    &usedBy,
		UsedAt:    &usedAt,
		Notes:     action,
		CreatedAt: createdAt,
	}
}

func subscriptionPaymentHistoryItem(id int64, code string, amount float64, status, paymentType string, validityDays int, planName string, userID int64, paidAt, completedAt sql.NullTime, createdAt time.Time) RedeemCode {
	usedBy := userID
	usedAt := createdAt
	if paidAt.Valid {
		usedAt = paidAt.Time
	} else if completedAt.Valid {
		usedAt = completedAt.Time
	}
	notes := "订阅消费"
	if planName = strings.TrimSpace(planName); planName != "" {
		notes += "：" + planName
	}
	if paymentType = strings.TrimSpace(paymentType); paymentType != "" {
		notes += "，支付方式 " + paymentType
	}
	if status = strings.TrimSpace(status); status != "" {
		notes += "，状态 " + status
	}
	return RedeemCode{
		ID:           -900000000000 - id,
		Code:         code,
		Type:         RedeemTypeSubscriptionPayment,
		Value:        amount,
		Status:       StatusUsed,
		UsedBy:       &usedBy,
		UsedAt:       &usedAt,
		CreatedAt:    createdAt,
		ValidityDays: validityDays,
		Notes:        notes,
	}
}
