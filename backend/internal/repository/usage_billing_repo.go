package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	return r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
}

func (r *usageBillingRepository) claimUsageBillingRequest(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64, requestFingerprint string) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, requestID, apiKeyID, requestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, requestID, apiKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) ReserveBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, reserveUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) CaptureBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, captureUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) ReleaseBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, releaseUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) applyBatchImageBalanceHold(
	ctx context.Context,
	cmd *service.BatchImageBalanceHoldCommand,
	apply func(context.Context, *sql.Tx, *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error),
) (_ *service.BatchImageBalanceHoldResult, err error) {
	if cmd == nil {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}
	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.BatchImageBalanceHoldResult{Applied: false}, nil
	}

	result, err := apply(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &service.BatchImageBalanceHoldResult{}
	}
	result.Applied = true

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionID != nil || len(cmd.SubscriptionIDs) > 0 {
		if err := r.applyUsageBillingSubscription(ctx, tx, cmd, result); err != nil {
			return err
		}
	} else {
		result.BalanceCost = cmd.BalanceCost
	}

	if cmd.BalanceCost > 0 {
		newBalance, deducted, err := deductUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
		if err != nil {
			return err
		}
		result.NewBalance = &newBalance
		// 实际扣减可能因余额下限保护而小于请求金额；以实际扣减为准，
		// 保证用量日志 / 余额缓存 / 通知与 DB 真实余额一致。
		result.BalanceCost = deducted
		cmd.BalanceCost = deducted
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		quotaState, err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost)
		if err != nil {
			return err
		}
		result.QuotaState = quotaState
	}

	return nil
}

func (r *usageBillingRepository) applyUsageBillingSubscription(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	actualCost := math.Max(cmd.SubscriptionCost+cmd.BalanceCost, 0)
	if actualCost <= 0 {
		cmd.SubscriptionCost = 0
		cmd.BalanceCost = 0
		result.SubscriptionCost = 0
		result.BalanceCost = 0
		return nil
	}
	if len(cmd.SubscriptionIDs) > 1 {
		return r.applyUsageBillingMultipleSubscriptions(ctx, tx, cmd, result, actualCost)
	}

	subID := *cmd.SubscriptionID
	now := time.Now().UTC()
	state, err := lockAndReadSubscription(ctx, tx, subID)
	if err != nil {
		return err
	}

	dailyStart, weeklyStart := subscriptionQuotaWindowStarts(now, cmd.SubscriptionQuotaResetConfig)
	resetDaily := needResetSubscriptionWindow(state.DailyWindowStart, dailyStart)
	resetWeekly := needResetSubscriptionWindow(state.WeeklyWindowStart, weeklyStart)

	if err := r.recordWindowResetEvents(ctx, tx, cmd, state, resetDaily, resetWeekly); err != nil {
		return err
	}

	alloc := service.AllocateSubscriptionCredit(service.SubscriptionCreditAllocationInput{
		ActualCostUSD: actualCost,
		QuotaLimitUSD: state.QuotaLimitUSD,
		QuotaUsedUSD:  state.QuotaUsedUSD,
		Daily: service.SubscriptionCreditWindowState{
			LimitUSD: nullableFloatPtr(state.DailyLimitUSD),
			UsedUSD:  effectiveWindowUsage(state.DailyUsageUSD, resetDaily),
		},
		Weekly: service.SubscriptionCreditWindowState{
			LimitUSD: nullableFloatPtr(state.WeeklyLimitUSD),
			UsedUSD:  effectiveWindowUsage(state.WeeklyUsageUSD, resetWeekly),
		},
	})

	// 余额下限保护：订阅额度池覆盖不足时，溢出费用只能在用户正余额范围内扣除，
	// 绝不把余额扣成负数。超出可用余额的部分由平台吸收。提前在此收紧拆分，
	// 使消费台账（recordConsumeLedger）与后续实际扣减保持一致。
	if alloc.BalanceCostUSD > 0 {
		avail, balErr := lockUserBalanceForUpdate(ctx, tx, state.UserID)
		if balErr != nil {
			return balErr
		}
		avail = math.Max(avail, 0)
		if alloc.BalanceCostUSD > avail {
			alloc.BalanceCostUSD = avail
		}
	}

	post, err := updateSubscriptionUsage(ctx, tx, updateSubscriptionUsageInput{
		SubscriptionID: subID,
		SubCostUSD:     alloc.SubscriptionCostUSD,
		ResetDaily:     resetDaily,
		ResetWeekly:    resetWeekly,
		DailyStart:     dailyStart,
		WeeklyStart:    weeklyStart,
		Now:            now,
		PreExhausted:   state.ExhaustedAt.Valid,
		PreDailyUsage:  effectiveWindowUsage(state.DailyUsageUSD, resetDaily),
		PreWeeklyUsage: effectiveWindowUsage(state.WeeklyUsageUSD, resetWeekly),
	})
	if err != nil {
		return err
	}

	if err := r.recordLimitReachedEvents(ctx, tx, cmd, post, result); err != nil {
		return err
	}
	if err := r.recordConsumeLedger(ctx, tx, cmd, alloc, post, actualCost); err != nil {
		return err
	}

	cmd.SubscriptionCost = alloc.SubscriptionCostUSD
	cmd.BalanceCost = alloc.BalanceCostUSD
	result.SubscriptionCost = alloc.SubscriptionCostUSD
	result.BalanceCost = alloc.BalanceCostUSD
	return nil
}

type subscriptionBillingAllocation struct {
	state       *subscriptionBillingState
	resetDaily  bool
	resetWeekly bool
	dailyStart  time.Time
	weeklyStart time.Time
	subCostUSD  float64
	balanceUSD  float64
}

func (r *usageBillingRepository) applyUsageBillingMultipleSubscriptions(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult, actualCost float64) error {
	now := time.Now().UTC()
	states, err := lockAndReadSubscriptions(ctx, tx, cmd.UserID, cmd.SubscriptionIDs, now)
	if err != nil {
		return err
	}
	if len(states) == 0 {
		cmd.SubscriptionCost = 0
		cmd.BalanceCost = actualCost
		result.SubscriptionCost = 0
		result.BalanceCost = actualCost
		return nil
	}

	remaining := actualCost
	allocations := make([]subscriptionBillingAllocation, 0, len(states))
	totalSubCost := 0.0
	for _, state := range states {
		if remaining <= 0 {
			break
		}
		dailyStart, weeklyStart := subscriptionQuotaWindowStarts(now, cmd.SubscriptionQuotaResetConfig)
		resetDaily := needResetSubscriptionWindow(state.DailyWindowStart, dailyStart)
		resetWeekly := needResetSubscriptionWindow(state.WeeklyWindowStart, weeklyStart)
		alloc := service.AllocateSubscriptionCredit(service.SubscriptionCreditAllocationInput{
			ActualCostUSD: remaining,
			QuotaLimitUSD: state.QuotaLimitUSD,
			QuotaUsedUSD:  state.QuotaUsedUSD,
			Daily: service.SubscriptionCreditWindowState{
				LimitUSD: nullableFloatPtr(state.DailyLimitUSD),
				UsedUSD:  effectiveWindowUsage(state.DailyUsageUSD, resetDaily),
			},
			Weekly: service.SubscriptionCreditWindowState{
				LimitUSD: nullableFloatPtr(state.WeeklyLimitUSD),
				UsedUSD:  effectiveWindowUsage(state.WeeklyUsageUSD, resetWeekly),
			},
		})
		subCost := math.Min(math.Max(alloc.SubscriptionCostUSD, 0), remaining)
		allocations = append(allocations, subscriptionBillingAllocation{
			state:       state,
			resetDaily:  resetDaily,
			resetWeekly: resetWeekly,
			dailyStart:  dailyStart,
			weeklyStart: weeklyStart,
			subCostUSD:  subCost,
		})
		totalSubCost += subCost
		remaining = math.Max(remaining-subCost, 0)
	}

	balanceCost := remaining
	if balanceCost > 0 {
		avail, balErr := lockUserBalanceForUpdate(ctx, tx, cmd.UserID)
		if balErr != nil {
			return balErr
		}
		avail = math.Max(avail, 0)
		if balanceCost > avail {
			balanceCost = avail
		}
	}
	if balanceCost > 0 {
		for i := len(allocations) - 1; i >= 0; i-- {
			if allocations[i].subCostUSD > 0 {
				allocations[i].balanceUSD = balanceCost
				break
			}
		}
	}

	previousSubscriptionID := cmd.SubscriptionID
	defer func() {
		cmd.SubscriptionID = previousSubscriptionID
	}()
	for _, alloc := range allocations {
		if alloc.subCostUSD <= 0 && alloc.balanceUSD <= 0 && !alloc.resetDaily && !alloc.resetWeekly {
			continue
		}
		subID := alloc.state.ID
		cmd.SubscriptionID = &subID
		if err := r.recordWindowResetEvents(ctx, tx, cmd, alloc.state, alloc.resetDaily, alloc.resetWeekly); err != nil {
			return err
		}
		post, err := updateSubscriptionUsage(ctx, tx, updateSubscriptionUsageInput{
			SubscriptionID: subID,
			SubCostUSD:     alloc.subCostUSD,
			ResetDaily:     alloc.resetDaily,
			ResetWeekly:    alloc.resetWeekly,
			DailyStart:     alloc.dailyStart,
			WeeklyStart:    alloc.weeklyStart,
			Now:            now,
			PreExhausted:   alloc.state.ExhaustedAt.Valid,
			PreDailyUsage:  effectiveWindowUsage(alloc.state.DailyUsageUSD, alloc.resetDaily),
			PreWeeklyUsage: effectiveWindowUsage(alloc.state.WeeklyUsageUSD, alloc.resetWeekly),
		})
		if err != nil {
			return err
		}
		if err := r.recordLimitReachedEvents(ctx, tx, cmd, post, result); err != nil {
			return err
		}
		if err := r.recordConsumeLedger(ctx, tx, cmd, service.SubscriptionCreditAllocation{
			SubscriptionCostUSD: alloc.subCostUSD,
			BalanceCostUSD:      alloc.balanceUSD,
		}, post, actualCost); err != nil {
			return err
		}
	}

	cmd.SubscriptionCost = totalSubCost
	cmd.BalanceCost = balanceCost
	result.SubscriptionCost = totalSubCost
	result.BalanceCost = balanceCost
	return nil
}

type subscriptionBillingState struct {
	ID                int64
	UserID            int64
	QuotaLimitUSD     float64
	QuotaUsedUSD      float64
	DailyLimitUSD     sql.NullFloat64
	WeeklyLimitUSD    sql.NullFloat64
	DailyWindowStart  sql.NullTime
	WeeklyWindowStart sql.NullTime
	DailyUsageUSD     float64
	WeeklyUsageUSD    float64
	ExhaustedAt       sql.NullTime
}

type subscriptionBillingPostState struct {
	QuotaLimitUSD      float64
	QuotaUsedUSD       float64
	DailyLimitUSD      sql.NullFloat64
	DailyUsageUSD      float64
	WeeklyLimitUSD     sql.NullFloat64
	WeeklyUsageUSD     float64
	DailyWindowStart   sql.NullTime
	WeeklyWindowStart  sql.NullTime
	ExhaustedAt        sql.NullTime
	JustExhaustedTotal bool
	JustHitDaily       bool
	JustHitWeekly      bool
}

func lockAndReadSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64) (*subscriptionBillingState, error) {
	var state subscriptionBillingState
	err := tx.QueryRowContext(ctx, `
		SELECT
			id, user_id,
			quota_limit_usd, quota_used_usd,
			daily_limit_usd, weekly_limit_usd,
			daily_window_start, weekly_window_start,
			daily_usage_usd, weekly_usage_usd,
			exhausted_at
		FROM user_subscriptions
		WHERE id = $1
			AND deleted_at IS NULL
			AND status = $2
		FOR UPDATE
	`, subscriptionID, service.SubscriptionStatusActive).Scan(
		&state.ID,
		&state.UserID,
		&state.QuotaLimitUSD,
		&state.QuotaUsedUSD,
		&state.DailyLimitUSD,
		&state.WeeklyLimitUSD,
		&state.DailyWindowStart,
		&state.WeeklyWindowStart,
		&state.DailyUsageUSD,
		&state.WeeklyUsageUSD,
		&state.ExhaustedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func lockAndReadSubscriptions(ctx context.Context, tx *sql.Tx, userID int64, subscriptionIDs []int64, now time.Time) ([]*subscriptionBillingState, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id, user_id,
			quota_limit_usd, quota_used_usd,
			daily_limit_usd, weekly_limit_usd,
			daily_window_start, weekly_window_start,
			daily_usage_usd, weekly_usage_usd,
			exhausted_at
		FROM user_subscriptions
		WHERE id = ANY($1)
			AND user_id = $2
			AND deleted_at IS NULL
			AND status = $3
			AND exhausted_at IS NULL
			AND expires_at > $4
		ORDER BY created_at ASC, id ASC
		FOR UPDATE
	`, pq.Array(subscriptionIDs), userID, service.SubscriptionStatusActive, now)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var states []*subscriptionBillingState
	for rows.Next() {
		var state subscriptionBillingState
		if err := rows.Scan(
			&state.ID,
			&state.UserID,
			&state.QuotaLimitUSD,
			&state.QuotaUsedUSD,
			&state.DailyLimitUSD,
			&state.WeeklyLimitUSD,
			&state.DailyWindowStart,
			&state.WeeklyWindowStart,
			&state.DailyUsageUSD,
			&state.WeeklyUsageUSD,
			&state.ExhaustedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, &state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

type updateSubscriptionUsageInput struct {
	SubscriptionID int64
	SubCostUSD     float64
	ResetDaily     bool
	ResetWeekly    bool
	DailyStart     time.Time
	WeeklyStart    time.Time
	Now            time.Time
	PreExhausted   bool
	PreDailyUsage  float64
	PreWeeklyUsage float64
}

func updateSubscriptionUsage(ctx context.Context, tx *sql.Tx, in updateSubscriptionUsageInput) (*subscriptionBillingPostState, error) {
	var post subscriptionBillingPostState
	err := tx.QueryRowContext(ctx, `
		UPDATE user_subscriptions
		SET
			quota_used_usd = quota_used_usd + $2,
			daily_usage_usd = CASE WHEN $3 THEN $2 ELSE daily_usage_usd + $2 END,
			weekly_usage_usd = CASE WHEN $4 THEN $2 ELSE weekly_usage_usd + $2 END,
			daily_window_start = CASE WHEN $3 THEN $5 ELSE daily_window_start END,
			weekly_window_start = CASE WHEN $4 THEN $6 ELSE weekly_window_start END,
			exhausted_at = CASE
				WHEN exhausted_at IS NULL
					AND quota_limit_usd > 0
					AND quota_used_usd < quota_limit_usd
					AND quota_used_usd + $2 >= quota_limit_usd - $11::numeric
				THEN $7
				ELSE exhausted_at
			END,
			updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			quota_limit_usd,
			quota_used_usd,
			daily_limit_usd,
			daily_usage_usd,
			weekly_limit_usd,
			weekly_usage_usd,
			daily_window_start,
			weekly_window_start,
			exhausted_at,
			(NOT $8::boolean AND exhausted_at IS NOT NULL) AS just_exhausted_total,
			(daily_limit_usd IS NOT NULL AND daily_limit_usd > 0 AND daily_usage_usd >= daily_limit_usd AND $9::numeric < daily_limit_usd) AS just_hit_daily,
			(weekly_limit_usd IS NOT NULL AND weekly_limit_usd > 0 AND weekly_usage_usd >= weekly_limit_usd AND $10::numeric < weekly_limit_usd) AS just_hit_weekly
	`,
		in.SubscriptionID,
		in.SubCostUSD,
		in.ResetDaily,
		in.ResetWeekly,
		in.DailyStart,
		in.WeeklyStart,
		in.Now,
		in.PreExhausted,
		in.PreDailyUsage,
		in.PreWeeklyUsage,
		service.SubscriptionQuotaExhaustionEpsilonUSD,
	).Scan(
		&post.QuotaLimitUSD,
		&post.QuotaUsedUSD,
		&post.DailyLimitUSD,
		&post.DailyUsageUSD,
		&post.WeeklyLimitUSD,
		&post.WeeklyUsageUSD,
		&post.DailyWindowStart,
		&post.WeeklyWindowStart,
		&post.ExhaustedAt,
		&post.JustExhaustedTotal,
		&post.JustHitDaily,
		&post.JustHitWeekly,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *usageBillingRepository) recordWindowResetEvents(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, state *subscriptionBillingState, resetDaily, resetWeekly bool) error {
	if resetDaily {
		if err := r.recordWindowResetEvent(ctx, tx, cmd, state, "daily", state.DailyLimitUSD, state.DailyUsageUSD, state.DailyWindowStart); err != nil {
			return err
		}
	}
	if resetWeekly {
		if err := r.recordWindowResetEvent(ctx, tx, cmd, state, "weekly", state.WeeklyLimitUSD, state.WeeklyUsageUSD, state.WeeklyWindowStart); err != nil {
			return err
		}
	}
	return nil
}

func (r *usageBillingRepository) recordWindowResetEvent(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, state *subscriptionBillingState, window string, limit sql.NullFloat64, used float64, windowStart sql.NullTime) error {
	if !limit.Valid || limit.Float64 <= 0 || !windowStart.Valid {
		return nil
	}
	wasted := math.Max(limit.Float64-used, 0)
	eventKey := fmt.Sprintf("window_reset_%s:%d:%s", window, state.ID, windowStart.Time.UTC().Format(time.RFC3339))
	_, err := NewSubscriptionCreditLedgerRepository(r.db).CreateLimitReachedEvent(ctx, tx, &service.SubscriptionCreditLedgerEntry{
		UserID:            cmd.UserID,
		SubscriptionID:    state.ID,
		Type:              service.SubscriptionCreditLedgerWindowReset,
		DeltaUSD:          0,
		RemainingAfterUSD: math.Max(state.QuotaLimitUSD-state.QuotaUsedUSD, 0),
		Reason:            window + " window reset",
		EventKey:          &eventKey,
		Metadata: map[string]any{
			"window":                window,
			"limit_usd":             limit.Float64,
			"used_before_reset_usd": used,
			"wasted_usd":            wasted,
			"wasted_ratio":          safeRatio(wasted, limit.Float64),
			"old_window_start":      windowStart.Time.UTC().Format(time.RFC3339),
		},
	})
	return err
}

func (r *usageBillingRepository) recordLimitReachedEvents(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, post *subscriptionBillingPostState, result *service.UsageBillingApplyResult) error {
	if post.JustExhaustedTotal {
		created, err := r.recordLimitReachedEvent(ctx, tx, cmd, service.SubscriptionLimitReachedTotal, fmt.Sprintf("total:%d", *cmd.SubscriptionID), post)
		if err != nil {
			return err
		}
		if created {
			result.LimitReachedKinds = append(result.LimitReachedKinds, service.SubscriptionLimitReachedTotal)
		}
	}
	if post.JustHitDaily {
		start := nullableTimeRFC3339(post.DailyWindowStart)
		created, err := r.recordLimitReachedEvent(ctx, tx, cmd, service.SubscriptionLimitReachedDaily, fmt.Sprintf("daily:%d:%s", *cmd.SubscriptionID, start), post)
		if err != nil {
			return err
		}
		if created {
			result.LimitReachedKinds = append(result.LimitReachedKinds, service.SubscriptionLimitReachedDaily)
		}
	}
	if post.JustHitWeekly {
		start := nullableTimeRFC3339(post.WeeklyWindowStart)
		created, err := r.recordLimitReachedEvent(ctx, tx, cmd, service.SubscriptionLimitReachedWeekly, fmt.Sprintf("weekly:%d:%s", *cmd.SubscriptionID, start), post)
		if err != nil {
			return err
		}
		if created {
			result.LimitReachedKinds = append(result.LimitReachedKinds, service.SubscriptionLimitReachedWeekly)
		}
	}
	return nil
}

func (r *usageBillingRepository) recordLimitReachedEvent(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, kind string, eventKey string, post *subscriptionBillingPostState) (bool, error) {
	created, err := NewSubscriptionCreditLedgerRepository(r.db).CreateLimitReachedEvent(ctx, tx, &service.SubscriptionCreditLedgerEntry{
		UserID:            cmd.UserID,
		SubscriptionID:    *cmd.SubscriptionID,
		APIKeyID:          positiveInt64Ptr(cmd.APIKeyID),
		Type:              service.SubscriptionCreditLedgerLimitReached,
		DeltaUSD:          0,
		RemainingAfterUSD: math.Max(post.QuotaLimitUSD-post.QuotaUsedUSD, 0),
		Reason:            "limit reached: " + kind,
		EventKey:          &eventKey,
		Metadata: map[string]any{
			"kind":             kind,
			"quota_limit_usd":  post.QuotaLimitUSD,
			"quota_used_usd":   post.QuotaUsedUSD,
			"daily_limit_usd":  nullableFloatValue(post.DailyLimitUSD),
			"daily_usage_usd":  post.DailyUsageUSD,
			"weekly_limit_usd": nullableFloatValue(post.WeeklyLimitUSD),
			"weekly_usage_usd": post.WeeklyUsageUSD,
		},
	})
	if err != nil || !created {
		return created, err
	}
	return true, enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventSubscriptionNotify, nil, nil, map[string]any{
		"user_id":         cmd.UserID,
		"subscription_id": *cmd.SubscriptionID,
		"kind":            "limit_reached_" + kind,
	})
}

func (r *usageBillingRepository) recordConsumeLedger(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, alloc service.SubscriptionCreditAllocation, post *subscriptionBillingPostState, actualCost float64) error {
	allocatedCost := alloc.SubscriptionCostUSD + alloc.BalanceCostUSD
	if actualCost <= 0 || allocatedCost <= 0 {
		return nil
	}
	return NewSubscriptionCreditLedgerRepository(r.db).Create(ctx, tx, &service.SubscriptionCreditLedgerEntry{
		UserID:            cmd.UserID,
		SubscriptionID:    *cmd.SubscriptionID,
		APIKeyID:          positiveInt64Ptr(cmd.APIKeyID),
		Type:              service.SubscriptionCreditLedgerConsume,
		DeltaUSD:          -alloc.SubscriptionCostUSD,
		BalanceDeltaUSD:   -alloc.BalanceCostUSD,
		RemainingAfterUSD: math.Max(post.QuotaLimitUSD-post.QuotaUsedUSD, 0),
		Reason:            "usage billing",
		Metadata: map[string]any{
			"actual_cost_usd":         allocatedCost,
			"request_actual_cost_usd": actualCost,
			"subscription_cost_usd":   alloc.SubscriptionCostUSD,
			"balance_cost_usd":        alloc.BalanceCostUSD,
		},
	})
}

func subscriptionQuotaWindowStarts(t time.Time, cfg service.SubscriptionQuotaResetConfig) (time.Time, time.Time) {
	cfg = service.NormalizeSubscriptionQuotaResetConfig(cfg)
	offset := time.Duration(cfg.UTCOffsetMinutes) * time.Minute
	local := t.UTC().Add(offset)

	dailyLocal := time.Date(local.Year(), local.Month(), local.Day(), cfg.ResetHour, 0, 0, 0, time.UTC)
	if local.Before(dailyLocal) {
		dailyLocal = dailyLocal.AddDate(0, 0, -1)
	}

	weekday := int(dailyLocal.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weeklyLocal := time.Date(dailyLocal.Year(), dailyLocal.Month(), dailyLocal.Day(), cfg.ResetHour, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(weekday - 1))

	return dailyLocal.Add(-offset), weeklyLocal.Add(-offset)
}

func needResetSubscriptionWindow(windowStart sql.NullTime, currentStart time.Time) bool {
	return !windowStart.Valid || windowStart.Time.UTC().Before(currentStart)
}

func effectiveWindowUsage(used float64, reset bool) float64 {
	if reset {
		return 0
	}
	return used
}

func nullableFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullableFloatValue(v sql.NullFloat64) any {
	if !v.Valid {
		return nil
	}
	return v.Float64
}

func nullableTimeRFC3339(v sql.NullTime) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339)
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func positiveInt64Ptr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

// deductUsageBillingBalance 扣减用户余额，并以 0 为下限：单次扣减最多扣到余额为 0，
// 绝不把余额扣成负数（产品语义：订阅扣完→作废→扣余额→余额不足直接拒绝后续请求）。
// 返回扣减后的余额以及本次实际扣减的金额（可能小于请求金额，差额由平台吸收）。
func deductUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (newBalance float64, deducted float64, err error) {
	var current float64
	err = tx.QueryRowContext(ctx, `
		SELECT balance FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, service.ErrUserNotFound
	}
	if err != nil {
		return 0, 0, err
	}

	deducted = math.Min(math.Max(amount, 0), math.Max(current, 0))
	if deducted <= 0 {
		return current, 0, nil
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, deducted, userID).Scan(&newBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, service.ErrUserNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	return newBalance, deducted, nil
}

// lockUserBalanceForUpdate 在当前事务内锁定并读取用户余额（FOR UPDATE）。
// 用于订阅溢出扣费前确定可用余额，确保扣费拆分与实际扣减一致、且不会把余额扣成负数。
func lockUserBalanceForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (float64, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `
		SELECT balance FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func reserveUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			frozen_balance = COALESCE(frozen_balance, 0) + $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, service.ErrBatchImageInsufficientBalance
}

func captureUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 && cmd.ActualAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if cmd.ActualAmount-cmd.HoldAmount > 0.00000001 {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance
				+ CASE WHEN $1 > $2 THEN $1 - $2 ELSE 0 END
				- CASE WHEN $2 > $1 THEN $2 - $1 ELSE 0 END,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.ActualAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

func releaseUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	// 释放前校验该 job 确实预留过 hold（hold request id 已被 claim），
	// 防止从未成功冻结的 job 触发"幻影释放"，从其他用户的冻结资金池中凭空生成余额。
	held, heldErr := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
	if heldErr != nil {
		return nil, heldErr
	}
	if !held {
		logger.LegacyPrintf("repository.usage_billing", "[BatchImage] release skipped, hold was never reserved: batch=%s", cmd.BatchID)
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

// batchImageHoldClaimExists 检查 hold request id 是否已在 dedup（或归档）表中被 claim，
// 即该 batch 的冻结操作确实成功提交过。
func batchImageHoldClaimExists(ctx context.Context, tx *sql.Tx, holdRequestID string, apiKeyID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	err = tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func userExistsForBilling(ctx context.Context, tx *sql.Tx, userID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) (*service.AccountQuotaState, error) {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN `+dailyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN `+dailyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+dailyExpiredExpr+` AND `+nextDailyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_daily_reset_at', `+nextDailyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+weeklyExpiredExpr+` AND `+nextWeeklyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_weekly_reset_at', `+nextWeeklyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0),
			COALESCE((extra->>'quota_daily_used')::numeric, 0),
			COALESCE((extra->>'quota_daily_limit')::numeric, 0),
			COALESCE((extra->>'quota_weekly_used')::numeric, 0),
			COALESCE((extra->>'quota_weekly_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return nil, err
	}

	var state service.AccountQuotaState
	if rows.Next() {
		if err := rows.Scan(
			&state.TotalUsed, &state.TotalLimit,
			&state.DailyUsed, &state.DailyLimit,
			&state.WeeklyUsed, &state.WeeklyLimit,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
	} else {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	// 必须在执行下一条 SQL 前显式关闭 rows：pq 驱动在同一连接上
	// 不允许前一条查询的结果集未耗尽时启动新查询，否则会返回
	// "unexpected Parse response" 错误。
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// 任意维度额度在本次递增中从"未超"跨越到"已超"时，必须刷新调度快照，
	// 否则 Redis 中缓存的 Account 仍显示旧的 used 值，后续请求会继续选中本账号，
	// 最终观察到 daily_used / weekly_used 大幅超过配置的 limit。
	// 对于日/周额度，即使本次触发了周期重置（pre=0、post=amount），
	// 判定式 (post-amount) < limit 同样成立，逻辑与总额度保持一致。
	crossedTotal := state.TotalLimit > 0 && state.TotalUsed >= state.TotalLimit && (state.TotalUsed-amount) < state.TotalLimit
	crossedDaily := state.DailyLimit > 0 && state.DailyUsed >= state.DailyLimit && (state.DailyUsed-amount) < state.DailyLimit
	crossedWeekly := state.WeeklyLimit > 0 && state.WeeklyUsed >= state.WeeklyLimit && (state.WeeklyUsed-amount) < state.WeeklyLimit
	if crossedTotal || crossedDaily || crossedWeekly {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return nil, err
		}
	}
	return &state, nil
}
