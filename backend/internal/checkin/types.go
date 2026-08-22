package checkin

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	StatusAwarded              = "awarded"
	StatusBudgetExhausted      = "budget_exhausted"
	RedeemTypeCheckinBalance   = "checkin_balance"
	RedeemTypeCheckinMilestone = "checkin_milestone"
	redeemCodeStatusUsed       = "used"
	maxMilestones              = 10
	PeriodDay                  = "day"
	PeriodWeek                 = "week"
	PeriodMonth                = "month"
	PeriodAll                  = "all"
)

var rewardUnit = decimal.NewFromInt(10000)

type MilestoneRequest struct {
	Day   int    `json:"day"`
	Bonus string `json:"bonus"`
}

type SettingsRequest struct {
	Enabled    bool               `json:"enabled"`
	MinReward  string             `json:"min_reward"`
	MaxReward  string             `json:"max_reward"`
	Timezone   string             `json:"timezone"`
	DailyCap   string             `json:"daily_cap"`
	Milestones []MilestoneRequest `json:"milestones"`
}

type Milestone struct {
	Day   int
	Bonus decimal.Decimal
}

type Settings struct {
	Enabled    bool
	MinReward  decimal.Decimal
	MaxReward  decimal.Decimal
	Timezone   string
	DailyCap   decimal.Decimal
	Milestones []Milestone
	UpdatedAt  time.Time
}

func (s Settings) MaximumSingleReward() decimal.Decimal {
	maximumBonus := decimal.Zero
	for _, milestone := range s.Milestones {
		if milestone.Bonus.GreaterThan(maximumBonus) {
			maximumBonus = milestone.Bonus
		}
	}
	return s.MaxReward.Add(maximumBonus)
}

type Record struct {
	ID             int64
	UserID         int64
	UserEmail      string
	Username       string
	BusinessDate   time.Time
	CheckedAt      time.Time
	Timezone       string
	StreakDays     int
	CycleDay       int
	MilestoneDay   int
	BaseReward     decimal.Decimal
	MilestoneBonus decimal.Decimal
	ActualReward   decimal.Decimal
	Status         string
	BalanceAfter   decimal.Decimal
	ClientIP       string
	UserAgent      string
}

type CheckInResult struct {
	Record           Record
	AlreadyCheckedIn bool
}

type ClientInfo struct {
	IP        string
	UserAgent string
}

type RewardBreakdown struct {
	BaseReward     decimal.Decimal
	MilestoneBonus decimal.Decimal
	CycleDay       int
	MilestoneDay   int
}

func (r RewardBreakdown) Requested() decimal.Decimal { return r.BaseReward.Add(r.MilestoneBonus) }

type NextMilestone struct {
	Day       int
	Bonus     decimal.Decimal
	DaysUntil int
}

type UserStatus struct {
	Enabled        bool
	CheckedInToday bool
	TotalCheckIns  int64
	TotalReward    decimal.Decimal
	CurrentStreak  int
	CycleDay       int
	NextMilestone  *NextMilestone
	Balance        decimal.Decimal
	TodayRecord    *Record
	RecentRecords  []Record
}

type AdminRecordFilter struct {
	UserID           int64
	Search           string
	BusinessDate     *time.Time
	BusinessDateFrom *time.Time
	BusinessDateTo   *time.Time
	Status           string
	SortBy           string
	SortOrder        string
	Page             int
	PageSize         int
}

type AdminStats struct {
	Period       string
	Timezone     string
	From         *time.Time
	To           *time.Time
	UniqueUsers  int64
	CheckInCount int64
	TotalAmount  decimal.Decimal
	AvgAmount    decimal.Decimal
	P50Amount    decimal.Decimal
	P90Amount    decimal.Decimal
	MaxAmount    decimal.Decimal
}

type randomSource interface{ Int63n(int64) (int64, error) }

type cryptoRandomSource struct{}

func (cryptoRandomSource) Int63n(limit int64) (int64, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("random limit must be positive")
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(limit))
	if err != nil {
		return 0, err
	}
	return value.Int64(), nil
}

func normalizeSettings(request SettingsRequest) (Settings, error) {
	minimum, err := parseConfiguredAmount("min_reward", request.MinReward)
	if err != nil {
		return Settings{}, err
	}
	maximum, err := parseConfiguredAmount("max_reward", request.MaxReward)
	if err != nil {
		return Settings{}, err
	}
	dailyCap, err := parseConfiguredAmount("daily_cap", request.DailyCap)
	if err != nil {
		return Settings{}, err
	}
	if minimum.GreaterThan(maximum) {
		return Settings{}, fmt.Errorf("min_reward must not exceed max_reward")
	}
	if request.Enabled && !maximum.IsPositive() {
		return Settings{}, fmt.Errorf("max_reward must be greater than zero when check-in is enabled")
	}
	timezone := strings.TrimSpace(request.Timezone)
	if timezone == "" {
		return Settings{}, fmt.Errorf("timezone is required")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Settings{}, fmt.Errorf("invalid timezone: %w", err)
	}
	if len(request.Milestones) > maxMilestones {
		return Settings{}, fmt.Errorf("milestones must contain at most %d items", maxMilestones)
	}

	milestones := make([]Milestone, 0, len(request.Milestones))
	seen := make(map[int]struct{}, len(request.Milestones))
	for index, item := range request.Milestones {
		if item.Day <= 0 {
			return Settings{}, fmt.Errorf("milestones[%d].day must be positive", index)
		}
		if _, exists := seen[item.Day]; exists {
			return Settings{}, fmt.Errorf("milestone day %d is duplicated", item.Day)
		}
		bonus, err := parseConfiguredAmount(fmt.Sprintf("milestones[%d].bonus", index), item.Bonus)
		if err != nil {
			return Settings{}, err
		}
		seen[item.Day] = struct{}{}
		milestones = append(milestones, Milestone{Day: item.Day, Bonus: bonus})
	}
	sort.Slice(milestones, func(i, j int) bool { return milestones[i].Day < milestones[j].Day })
	return Settings{Enabled: request.Enabled, MinReward: minimum, MaxReward: maximum, Timezone: timezone, DailyCap: dailyCap, Milestones: milestones}, nil
}

func parseConfiguredAmount(field, raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Zero, fmt.Errorf("%s is required", field)
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%s must be a valid amount", field)
	}
	if amount.IsNegative() {
		return decimal.Zero, fmt.Errorf("%s must not be negative", field)
	}
	if amount.Exponent() < -4 {
		return decimal.Zero, fmt.Errorf("%s must have at most four decimal places", field)
	}
	return amount, nil
}

func formatAmount(amount decimal.Decimal) string { return amount.Round(4).StringFixed(4) }

func normalizeAdminStatsPeriod(period string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "", PeriodDay:
		return PeriodDay, nil
	case PeriodWeek:
		return PeriodWeek, nil
	case PeriodMonth:
		return PeriodMonth, nil
	case PeriodAll:
		return PeriodAll, nil
	default:
		return "", ErrInvalidStatsPeriod
	}
}

func calendarDateUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func periodDateRange(period string, now time.Time, timezone string) (from, to *time.Time, err error) {
	period, err = normalizeAdminStatsPeriod(period)
	if err != nil {
		return nil, nil, err
	}
	if period == PeriodAll {
		return nil, nil, nil
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid timezone: %w", err)
	}
	local := now.In(location)
	today := calendarDateUTC(local)
	toDate := today
	switch period {
	case PeriodDay:
		fromDate := today
		return &fromDate, &toDate, nil
	case PeriodWeek:
		weekday := int(local.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		fromDate := today.AddDate(0, 0, -(weekday - 1))
		return &fromDate, &toDate, nil
	case PeriodMonth:
		fromDate := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		return &fromDate, &toDate, nil
	default:
		return nil, nil, ErrInvalidStatsPeriod
	}
}

func randomReward(minimum, maximum decimal.Decimal, source randomSource) (decimal.Decimal, error) {
	minimumUnits, maximumUnits := minimum.Mul(rewardUnit).IntPart(), maximum.Mul(rewardUnit).IntPart()
	if minimumUnits > maximumUnits {
		return decimal.Zero, fmt.Errorf("minimum reward exceeds maximum reward")
	}
	offset, err := source.Int63n(maximumUnits - minimumUnits + 1)
	if err != nil {
		return decimal.Zero, fmt.Errorf("generate reward: %w", err)
	}
	return decimal.NewFromInt(minimumUnits + offset).Div(rewardUnit), nil
}

func nextStreak(today time.Time, previousBusinessDate *time.Time, previousStreak int) int {
	if previousBusinessDate == nil || !previousBusinessDate.Equal(today.AddDate(0, 0, -1)) {
		return 1
	}
	return previousStreak + 1
}

func rewardForStreak(settings Settings, streak int, source randomSource) (RewardBreakdown, error) {
	base, err := randomReward(settings.MinReward, settings.MaxReward, source)
	if err != nil {
		return RewardBreakdown{}, err
	}
	cycleDay := streak
	if len(settings.Milestones) > 0 {
		cycleLength := settings.Milestones[len(settings.Milestones)-1].Day
		cycleDay = ((streak - 1) % cycleLength) + 1
	}
	result := RewardBreakdown{BaseReward: base, CycleDay: cycleDay}
	for _, milestone := range settings.Milestones {
		if milestone.Day == cycleDay {
			result.MilestoneDay = milestone.Day
			result.MilestoneBonus = milestone.Bonus
			break
		}
	}
	return result, nil
}

func applyDailyCap(cap, awarded, requested decimal.Decimal) (decimal.Decimal, string) {
	if cap.IsZero() || awarded.Add(requested).LessThanOrEqual(cap) {
		return requested, StatusAwarded
	}
	return decimal.Zero, StatusBudgetExhausted
}

func nextMilestone(settings Settings, cycleDay int) *NextMilestone {
	if len(settings.Milestones) == 0 {
		return nil
	}
	for _, milestone := range settings.Milestones {
		if milestone.Day > cycleDay {
			return &NextMilestone{Day: milestone.Day, Bonus: milestone.Bonus, DaysUntil: milestone.Day - cycleDay}
		}
	}
	cycleLength := settings.Milestones[len(settings.Milestones)-1].Day
	first := settings.Milestones[0]
	return &NextMilestone{Day: first.Day, Bonus: first.Bonus, DaysUntil: cycleLength - cycleDay + first.Day}
}
