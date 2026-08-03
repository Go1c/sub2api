package service

import (
	"math"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

func calculateCreditedBalance(paymentAmount, multiplier float64) float64 {
	return decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2).
		InexactFloat64()
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64) float64 {
	return calculateGatewayRefundAmountForCurrency(orderAmount, payAmount, refundAmount, payment.DefaultPaymentCurrency)
}

// calculateGatewayRefundAmountForCurrency scales the gateway refund using the
// order's settlement currency precision. Full refunds reuse pay_amount exactly.
func calculateGatewayRefundAmountForCurrency(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	digits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= amountToleranceCNY {
		return decimal.NewFromFloat(payAmount).Round(digits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(digits).
		InexactFloat64()
}

func paymentOrderSettlementCurrency(o *dbent.PaymentOrder) string {
	if o == nil {
		return payment.DefaultPaymentCurrency
	}
	if snapshot := psOrderProviderSnapshot(o); snapshot != nil {
		if currency := strings.TrimSpace(snapshot.Currency); currency != "" {
			return strings.ToUpper(currency)
		}
	}
	return payment.DefaultPaymentCurrency
}
