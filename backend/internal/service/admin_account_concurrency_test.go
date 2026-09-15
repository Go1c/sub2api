//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountConcurrencyDefaultsInvalidGrokOAuthToOne(t *testing.T) {
	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 0))
	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, -5))
}

func TestNormalizeAccountConcurrencyDefaultsOpenAIOmittedToTen(t *testing.T) {
	require.Equal(t, DefaultOpenAIAccountConcurrency, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeOAuth, 0))
	require.Equal(t, DefaultOpenAIAccountConcurrency, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeSetupToken, -1))
	require.Equal(t, 0, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeAPIKey, 0))
}

func TestNormalizeAccountConcurrencyPreservesExplicitValues(t *testing.T) {
	require.Equal(t, 50, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 50))
	require.Equal(t, 2, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeOAuth, 2))
	require.Equal(t, 2, normalizeAccountConcurrency(PlatformGrok, AccountTypeAPIKey, 2))
}
