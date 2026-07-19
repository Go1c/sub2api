package service

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCalculateCreateOrderPayAmountUsesCurrencyPrecision(t *testing.T) {
	t.Parallel()

	// JPY: 0 fraction digits; fee rounds up to whole yen.
	payStr, payAmt, err := calculateCreateOrderPayAmount(1000, 3, "JPY")
	require.NoError(t, err)
	require.Equal(t, "1030", payStr)
	require.Equal(t, 1030.0, payAmt)

	// CNY default 2 digits with fee round-up.
	payStr, payAmt, err = calculateCreateOrderPayAmount(10, 3.33, payment.DefaultPaymentCurrency)
	require.NoError(t, err)
	require.Equal(t, "10.34", payStr)
	require.Equal(t, 10.34, payAmt)
}

func TestCalculateSubscriptionGatewayBaseAmount(t *testing.T) {
	t.Parallel()

	// No rate → pass-through.
	require.Equal(t, 10.0, calculateSubscriptionGatewayBaseAmount(10, 0, "CNY"))
	// Non-CNY gateway → no conversion even with rate.
	require.Equal(t, 10.0, calculateSubscriptionGatewayBaseAmount(10, 7.2, "USD"))
	// CNY + rate → multiply and round to currency digits.
	require.Equal(t, 72.0, calculateSubscriptionGatewayBaseAmount(10, 7.2, "CNY"))
}

func TestValidateCreateOrderAmountCurrencyRejectsSubMinorUnits(t *testing.T) {
	t.Parallel()

	err := validateCreateOrderAmountCurrency(10.5, "JPY")
	require.Error(t, err)
	require.NoError(t, validateCreateOrderAmountCurrency(10, "JPY"))
	require.NoError(t, validateCreateOrderAmountCurrency(10.5, "CNY"))
}

func TestBuildPaymentSubjectUsesSelectedCurrency(t *testing.T) {
	t.Parallel()

	svc := &PaymentService{}
	cfg := &PaymentConfig{}
	sel := &payment.InstanceSelection{
		ProviderKey: payment.TypeStripe,
		Config:      map[string]string{"currency": "usd"},
	}
	subject := svc.buildPaymentSubject(nil, 12.5, cfg, sel)
	require.Equal(t, "Sub2API 12.50 USD", subject)

	// Default when no selection.
	subject = svc.buildPaymentSubject(nil, 12.5, cfg, nil)
	require.Equal(t, "Sub2API 12.50 CNY", subject)
}

func TestPcAggregateMethodCurrency(t *testing.T) {
	t.Parallel()

	svc := &PaymentConfigService{}
	stripe := makeInstance(1, payment.TypeStripe, payment.TypeStripe, "")
	stripe.Config = `{"currency":"hkd"}`
	currency, ok := svc.pcAggregateMethodCurrency([]*dbent.PaymentProviderInstance{stripe})
	require.True(t, ok)
	require.Equal(t, "HKD", currency)

	airwallex := makeInstance(2, payment.TypeAirwallex, payment.TypeAirwallex, "")
	airwallex.Config = `{"currency":"usd"}`
	currency, ok = svc.pcAggregateMethodCurrency([]*dbent.PaymentProviderInstance{stripe, airwallex})
	require.False(t, ok)
	require.Empty(t, currency)

	easypay := makeInstance(3, payment.TypeEasyPay, payment.TypeAlipay, "")
	currency, ok = svc.pcAggregateMethodCurrency([]*dbent.PaymentProviderInstance{easypay})
	require.True(t, ok)
	require.Equal(t, payment.DefaultPaymentCurrency, currency)
}
