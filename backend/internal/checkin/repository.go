package checkin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrDisabled     = errors.New("daily check-in is disabled")
	ErrUserInactive = errors.New("user is not active")
	ErrUserNotFound = errors.New("user not found")
)

const settingsSelectColumns = `enabled, min_reward::text, max_reward::text, timezone,
	daily_cap::text, milestones, updated_at`

const recordSelectColumns = `id, user_id, user_email, username, business_date, checked_at,
	timezone, streak_days, cycle_day, milestone_day, base_reward::text,
	milestone_bonus::text, actual_reward::text, status, balance_after::text,
	client_ip, user_agent`

type sqlRepository struct {
	db     *sql.DB
	random randomSource
}

func newSQLRepository(db *sql.DB, random randomSource) *sqlRepository {
	if random == nil {
		random = cryptoRandomSource{}
	}
	return &sqlRepository{db: db, random: random}
}

type rowScanner interface {
	Scan(dest ...any) error
}

type storedMilestone struct {
	Day   int    `json:"day"`
	Bonus string `json:"bonus"`
}

func scanSettings(row rowScanner) (Settings, error) {
	var (
		settings       Settings
		minimum        string
		maximum        string
		dailyCap       string
		milestonesJSON []byte
	)
	if err := row.Scan(
		&settings.Enabled,
		&minimum,
		&maximum,
		&settings.Timezone,
		&dailyCap,
		&milestonesJSON,
		&settings.UpdatedAt,
	); err != nil {
		return Settings{}, err
	}

	var err error
	if settings.MinReward, err = decimal.NewFromString(minimum); err != nil {
		return Settings{}, fmt.Errorf("parse min_reward: %w", err)
	}
	if settings.MaxReward, err = decimal.NewFromString(maximum); err != nil {
		return Settings{}, fmt.Errorf("parse max_reward: %w", err)
	}
	if settings.DailyCap, err = decimal.NewFromString(dailyCap); err != nil {
		return Settings{}, fmt.Errorf("parse daily_cap: %w", err)
	}

	var stored []storedMilestone
	if err := json.Unmarshal(milestonesJSON, &stored); err != nil {
		return Settings{}, fmt.Errorf("decode milestones: %w", err)
	}
	settings.Milestones = make([]Milestone, 0, len(stored))
	for _, item := range stored {
		bonus, err := decimal.NewFromString(item.Bonus)
		if err != nil {
			return Settings{}, fmt.Errorf("parse milestone bonus for day %d: %w", item.Day, err)
		}
		settings.Milestones = append(settings.Milestones, Milestone{Day: item.Day, Bonus: bonus})
	}
	return settings, nil
}

func encodeMilestones(milestones []Milestone) ([]byte, error) {
	stored := make([]storedMilestone, 0, len(milestones))
	for _, milestone := range milestones {
		stored = append(stored, storedMilestone{Day: milestone.Day, Bonus: formatAmount(milestone.Bonus)})
	}
	return json.Marshal(stored)
}

func (r *sqlRepository) GetSettings(ctx context.Context) (Settings, error) {
	if r == nil || r.db == nil {
		return Settings{}, errors.New("check-in repository is not initialized")
	}
	settings, err := scanSettings(r.db.QueryRowContext(ctx, `
		SELECT `+settingsSelectColumns+`
		FROM daily_checkin_settings
		WHERE id = 1`))
	if err != nil {
		return Settings{}, fmt.Errorf("get check-in settings: %w", err)
	}
	return settings, nil
}

func (r *sqlRepository) UpdateSettings(ctx context.Context, settings Settings) (Settings, error) {
	if r == nil || r.db == nil {
		return Settings{}, errors.New("check-in repository is not initialized")
	}
	milestones, err := encodeMilestones(settings.Milestones)
	if err != nil {
		return Settings{}, fmt.Errorf("encode milestones: %w", err)
	}
	updated, err := scanSettings(r.db.QueryRowContext(ctx, `
		UPDATE daily_checkin_settings
		SET enabled = $1,
			min_reward = $2,
			max_reward = $3,
			timezone = $4,
			daily_cap = $5,
			milestones = $6,
			updated_at = NOW()
		WHERE id = 1
		RETURNING `+settingsSelectColumns,
		settings.Enabled,
		settings.MinReward.StringFixed(4),
		settings.MaxReward.StringFixed(4),
		settings.Timezone,
		settings.DailyCap.StringFixed(4),
		milestones,
	))
	if err != nil {
		return Settings{}, fmt.Errorf("update check-in settings: %w", err)
	}
	return updated, nil
}

func businessDateAt(now time.Time, timezone string) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load check-in timezone: %w", err)
	}
	year, month, day := now.In(location).Date()
	// PostgreSQL DATE has no timezone. Keep it in UTC so calendar comparisons do
	// not depend on the database driver's session timezone.
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func parseDatabaseAmount(field, value string) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse %s: %w", field, err)
	}
	return amount, nil
}

func scanRecord(row rowScanner) (Record, error) {
	var (
		record                       Record
		userID, milestoneDay         sql.NullInt64
		base, bonus, actual, balance string
	)
	if err := row.Scan(
		&record.ID,
		&userID,
		&record.UserEmail,
		&record.Username,
		&record.BusinessDate,
		&record.CheckedAt,
		&record.Timezone,
		&record.StreakDays,
		&record.CycleDay,
		&milestoneDay,
		&base,
		&bonus,
		&actual,
		&record.Status,
		&balance,
		&record.ClientIP,
		&record.UserAgent,
	); err != nil {
		return Record{}, err
	}
	if userID.Valid {
		record.UserID = userID.Int64
	}
	if milestoneDay.Valid {
		record.MilestoneDay = int(milestoneDay.Int64)
	}

	var err error
	if record.BaseReward, err = parseDatabaseAmount("base_reward", base); err != nil {
		return Record{}, err
	}
	if record.MilestoneBonus, err = parseDatabaseAmount("milestone_bonus", bonus); err != nil {
		return Record{}, err
	}
	if record.ActualReward, err = parseDatabaseAmount("actual_reward", actual); err != nil {
		return Record{}, err
	}
	if record.BalanceAfter, err = parseDatabaseAmount("balance_after", balance); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (r *sqlRepository) GetUserStatus(ctx context.Context, userID int64, now time.Time) (UserStatus, error) {
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return UserStatus{}, err
	}
	businessDate, err := businessDateAt(now, settings.Timezone)
	if err != nil {
		return UserStatus{}, err
	}

	var userState, balanceRaw string
	err = r.db.QueryRowContext(ctx, `
		SELECT status, balance::text
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&userState, &balanceRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return UserStatus{}, ErrUserNotFound
	}
	if err != nil {
		return UserStatus{}, fmt.Errorf("get check-in user: %w", err)
	}
	if userState != "active" {
		return UserStatus{}, ErrUserInactive
	}
	balance, err := parseDatabaseAmount("balance", balanceRaw)
	if err != nil {
		return UserStatus{}, err
	}

	status := UserStatus{Enabled: settings.Enabled, Balance: balance}
	var totalReward string
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(actual_reward), 0)::text
		FROM daily_checkin_records
		WHERE user_id = $1`, userID).Scan(&status.TotalCheckIns, &totalReward); err != nil {
		return UserStatus{}, fmt.Errorf("aggregate check-in records: %w", err)
	}
	if status.TotalReward, err = parseDatabaseAmount("total_reward", totalReward); err != nil {
		return UserStatus{}, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT `+recordSelectColumns+`
		FROM daily_checkin_records
		WHERE user_id = $1
		ORDER BY business_date DESC, checked_at DESC
		LIMIT 20`, userID)
	if err != nil {
		return UserStatus{}, fmt.Errorf("list recent check-in records: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return UserStatus{}, fmt.Errorf("scan recent check-in record: %w", err)
		}
		status.RecentRecords = append(status.RecentRecords, record)
	}
	if err := rows.Err(); err != nil {
		return UserStatus{}, fmt.Errorf("iterate recent check-in records: %w", err)
	}

	if len(status.RecentRecords) > 0 {
		latest := status.RecentRecords[0]
		switch {
		case sameBusinessDate(latest.BusinessDate, businessDate):
			status.CheckedInToday = true
			status.CurrentStreak = latest.StreakDays
			status.CycleDay = latest.CycleDay
			status.TodayRecord = &latest
		case sameBusinessDate(latest.BusinessDate, businessDate.AddDate(0, 0, -1)):
			status.CurrentStreak = latest.StreakDays
			status.CycleDay = latest.CycleDay
		}
	}
	status.NextMilestone = nextMilestone(settings, status.CycleDay)
	return status, nil
}

func sameBusinessDate(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func normalizeAdminFilter(filter AdminRecordFilter) AdminRecordFilter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 1000 {
		filter.PageSize = 1000
	}
	filter.Search = strings.TrimSpace(filter.Search)
	if len(filter.Search) > 100 {
		filter.Search = filter.Search[:100]
	}
	if filter.Status != StatusAwarded && filter.Status != StatusBudgetExhausted {
		filter.Status = ""
	}
	return filter
}
