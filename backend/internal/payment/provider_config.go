package payment

import (
	"math"
	"strconv"
	"strings"
)

const (
	// ConfigKeyStripeCurrency controls the Stripe PaymentIntent currency.
	ConfigKeyStripeCurrency = "currency"
	// ConfigKeyBalanceRechargeMultiplier controls credited balance for this provider instance.
	ConfigKeyBalanceRechargeMultiplier       = "balanceRechargeMultiplier"
	ConfigKeyBalanceRechargeMultiplierLegacy = "balance_recharge_multiplier"
)

const defaultStripeCurrency = "cny"

// NormalizeStripeCurrency returns a Stripe currency supported by this payment integration.
func NormalizeStripeCurrency(config map[string]string) string {
	currency := strings.ToLower(strings.TrimSpace(config[ConfigKeyStripeCurrency]))
	switch currency {
	case "usd", "cny":
		return currency
	default:
		return defaultStripeCurrency
	}
}

// ProviderBalanceRechargeMultiplier resolves the balance credit multiplier for a provider instance.
func ProviderBalanceRechargeMultiplier(providerKey string, config map[string]string, fallback float64) float64 {
	if strings.TrimSpace(providerKey) != TypeStripe {
		return fallback
	}
	if v, ok := parsePositiveConfigFloat(config[ConfigKeyBalanceRechargeMultiplier]); ok {
		return v
	}
	if v, ok := parsePositiveConfigFloat(config[ConfigKeyBalanceRechargeMultiplierLegacy]); ok {
		return v
	}
	return fallback
}

func parsePositiveConfigFloat(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, false
	}
	return v, true
}
