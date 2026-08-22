//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountGrokNeedsReauth(t *testing.T) {
	require.False(t, accountGrokNeedsReauth(nil))
	require.True(t, accountGrokNeedsReauth(&Account{
		Status:       StatusError,
		ErrorMessage: "spending limit reached, please reauthorize",
	}))
	require.True(t, accountGrokNeedsReauth(&Account{
		Extra: map[string]any{"grok_needs_reauth": true},
	}))
	require.False(t, accountGrokNeedsReauth(&Account{
		Status: StatusActive,
		Extra:  map[string]any{"grok_needs_reauth": false},
	}))
}
