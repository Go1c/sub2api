package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountPoolIQAnswerMatchesFuzzyContains21(t *testing.T) {
	require.True(t, accountPoolIQAnswerMatches("答案是 21。", "21", true))
	require.True(t, accountPoolIQAnswerMatches("最少21颗", "21", true))
	require.False(t, accountPoolIQAnswerMatches("20", "21", true))
	require.False(t, accountPoolIQAnswerMatches("", "21", true))
}

func TestAccountPoolIQAnswerMatchesExactIgnoresPunctuation(t *testing.T) {
	require.True(t, accountPoolIQAnswerMatches("21。", "21", false))
	require.False(t, accountPoolIQAnswerMatches("答案是21", "21", false))
}

func TestPlanAccountPoolIQGroupsReplacesTheOtherIQGroup(t *testing.T) {
	next, changed := planAccountPoolIQGroups([]int64{1, 9}, 3, 9, 3)
	require.True(t, changed)
	require.Equal(t, []int64{1, 3}, next)

	same, changed := planAccountPoolIQGroups([]int64{1, 3}, 3, 9, 3)
	require.False(t, changed)
	require.Equal(t, []int64{1, 3}, same)
}

func TestPlanAccountPoolIQGroupsKeepsUnrelatedGroups(t *testing.T) {
	next, changed := planAccountPoolIQGroups([]int64{4, 5}, 3, 9, 9)
	require.True(t, changed)
	require.Equal(t, []int64{4, 5, 9}, next)
}

func TestAccountPoolAutoInspectNextDelayStaysInsideJitter(t *testing.T) {
	for i := 0; i < 50; i++ {
		delay := accountPoolAutoInspectNextDelay(10, 60)
		require.GreaterOrEqual(t, delay, 9*time.Minute)
		require.LessOrEqual(t, delay, 11*time.Minute)
	}
	require.Equal(t, 10*time.Minute, accountPoolAutoInspectNextDelay(10, 0))
}

func TestParseAccountPoolAutoInspectConfigFillsQuizDefaults(t *testing.T) {
	cfg := parseAccountPoolAutoInspectConfig(`{"interval_minutes":10,"jitter_seconds":60,"pause_minutes":1}`)
	require.Equal(t, "gpt-6-astra", cfg.Model)
	require.Equal(t, "21", cfg.Answer)
	require.Contains(t, cfg.Question, "糖果")
	require.True(t, cfg.FuzzyMatch)
}

func TestValidateAccountPoolAutoInspectConfigRejectsSameGroup(t *testing.T) {
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.Enabled = true
	cfg.CorrectGroupID = 7
	cfg.IncorrectGroupID = 7
	require.Error(t, validateAccountPoolAutoInspectConfig(cfg))
}

func TestValidateAccountPoolAutoInspectConfigCapsJitter(t *testing.T) {
	cfg := defaultAccountPoolAutoInspectConfig()
	cfg.IntervalMinutes = 10
	cfg.JitterSeconds = 600
	require.Error(t, validateAccountPoolAutoInspectConfig(cfg))
	require.Equal(t, 300, accountPoolAutoInspectMaxJitterSeconds(10))
}

func TestParseAccountPoolAutoInspectConfigIgnoresLegacyHealthFields(t *testing.T) {
	parsed := parseAccountPoolAutoInspectConfig(`{"enabled":true,"interval_minutes":10,"success_rate_threshold":50,"add_group_ids":[7],"remove_models":["gpt-6-astra"]}`)
	require.True(t, parsed.Enabled)
	require.Equal(t, 10, parsed.IntervalMinutes)
	require.Equal(t, "21", parsed.Answer)
	require.Equal(t, int64(0), parsed.CorrectGroupID)
}
