package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRedeemCodeFromServiceIncludesCheckinMilestoneNotes(t *testing.T) {
	notes := "daily_checkin_milestone:8:7"
	out := RedeemCodeFromService(&service.RedeemCode{
		ID:    11,
		Code:  "aabbccddeeff00112233445566778899",
		Type:  "checkin_milestone",
		Value: 1,
		Notes: notes,
	})
	require.NotNil(t, out)
	require.NotNil(t, out.Notes)
	require.Equal(t, notes, *out.Notes)
}

func TestRedeemCodeFromServiceOmitsCheckinBalanceNotes(t *testing.T) {
	out := RedeemCodeFromService(&service.RedeemCode{
		ID:    8,
		Code:  "aabbccddeeff00112233445566778899",
		Type:  "checkin_balance",
		Value: 0.1,
		Notes: "daily_checkin:8",
	})
	require.NotNil(t, out)
	require.Nil(t, out.Notes)
}
