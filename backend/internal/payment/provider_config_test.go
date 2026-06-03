//go:build unit

package payment

import "testing"

func TestNormalizeStripeCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config map[string]string
		want   string
	}{
		{name: "default", config: nil, want: "cny"},
		{name: "usd", config: map[string]string{"currency": "usd"}, want: "usd"},
		{name: "upper cny", config: map[string]string{"currency": "CNY"}, want: "cny"},
		{name: "unsupported falls back", config: map[string]string{"currency": "eur"}, want: "cny"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeStripeCurrency(tt.config); got != tt.want {
				t.Fatalf("NormalizeStripeCurrency() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProviderBalanceRechargeMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		providerKey string
		config      map[string]string
		fallback    float64
		want        float64
	}{
		{name: "non stripe uses fallback", providerKey: TypeAlipay, config: map[string]string{ConfigKeyBalanceRechargeMultiplier: "7"}, fallback: 1.5, want: 1.5},
		{name: "stripe camel key", providerKey: TypeStripe, config: map[string]string{ConfigKeyBalanceRechargeMultiplier: "7"}, fallback: 1, want: 7},
		{name: "stripe legacy snake key", providerKey: TypeStripe, config: map[string]string{ConfigKeyBalanceRechargeMultiplierLegacy: "6.8"}, fallback: 1, want: 6.8},
		{name: "stripe invalid uses fallback", providerKey: TypeStripe, config: map[string]string{ConfigKeyBalanceRechargeMultiplier: "-1"}, fallback: 2, want: 2},
		{name: "stripe empty uses fallback", providerKey: TypeStripe, config: nil, fallback: 3, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProviderBalanceRechargeMultiplier(tt.providerKey, tt.config, tt.fallback)
			if got != tt.want {
				t.Fatalf("ProviderBalanceRechargeMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}
