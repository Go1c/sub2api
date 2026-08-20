//go:build unit

package checkin

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type fixedRandom struct {
	value int64
	err   error
}

func (f fixedRandom) Int63n(int64) (int64, error) { return f.value, f.err }

func TestPeriodDateRangeUsesOperatingTimezoneAndMondayWeek(t *testing.T) {
	now := time.Date(2026, 8, 19, 16, 30, 0, 0, time.UTC) // 2026-08-20 00:30 Asia/Shanghai, Thursday

	from, to, err := periodDateRange(PeriodDay, now, "Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, "2026-08-20", formatBusinessDate(*from))
	require.Equal(t, "2026-08-20", formatBusinessDate(*to))

	from, to, err = periodDateRange(PeriodWeek, now, "Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, "2026-08-17", formatBusinessDate(*from))
	require.Equal(t, "2026-08-20", formatBusinessDate(*to))

	from, to, err = periodDateRange(PeriodMonth, now, "Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, "2026-08-01", formatBusinessDate(*from))
	require.Equal(t, "2026-08-20", formatBusinessDate(*to))

	from, to, err = periodDateRange(PeriodAll, now, "Asia/Shanghai")
	require.NoError(t, err)
	require.Nil(t, from)
	require.Nil(t, to)

	_, _, err = periodDateRange("quarter", now, "Asia/Shanghai")
	require.ErrorIs(t, err, ErrInvalidStatsPeriod)
}

func TestNormalizeSettingsValidation(t *testing.T) {
	valid := SettingsRequest{Enabled: true, MinReward: "0.1000", MaxReward: "0.5000", Timezone: "Asia/Shanghai", DailyCap: "10", Milestones: []MilestoneRequest{{Day: 30, Bonus: "2"}, {Day: 7, Bonus: "1.2500"}}}
	settings, err := normalizeSettings(valid)
	require.NoError(t, err)
	require.Equal(t, []Milestone{{Day: 7, Bonus: decimal.RequireFromString("1.2500")}, {Day: 30, Bonus: decimal.RequireFromString("2")}}, settings.Milestones)
	require.Equal(t, "2.5000", formatAmount(settings.MaximumSingleReward()))

	tests := []SettingsRequest{
		{Enabled: true, MinReward: "0", MaxReward: "0", Timezone: "UTC", DailyCap: "0"},
		{MinReward: "1", MaxReward: "0.5", Timezone: "UTC", DailyCap: "0"},
		{MinReward: "0.00001", MaxReward: "1", Timezone: "UTC", DailyCap: "0"},
		{MinReward: "0", MaxReward: "1", Timezone: "Not/AZone", DailyCap: "0"},
		{MinReward: "0", MaxReward: "1", Timezone: "UTC", DailyCap: "0", Milestones: []MilestoneRequest{{Day: 7, Bonus: "1"}, {Day: 7, Bonus: "2"}}},
		{MinReward: "0", MaxReward: "1", Timezone: "UTC", DailyCap: "0", Milestones: []MilestoneRequest{{Day: 0, Bonus: "1"}}},
	}
	for _, request := range tests {
		_, err := normalizeSettings(request)
		require.Error(t, err)
	}
}

func TestRandomRewardIncludesBothBoundaries(t *testing.T) {
	minimum := decimal.RequireFromString("0.1000")
	maximum := decimal.RequireFromString("0.1002")
	low, err := randomReward(minimum, maximum, fixedRandom{value: 0})
	require.NoError(t, err)
	require.Equal(t, "0.1000", formatAmount(low))
	high, err := randomReward(minimum, maximum, fixedRandom{value: 2})
	require.NoError(t, err)
	require.Equal(t, "0.1002", formatAmount(high))
	_, err = randomReward(minimum, maximum, fixedRandom{err: errors.New("entropy")})
	require.ErrorContains(t, err, "entropy")
}

func TestStreakCycleMilestoneAndBudget(t *testing.T) {
	today := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	old := today.AddDate(0, 0, -2)
	require.Equal(t, 7, nextStreak(today, &yesterday, 6))
	require.Equal(t, 1, nextStreak(today, &old, 6))

	settings := Settings{MinReward: decimal.RequireFromString("0.1"), MaxReward: decimal.RequireFromString("0.1"), Milestones: []Milestone{{Day: 7, Bonus: decimal.RequireFromString("1")}, {Day: 30, Bonus: decimal.RequireFromString("3")}}}
	reward, err := rewardForStreak(settings, 37, fixedRandom{})
	require.NoError(t, err)
	require.Equal(t, 7, reward.CycleDay)
	require.Equal(t, 7, reward.MilestoneDay)
	require.Equal(t, "1.0000", formatAmount(reward.MilestoneBonus))

	actual, status := applyDailyCap(decimal.RequireFromString("10"), decimal.RequireFromString("9.9"), decimal.RequireFromString("0.2"))
	require.True(t, actual.IsZero())
	require.Equal(t, StatusBudgetExhausted, status)
	actual, status = applyDailyCap(decimal.Zero, decimal.RequireFromString("999"), decimal.RequireFromString("0.2"))
	require.Equal(t, "0.2000", formatAmount(actual))
	require.Equal(t, StatusAwarded, status)
}
