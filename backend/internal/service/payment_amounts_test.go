package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateGatewayRefundAmountForCurrencyThreeDecimals(t *testing.T) {
	// BHD 1.000 + 2.5% fee => pay 1.025; full refund must stay 1.025 not 1.03.
	got := calculateGatewayRefundAmountForCurrency(1.0, 1.025, 1.0, "BHD")
	require.InDelta(t, 1.025, got, 1e-9)

	// partial refund should keep three decimals
	got = calculateGatewayRefundAmountForCurrency(1.0, 1.025, 0.5, "BHD")
	require.InDelta(t, 0.513, got, 1e-9)
}

func TestCalculateGatewayRefundAmountDefaultTwoDecimals(t *testing.T) {
	got := calculateGatewayRefundAmount(10, 10.25, 10)
	require.InDelta(t, 10.25, got, 1e-9)
}
