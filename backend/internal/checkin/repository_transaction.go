package checkin

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type lockedUser struct {
	email    string
	username string
	status   string
	balance  decimal.Decimal
}

func (r *sqlRepository) CheckIn(ctx context.Context, userID int64, now time.Time, client ClientInfo) (CheckInResult, error) {
	if r == nil || r.db == nil {
		return CheckInResult{}, errors.New("check-in repository is not initialized")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckInResult{}, fmt.Errorf("begin check-in transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	settings, err := scanSettings(tx.QueryRowContext(ctx, `
		SELECT `+settingsSelectColumns+`
		FROM daily_checkin_settings
		WHERE id = 1
		FOR UPDATE`))
	if err != nil {
		return CheckInResult{}, fmt.Errorf("lock check-in settings: %w", err)
	}
	if !settings.Enabled {
		return CheckInResult{}, ErrDisabled
	}
	businessDate, err := businessDateAt(now, settings.Timezone)
	if err != nil {
		return CheckInResult{}, err
	}

	user, err := lockCheckInUser(ctx, tx, userID)
	if err != nil {
		return CheckInResult{}, err
	}
	existing, err := scanRecord(tx.QueryRowContext(ctx, `
		SELECT `+recordSelectColumns+`
		FROM daily_checkin_records
		WHERE user_id = $1 AND business_date = $2`, userID, businessDate))
	if err == nil {
		if err := tx.Commit(); err != nil {
			return CheckInResult{}, fmt.Errorf("commit check-in replay: %w", err)
		}
		return CheckInResult{Record: existing, AlreadyCheckedIn: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CheckInResult{}, fmt.Errorf("find existing check-in: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO daily_checkin_daily_counters (business_date)
		VALUES ($1)
		ON CONFLICT (business_date) DO NOTHING`, businessDate); err != nil {
		return CheckInResult{}, fmt.Errorf("ensure daily check-in counter: %w", err)
	}
	var awardedRaw string
	if err := tx.QueryRowContext(ctx, `
		SELECT awarded_total::text
		FROM daily_checkin_daily_counters
		WHERE business_date = $1
		FOR UPDATE`, businessDate).Scan(&awardedRaw); err != nil {
		return CheckInResult{}, fmt.Errorf("lock daily check-in counter: %w", err)
	}
	awarded, err := parseDatabaseAmount("awarded_total", awardedRaw)
	if err != nil {
		return CheckInResult{}, err
	}

	previousDate, previousStreak, err := previousCheckIn(ctx, tx, userID, businessDate)
	if err != nil {
		return CheckInResult{}, err
	}
	streak := nextStreak(businessDate, previousDate, previousStreak)
	reward, err := rewardForStreak(settings, streak, r.random)
	if err != nil {
		return CheckInResult{}, err
	}
	actual, recordStatus := applyDailyCap(settings.DailyCap, awarded, reward.Requested())

	balanceAfter := user.balance
	if actual.IsPositive() {
		var balanceRaw string
		if err := tx.QueryRowContext(ctx, `
			UPDATE users
			SET balance = balance + $2, updated_at = NOW()
			WHERE id = $1
			RETURNING balance::text`, userID, actual.StringFixed(4)).Scan(&balanceRaw); err != nil {
			return CheckInResult{}, fmt.Errorf("award check-in balance: %w", err)
		}
		balanceAfter, err = parseDatabaseAmount("balance_after", balanceRaw)
		if err != nil {
			return CheckInResult{}, err
		}
	}

	record := Record{
		UserID:         userID,
		UserEmail:      user.email,
		Username:       user.username,
		BusinessDate:   businessDate,
		Timezone:       settings.Timezone,
		StreakDays:     streak,
		CycleDay:       reward.CycleDay,
		MilestoneDay:   reward.MilestoneDay,
		BaseReward:     reward.BaseReward,
		MilestoneBonus: reward.MilestoneBonus,
		ActualReward:   actual,
		Status:         recordStatus,
		BalanceAfter:   balanceAfter,
		ClientIP:       truncate(client.IP, 64),
		UserAgent:      client.UserAgent,
	}
	var milestoneDay any
	if record.MilestoneDay > 0 {
		milestoneDay = record.MilestoneDay
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO daily_checkin_records (
			user_id, user_email, username, business_date, timezone,
			streak_days, cycle_day, milestone_day, base_reward,
			milestone_bonus, actual_reward, status, balance_after,
			client_ip, user_agent
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15
		)
		RETURNING id, checked_at`,
		userID,
		record.UserEmail,
		record.Username,
		record.BusinessDate,
		record.Timezone,
		record.StreakDays,
		record.CycleDay,
		milestoneDay,
		record.BaseReward.StringFixed(4),
		record.MilestoneBonus.StringFixed(4),
		record.ActualReward.StringFixed(4),
		record.Status,
		record.BalanceAfter.StringFixed(4),
		record.ClientIP,
		record.UserAgent,
	).Scan(&record.ID, &record.CheckedAt); err != nil {
		return CheckInResult{}, fmt.Errorf("insert check-in record: %w", err)
	}

	if actual.IsPositive() {
		if _, err := tx.ExecContext(ctx, `
			UPDATE daily_checkin_daily_counters
			SET awarded_total = awarded_total + $2, updated_at = NOW()
			WHERE business_date = $1`, businessDate, actual.StringFixed(4)); err != nil {
			return CheckInResult{}, fmt.Errorf("update daily check-in counter: %w", err)
		}
		if err := insertUsedCheckInRedeemCode(ctx, tx, userID, record.ID, actual); err != nil {
			return CheckInResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return CheckInResult{}, fmt.Errorf("commit check-in transaction: %w", err)
	}
	return CheckInResult{Record: record}, nil
}

func generateCheckInRedeemCode() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func insertUsedCheckInRedeemCode(ctx context.Context, tx *sql.Tx, userID, recordID int64, actual decimal.Decimal) error {
	code, err := generateCheckInRedeemCode()
	if err != nil {
		return fmt.Errorf("generate check-in redeem code: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at)
VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW())`,
		code,
		RedeemTypeCheckinBalance,
		actual.StringFixed(4),
		redeemCodeStatusUsed,
		userID,
		fmt.Sprintf("daily_checkin:%d", recordID),
	); err != nil {
		return fmt.Errorf("insert check-in redeem code: %w", err)
	}
	return nil
}

func lockCheckInUser(ctx context.Context, tx *sql.Tx, userID int64) (lockedUser, error) {
	var user lockedUser
	var balanceRaw string
	err := tx.QueryRowContext(ctx, `
		SELECT email, username, status, balance::text
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, userID).Scan(&user.email, &user.username, &user.status, &balanceRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedUser{}, ErrUserNotFound
	}
	if err != nil {
		return lockedUser{}, fmt.Errorf("lock check-in user: %w", err)
	}
	if user.status != "active" {
		return lockedUser{}, ErrUserInactive
	}
	user.balance, err = parseDatabaseAmount("balance", balanceRaw)
	if err != nil {
		return lockedUser{}, err
	}
	return user, nil
}

func previousCheckIn(ctx context.Context, tx *sql.Tx, userID int64, businessDate time.Time) (*time.Time, int, error) {
	var previousDate time.Time
	var previousStreak int
	err := tx.QueryRowContext(ctx, `
		SELECT business_date, streak_days
		FROM daily_checkin_records
		WHERE user_id = $1 AND business_date < $2
		ORDER BY business_date DESC
		LIMIT 1`, userID, businessDate).Scan(&previousDate, &previousStreak)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get previous check-in: %w", err)
	}
	return &previousDate, previousStreak, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
