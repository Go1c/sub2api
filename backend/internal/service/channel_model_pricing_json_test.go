//go:build unit

package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelModelPricing_JSONUsesAdminFieldNames(t *testing.T) {
	raw := []byte(`{
		"platform": "deepseek",
		"models": ["deepseek-v4.1-flash"],
		"billing_mode": "token",
		"input_price": 0.000002,
		"output_price": 0.000008,
		"cache_read_price": 0.00000004
	}`)

	var got ChannelModelPricing
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "deepseek", got.Platform)
	require.Equal(t, []string{"deepseek-v4.1-flash"}, got.Models)
	require.Equal(t, BillingModeToken, got.BillingMode)
	require.NotNil(t, got.InputPrice)
	require.InDelta(t, 2e-6, *got.InputPrice, 1e-15)
	require.NotNil(t, got.OutputPrice)
	require.InDelta(t, 8e-6, *got.OutputPrice, 1e-15)
	require.NotNil(t, got.CacheReadPrice)
	require.InDelta(t, 4e-8, *got.CacheReadPrice, 1e-18)

	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	var object map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &object))
	for _, key := range []string{"models", "billing_mode", "input_price", "output_price", "cache_read_price"} {
		_, ok := object[key]
		require.Truef(t, ok, "response JSON missing %s: %s", key, encoded)
	}
	for _, key := range []string{"Models", "InputPrice", "BillingMode"} {
		_, ok := object[key]
		require.Falsef(t, ok, "response JSON still uses %s: %s", key, encoded)
	}

	var again ChannelModelPricing
	require.NoError(t, json.Unmarshal(encoded, &again))
	require.Equal(t, got.Models, again.Models)
	require.InDelta(t, *got.InputPrice, *again.InputPrice, 1e-15)
	require.InDelta(t, *got.CacheReadPrice, *again.CacheReadPrice, 1e-18)
}
