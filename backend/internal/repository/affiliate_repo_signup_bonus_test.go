package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestComputeAffiliateSignupBonusAwardKeepsFixedAmount(t *testing.T) {
	t.Parallel()

	award, reason := computeAffiliateSignupBonusAward(2.5, 10, 7.5, 100, 20)

	require.InDelta(t, 2.5, award, 1e-9)
	require.Empty(t, reason)
}

func TestComputeAffiliateSignupBonusAwardRejectsPartialInviterCap(t *testing.T) {
	t.Parallel()

	award, reason := computeAffiliateSignupBonusAward(2.5, 10, 8, 100, 20)

	require.Zero(t, award)
	require.Equal(t, "inviter_total_cap_reached", reason)
}

func TestComputeAffiliateSignupBonusAwardRejectsPartialDailyCap(t *testing.T) {
	t.Parallel()

	award, reason := computeAffiliateSignupBonusAward(2.5, 100, 20, 10, 8)

	require.Zero(t, award)
	require.Equal(t, "daily_total_cap_reached", reason)
}

func TestAffiliateSignupBonusLockKeysScopesByConfiguredGuardrails(t *testing.T) {
	t.Parallel()

	keys := affiliateSignupBonusLockKeys(service.AffiliateSignupBonusRequest{
		InviterID:       42,
		InviterTotalCap: 10,
	})

	require.Equal(t, []string{"affiliate_signup_bonus:inviter:42"}, keys)

	keys = affiliateSignupBonusLockKeys(service.AffiliateSignupBonusRequest{
		InviterID:       42,
		InviterTotalCap: 10,
		DailyTotalCap:   100,
		FingerprintHash: "abc123",
	})

	require.Equal(t, []string{
		"affiliate_signup_bonus:fingerprint:abc123",
		"affiliate_signup_bonus:inviter:42",
		"affiliate_signup_bonus:daily",
	}, keys)
}
