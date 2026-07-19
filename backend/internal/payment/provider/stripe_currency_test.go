package provider

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v85"
)

func TestNewStripeNormalizesCurrency(t *testing.T) {
	t.Parallel()

	s, err := NewStripe("inst-1", map[string]string{
		"secretKey": "sk_test",
		"currency":  "usd",
	})
	require.NoError(t, err)
	require.Equal(t, "USD", s.currency())
	require.Equal(t, "USD", s.MerchantIdentityMetadata()["currency"])

	_, err = NewStripe("inst-2", map[string]string{
		"secretKey": "sk_test",
		"currency":  "US",
	})
	require.Error(t, err)

	s, err = NewStripe("inst-3", map[string]string{
		"secretKey": "sk_test",
	})
	require.NoError(t, err)
	require.Equal(t, payment.DefaultPaymentCurrency, s.currency())
}

func TestStripeIntentCurrency(t *testing.T) {
	t.Parallel()

	require.Equal(t, "USD", stripeIntentCurrency(stripe.CurrencyUSD, "CNY"))
	require.Equal(t, "CNY", stripeIntentCurrency(stripe.Currency(""), "cny"))
	// Empty intent currency uses fallback when it is a valid ISO code.
	require.Equal(t, "HKD", stripeIntentCurrency(stripe.Currency(""), "hkd"))
	// Invalid fallback falls back to default payment currency.
	require.Equal(t, payment.DefaultPaymentCurrency, stripeIntentCurrency(stripe.Currency(""), "US"))
}

func TestStripeRefundProviderStatus(t *testing.T) {
	t.Parallel()

	require.Equal(t, payment.ProviderStatusSuccess, stripeRefundProviderStatus(stripe.RefundStatusSucceeded))
	require.Equal(t, payment.ProviderStatusFailed, stripeRefundProviderStatus(stripe.RefundStatusFailed))
	require.Equal(t, payment.ProviderStatusFailed, stripeRefundProviderStatus(stripe.RefundStatusCanceled))
	require.Equal(t, payment.ProviderStatusPending, stripeRefundProviderStatus(stripe.RefundStatusPending))
}

func TestStripeCurrencyAmountMinorUnits(t *testing.T) {
	t.Parallel()

	// USD/CNY: 2 decimals
	usd, err := payment.AmountToMinorUnit("10.50", "USD")
	require.NoError(t, err)
	require.Equal(t, int64(1050), usd)

	// JPY: zero decimals
	jpy, err := payment.AmountToMinorUnit("1050", "JPY")
	require.NoError(t, err)
	require.Equal(t, int64(1050), jpy)

	_, err = payment.AmountToMinorUnit("10.5", "JPY")
	require.Error(t, err)
}
