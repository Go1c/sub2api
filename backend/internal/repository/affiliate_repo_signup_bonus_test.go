package repository

import (
	"testing"

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
