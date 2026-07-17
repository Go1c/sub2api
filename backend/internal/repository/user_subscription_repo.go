package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userSubscriptionRepository struct {
	client *dbent.Client
	db     *sql.DB
}

// NewUserSubscriptionRepository 构造用户订阅仓储。
// sqlDB 用于额度池扩展方法（LockUserForSubscriptionWrite 等需要直接 SQL 操作）；
// 传 nil 时这些方法会返回 ErrSQLDBUnavailable。
func NewUserSubscriptionRepository(client *dbent.Client, sqlDB *sql.DB) service.UserSubscriptionRepository {
	return &userSubscriptionRepository{client: client, db: sqlDB}
}

func (r *userSubscriptionRepository) Create(ctx context.Context, sub *service.UserSubscription) error {
	if sub == nil {
		return service.ErrSubscriptionNilInput
	}

	client := clientFromContext(ctx, r.client)
	builder := client.UserSubscription.Create().
		SetUserID(sub.UserID).
		SetNillableGroupID(sub.GroupID).
		SetExpiresAt(sub.ExpiresAt).
		SetNillableDailyWindowStart(sub.DailyWindowStart).
		SetNillableWeeklyWindowStart(sub.WeeklyWindowStart).
		SetDailyUsageUsd(sub.DailyUsageUSD).
		SetWeeklyUsageUsd(sub.WeeklyUsageUSD).
		SetNillableAssignedBy(sub.AssignedBy)
	// 额度池字段
	if sub.PlanID != nil {
		builder.SetNillablePlanID(sub.PlanID)
	}
	if sub.ScopeType != "" {
		builder.SetScopeType(sub.ScopeType)
	}
	if sub.ScopeConfig != nil {
		builder.SetScopeConfig(sub.ScopeConfig)
	}
	builder.SetQuotaLimitUsd(sub.QuotaLimitUSD).
		SetQuotaUsedUsd(sub.QuotaUsedUSD).
		SetNillableDailyLimitUsd(sub.DailyLimitUSD).
		SetNillableWeeklyLimitUsd(sub.WeeklyLimitUSD).
		SetNillableExhaustedAt(sub.ExhaustedAt).
		SetNillableExpiredCreditLoggedAt(sub.ExpiredCreditLoggedAt).
		SetNillableWeeklyLimitUserResetAt(sub.WeeklyLimitUserResetAt)

	if sub.StartsAt.IsZero() {
		builder.SetStartsAt(time.Now())
	} else {
		builder.SetStartsAt(sub.StartsAt)
	}
	if sub.Status != "" {
		builder.SetStatus(sub.Status)
	}
	if !sub.AssignedAt.IsZero() {
		builder.SetAssignedAt(sub.AssignedAt)
	}
	// Keep compatibility with historical behavior: always store notes as a string value.
	builder.SetNotes(sub.Notes)

	created, err := builder.Save(ctx)
	if err == nil {
		applyUserSubscriptionEntityToService(sub, created)
	}
	return translatePersistenceError(err, nil, service.ErrSubscriptionAlreadyExists)
}

func (r *userSubscriptionRepository) GetByID(ctx context.Context, id int64) (*service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserSubscription.Query().
		Where(usersubscription.IDEQ(id)).
		WithUser().
		WithGroup().
		WithAssignedByUser().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	sub := userSubscriptionEntityToService(m)
	if err := r.attachPlanSummary(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (r *userSubscriptionRepository) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID)).
		WithGroup().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	sub := userSubscriptionEntityToService(m)
	if err := r.attachPlanSummary(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (r *userSubscriptionRepository) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.GroupIDEQ(groupID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		WithGroup().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	sub := userSubscriptionEntityToService(m)
	if err := r.attachPlanSummary(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (r *userSubscriptionRepository) Update(ctx context.Context, sub *service.UserSubscription) error {
	if sub == nil {
		return service.ErrSubscriptionNilInput
	}

	client := clientFromContext(ctx, r.client)
	builder := client.UserSubscription.UpdateOneID(sub.ID).
		SetUserID(sub.UserID).
		SetStartsAt(sub.StartsAt).
		SetExpiresAt(sub.ExpiresAt).
		SetStatus(sub.Status).
		SetNillableDailyWindowStart(sub.DailyWindowStart).
		SetNillableWeeklyWindowStart(sub.WeeklyWindowStart).
		SetDailyUsageUsd(sub.DailyUsageUSD).
		SetWeeklyUsageUsd(sub.WeeklyUsageUSD).
		SetNillableAssignedBy(sub.AssignedBy).
		SetAssignedAt(sub.AssignedAt).
		SetNotes(sub.Notes)
	if sub.GroupID != nil {
		builder.SetGroupID(*sub.GroupID)
	} else {
		builder.ClearGroupID()
	}
	// 额度池字段
	if sub.PlanID != nil {
		builder.SetNillablePlanID(sub.PlanID)
	} else {
		builder.ClearPlanID()
	}
	if sub.ScopeType != "" {
		builder.SetScopeType(sub.ScopeType)
	}
	if sub.ScopeConfig != nil {
		builder.SetScopeConfig(sub.ScopeConfig)
	}
	builder.SetQuotaLimitUsd(sub.QuotaLimitUSD).
		SetQuotaUsedUsd(sub.QuotaUsedUSD).
		SetNillableDailyLimitUsd(sub.DailyLimitUSD).
		SetNillableWeeklyLimitUsd(sub.WeeklyLimitUSD).
		SetNillableExhaustedAt(sub.ExhaustedAt).
		SetNillableExpiredCreditLoggedAt(sub.ExpiredCreditLoggedAt)
	if sub.WeeklyLimitUserResetAt != nil {
		builder.SetWeeklyLimitUserResetAt(*sub.WeeklyLimitUserResetAt)
	} else {
		builder.ClearWeeklyLimitUserResetAt()
	}

	updated, err := builder.Save(ctx)
	if err == nil {
		applyUserSubscriptionEntityToService(sub, updated)
		return nil
	}
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, service.ErrSubscriptionAlreadyExists)
}

func (r *userSubscriptionRepository) Delete(ctx context.Context, id int64) error {
	// Match GORM semantics: deleting a missing row is not an error.
	client := clientFromContext(ctx, r.client)
	_, err := client.UserSubscription.Delete().Where(usersubscription.IDEQ(id)).Exec(ctx)
	return err
}

func (r *userSubscriptionRepository) ListByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	subs, err := client.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(userID)).
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := userSubscriptionEntitiesToService(subs)
	if err := r.attachPlanSummaries(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userSubscriptionRepository) ListActiveByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	subs, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := userSubscriptionEntitiesToService(subs)
	if err := r.attachPlanSummaries(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userSubscriptionRepository) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	q := client.UserSubscription.Query().Where(usersubscription.GroupIDEQ(groupID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	subs, err := q.
		WithUser().
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := userSubscriptionEntitiesToService(subs)
	if err := r.attachRecent30dWaste(ctx, out); err != nil {
		return nil, nil, err
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *userSubscriptionRepository) attachRecent30dWaste(ctx context.Context, subs []service.UserSubscription) error {
	if r.db == nil || len(subs) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(subs))
	for i := range subs {
		ids = append(ids, subs[i].ID)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
	subscription_id,
	COALESCE(SUM(
		CASE
			WHEN type = 'expire' THEN ABS(delta_usd)
			WHEN type = 'window_reset' AND jsonb_typeof(metadata->'wasted_usd') = 'number'
				THEN (metadata->>'wasted_usd')::double precision
			ELSE 0
		END
	), 0)::double precision AS wasted_usd
FROM subscription_credit_ledger
WHERE subscription_id = ANY($1)
	AND created_at >= $2
	AND type IN ('expire', 'window_reset')
GROUP BY subscription_id
`, pq.Array(ids), time.Now().AddDate(0, 0, -30))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	wasteBySubID := make(map[int64]float64, len(subs))
	for rows.Next() {
		var subscriptionID int64
		var wastedUSD float64
		if err := rows.Scan(&subscriptionID, &wastedUSD); err != nil {
			return err
		}
		wasteBySubID[subscriptionID] = wastedUSD
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range subs {
		subs[i].Recent30dWastedUSD = wasteBySubID[subs[i].ID]
	}
	return nil
}

func (r *userSubscriptionRepository) List(ctx context.Context, params pagination.PaginationParams, filters service.UserSubscriptionListFilters) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	q := client.UserSubscription.Query()
	if filters.UserID != nil {
		q = q.Where(usersubscription.UserIDEQ(*filters.UserID))
	}
	if filters.GroupID != nil {
		q = q.Where(usersubscription.GroupIDEQ(*filters.GroupID))
	}
	if filters.PlanID != nil {
		q = q.Where(usersubscription.PlanIDEQ(*filters.PlanID))
	}
	if filters.Platform != "" {
		q = q.Where(usersubscription.HasGroupWith(group.PlatformEQ(filters.Platform)))
	}
	if filters.Email != "" {
		q = q.Where(usersubscription.HasUserWith(user.EmailContainsFold(filters.Email)))
	}
	if filters.CreatedStart != nil {
		q = q.Where(usersubscription.CreatedAtGTE(*filters.CreatedStart))
	}
	if filters.CreatedEnd != nil {
		q = q.Where(usersubscription.CreatedAtLTE(*filters.CreatedEnd))
	}

	// Status filtering with real-time expiration check
	now := time.Now()
	switch filters.Status {
	case service.SubscriptionStatusActive:
		// Active: status is active AND not yet expired
		q = q.Where(
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(now),
		)
	case service.SubscriptionStatusExpired:
		// Expired: status is expired OR (status is active but already expired)
		q = q.Where(
			usersubscription.Or(
				usersubscription.StatusEQ(service.SubscriptionStatusExpired),
				usersubscription.And(
					usersubscription.StatusEQ(service.SubscriptionStatusActive),
					usersubscription.ExpiresAtLTE(now),
				),
			),
		)
	case "":
		// No filter
	default:
		// Other status (e.g., revoked)
		q = q.Where(usersubscription.StatusEQ(filters.Status))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Apply sorting
	q = q.WithUser().WithGroup().WithAssignedByUser()

	// Determine sort field
	var field string
	switch filters.SortBy {
	case "expires_at":
		field = usersubscription.FieldExpiresAt
	case "status":
		field = usersubscription.FieldStatus
	default:
		field = usersubscription.FieldCreatedAt
	}

	// Determine sort order (default: desc)
	if filters.SortOrder == "asc" && filters.SortBy != "" {
		q = q.Order(dbent.Asc(field))
	} else {
		q = q.Order(dbent.Desc(field))
	}

	subs, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := userSubscriptionEntitiesToService(subs)
	if err := r.attachPlanSummaries(ctx, out); err != nil {
		return nil, nil, err
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *userSubscriptionRepository) ExistsByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	client := clientFromContext(ctx, r.client)
	return client.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID)).
		Exist(ctx)
}

func (r *userSubscriptionRepository) ExtendExpiry(ctx context.Context, subscriptionID int64, newExpiresAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).
		SetExpiresAt(newExpiresAt).
		Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}

func (r *userSubscriptionRepository) UpdateStatus(ctx context.Context, subscriptionID int64, status string) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).
		SetStatus(status).
		Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}

func (r *userSubscriptionRepository) UpdateNotes(ctx context.Context, subscriptionID int64, notes string) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).
		SetNotes(notes).
		Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}

func (r *userSubscriptionRepository) ActivateWindows(ctx context.Context, id int64, start time.Time) error {
	client := clientFromContext(ctx, r.client)
	// 月窗口字段已移除（订阅最长 30 天，月限≡总额度）
	_, err := client.UserSubscription.UpdateOneID(id).
		SetDailyWindowStart(start).
		SetWeeklyWindowStart(start).
		Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}

func (r *userSubscriptionRepository) ResetUsageWindows(ctx context.Context, id int64, resetDaily, resetWeekly, resetMonthly bool, newWindowStart time.Time) error {
	client := clientFromContext(ctx, r.client)
	update := client.UserSubscription.UpdateOneID(id)
	if resetDaily {
		update.SetDailyUsageUsd(0).SetDailyWindowStart(newWindowStart)
	}
	if resetWeekly {
		update.SetWeeklyUsageUsd(0).SetWeeklyWindowStart(newWindowStart)
	}
	// Fork schema intentionally has no monthly usage window; monthly reset is a no-op.
	_ = resetMonthly
	_, err := update.Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}

// UserResetWeeklyLimit 原子 CAS 重置用户周限：
// 条件 id + user_id + weekly_limit_user_reset_at IS NULL（软删除拦截器自动过滤 deleted_at）。
// 设置 weekly_usage_usd=0、weekly_window_start、weekly_limit_user_reset_at。
// 返回影响行数；0 表示机会已用尽或并发未命中。
func (r *userSubscriptionRepository) UserResetWeeklyLimit(
	ctx context.Context,
	subscriptionID, userID int64,
	windowStart, resetAt time.Time,
) (int, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.UserSubscription.Update().
		Where(
			usersubscription.IDEQ(subscriptionID),
			usersubscription.UserIDEQ(userID),
			usersubscription.WeeklyLimitUserResetAtIsNil(),
		).
		SetWeeklyUsageUsd(0).
		SetWeeklyWindowStart(windowStart).
		SetWeeklyLimitUserResetAt(resetAt).
		Save(ctx)
	if err != nil {
		return 0, translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	return n, nil
}

func (r *userSubscriptionRepository) ResetDailyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	client := clientFromContext(ctx, r.client)
	query := client.UserSubscription.Update().Where(usersubscription.IDEQ(id))
	if expectedWindowStart == nil {
		query = query.Where(usersubscription.DailyWindowStartIsNil())
	} else {
		query = query.Where(usersubscription.DailyWindowStartEQ(*expectedWindowStart))
	}
	n, err := query.
		SetDailyUsageUsd(0).
		SetDailyWindowStart(newWindowStart).
		Save(ctx)
	return r.translateConditionalWindowReset(ctx, client, id, n, err)
}

func (r *userSubscriptionRepository) ResetWeeklyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	client := clientFromContext(ctx, r.client)
	query := client.UserSubscription.Update().Where(usersubscription.IDEQ(id))
	if expectedWindowStart == nil {
		query = query.Where(usersubscription.WeeklyWindowStartIsNil())
	} else {
		query = query.Where(usersubscription.WeeklyWindowStartEQ(*expectedWindowStart))
	}
	n, err := query.
		SetWeeklyUsageUsd(0).
		SetWeeklyWindowStart(newWindowStart).
		Save(ctx)
	return r.translateConditionalWindowReset(ctx, client, id, n, err)
}

func (r *userSubscriptionRepository) ResetMonthlyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	// 月窗口已从 schema 移除（订阅最长 30 天，月限≡总额度）。
	// 此方法保留接口兼容性，调用即为 no-op，旧调用方应迁移到 ResetDaily/Weekly。
	_ = ctx
	_ = id
	_ = expectedWindowStart
	_ = newWindowStart
	return nil
}

func (r *userSubscriptionRepository) translateConditionalWindowReset(ctx context.Context, client *dbent.Client, id int64, affected int, err error) error {
	if err != nil {
		return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	if affected > 0 {
		return nil
	}
	exists, err := client.UserSubscription.Query().Where(usersubscription.IDEQ(id)).Exist(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	if !exists {
		return service.ErrSubscriptionNotFound
	}
	return nil
}

// IncrementUsage 原子性地累加订阅用量。
// 限额检查已在请求前由 BillingCacheService.CheckBillingEligibility 完成，
// 此处仅负责记录实际消费，确保消费数据的完整性。
func (r *userSubscriptionRepository) IncrementUsage(ctx context.Context, id int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			quota_used_usd = us.quota_used_usd + $1,
			daily_usage_usd = us.daily_usage_usd + $1,
			weekly_usage_usd = us.weekly_usage_usd + $1,
			exhausted_at = CASE
				WHEN us.exhausted_at IS NULL
					AND us.quota_limit_usd > 0
					AND us.quota_used_usd < us.quota_limit_usd
					AND us.quota_used_usd + $1 >= us.quota_limit_usd - $3::numeric
				THEN NOW()
				ELSE us.exhausted_at
			END,
			updated_at = NOW()
		WHERE us.id = $2
			AND us.deleted_at IS NULL
			AND (
				us.group_id IS NULL
				OR EXISTS (
					SELECT 1
					FROM groups g
					WHERE g.id = us.group_id
						AND g.deleted_at IS NULL
				)
			)
	`

	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, updateSQL, costUSD, id, service.SubscriptionQuotaExhaustionEpsilonUSD)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected > 0 {
		return nil
	}

	// affected == 0：订阅不存在或已删除
	return service.ErrSubscriptionNotFound
}

func (r *userSubscriptionRepository) BatchUpdateExpiredStatus(ctx context.Context) (int64, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.UserSubscription.Update().
		Where(
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtLTE(time.Now()),
		).
		SetStatus(service.SubscriptionStatusExpired).
		Save(ctx)
	return int64(n), err
}

// Extra repository helpers (currently used only by integration tests).

func (r *userSubscriptionRepository) ListExpired(ctx context.Context) ([]service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	subs, err := client.UserSubscription.Query().
		Where(
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtLTE(time.Now()),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return userSubscriptionEntitiesToService(subs), nil
}

func (r *userSubscriptionRepository) CountByGroupID(ctx context.Context, groupID int64) (int64, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.UserSubscription.Query().Where(usersubscription.GroupIDEQ(groupID)).Count(ctx)
	return int64(count), err
}

func (r *userSubscriptionRepository) CountActiveByGroupID(ctx context.Context, groupID int64) (int64, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.UserSubscription.Query().
		Where(
			usersubscription.GroupIDEQ(groupID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		Count(ctx)
	return int64(count), err
}

func (r *userSubscriptionRepository) DeleteByGroupID(ctx context.Context, groupID int64) (int64, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.UserSubscription.Delete().Where(usersubscription.GroupIDEQ(groupID)).Exec(ctx)
	return int64(n), err
}

func userSubscriptionEntityToService(m *dbent.UserSubscription) *service.UserSubscription {
	if m == nil {
		return nil
	}
	out := &service.UserSubscription{
		ID:                    m.ID,
		UserID:                m.UserID,
		GroupID:               m.GroupID,
		PlanID:                m.PlanID,
		ScopeType:             m.ScopeType,
		ScopeConfig:           m.ScopeConfig,
		QuotaLimitUSD:         m.QuotaLimitUsd,
		QuotaUsedUSD:          m.QuotaUsedUsd,
		DailyLimitUSD:         m.DailyLimitUsd,
		WeeklyLimitUSD:        m.WeeklyLimitUsd,
		StartsAt:              m.StartsAt,
		ExpiresAt:             m.ExpiresAt,
		Status:                m.Status,
		ExhaustedAt:           m.ExhaustedAt,
		ExpiredCreditLoggedAt: m.ExpiredCreditLoggedAt,
		DailyWindowStart:      m.DailyWindowStart,
		WeeklyWindowStart:     m.WeeklyWindowStart,
		// MonthlyWindowStart / MonthlyUsageUSD 已从 schema 移除
		DailyUsageUSD:          m.DailyUsageUsd,
		WeeklyUsageUSD:         m.WeeklyUsageUsd,
		WeeklyLimitUserResetAt: m.WeeklyLimitUserResetAt,
		AssignedBy:             m.AssignedBy,
		AssignedAt:             m.AssignedAt,
		Notes:                  derefString(m.Notes),
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
	if m.Edges.User != nil {
		out.User = userEntityToService(m.Edges.User)
	}
	if m.Edges.Group != nil {
		out.Group = groupEntityToService(m.Edges.Group)
	}
	if m.Edges.AssignedByUser != nil {
		out.AssignedByUser = userEntityToService(m.Edges.AssignedByUser)
	}
	return out
}

func userSubscriptionEntitiesToService(models []*dbent.UserSubscription) []service.UserSubscription {
	out := make([]service.UserSubscription, 0, len(models))
	for i := range models {
		if s := userSubscriptionEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

func (r *userSubscriptionRepository) attachPlanSummary(ctx context.Context, sub *service.UserSubscription) error {
	if sub == nil {
		return nil
	}
	subs := []service.UserSubscription{*sub}
	if err := r.attachPlanSummaries(ctx, subs); err != nil {
		return err
	}
	*sub = subs[0]
	return nil
}

func (r *userSubscriptionRepository) attachPlanSummaries(ctx context.Context, subs []service.UserSubscription) error {
	planIDSet := make(map[int64]struct{})
	for i := range subs {
		if subs[i].PlanID != nil {
			planIDSet[*subs[i].PlanID] = struct{}{}
		}
	}
	if len(planIDSet) == 0 {
		return nil
	}

	planIDs := make([]int64, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}

	client := clientFromContext(ctx, r.client)
	plans, err := client.SubscriptionPlan.Query().
		Where(subscriptionplan.IDIn(planIDs...)).
		All(ctx)
	if err != nil {
		return err
	}

	type planSummary struct {
		name        string
		productName string
	}
	byID := make(map[int64]planSummary, len(plans))
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		byID[plan.ID] = planSummary{
			name:        strings.TrimSpace(plan.Name),
			productName: strings.TrimSpace(plan.ProductName),
		}
	}

	for i := range subs {
		if subs[i].PlanID == nil {
			continue
		}
		if summary, ok := byID[*subs[i].PlanID]; ok {
			subs[i].PlanName = summary.name
			subs[i].PlanProductName = summary.productName
		}
	}
	return nil
}

func applyUserSubscriptionEntityToService(dst *service.UserSubscription, src *dbent.UserSubscription) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

// ====================================================================================
// 订阅额度池扩展方法（实现 service.SubscriptionCreditExtension）
// ====================================================================================

// GetUsableCreditSubscription 返回用户当前最早创建的"可消费"订阅。
//
//	可消费 = status='active' AND exhausted_at IS NULL
//	       AND expires_at > now AND deleted_at IS NULL
//
// 不存在时返回 (nil, ErrSubscriptionNotFound)。
func (r *userSubscriptionRepository) GetUsableCreditSubscription(ctx context.Context, userID int64) (*service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExhaustedAtIsNil(),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		WithUser().
		WithGroup().
		Order(dbent.Asc(usersubscription.FieldCreatedAt), dbent.Asc(usersubscription.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
	}
	sub := userSubscriptionEntityToService(m)
	if err := r.attachPlanSummary(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// ListUsableCreditSubscriptions 返回用户当前全部"可消费"订阅，按创建时间从早到晚排序。
func (r *userSubscriptionRepository) ListUsableCreditSubscriptions(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	client := clientFromContext(ctx, r.client)
	subs, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExhaustedAtIsNil(),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		WithUser().
		WithGroup().
		Order(dbent.Asc(usersubscription.FieldCreatedAt), dbent.Asc(usersubscription.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := userSubscriptionEntitiesToService(subs)
	if err := r.attachPlanSummaries(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// HasUsableCreditSubscription 用户是否当前有可消费订阅。
func (r *userSubscriptionRepository) HasUsableCreditSubscription(ctx context.Context, userID int64) (bool, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.ExhaustedAtIsNil(),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateSubscriptionWithUsableGuard inserts a subscription after enforcing the
// single usable-subscription invariant when multiple purchases are disabled.
func (r *userSubscriptionRepository) CreateSubscriptionWithUsableGuard(ctx context.Context, sub *service.UserSubscription, allowMultiple bool) (*service.UserSubscription, error) {
	if sub == nil {
		return nil, service.ErrSubscriptionNilInput
	}
	if r.db == nil {
		if !allowMultiple {
			hasUsable, err := r.HasUsableCreditSubscription(ctx, sub.UserID)
			if err != nil {
				return nil, err
			}
			if hasUsable {
				return nil, service.ErrAlreadyHasUsableSubscription
			}
		}
		if err := r.Create(ctx, sub); err != nil {
			return nil, err
		}
		return sub, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if !allowMultiple {
		if err := r.LockUserForSubscriptionWrite(ctx, tx, sub.UserID); err != nil {
			return nil, err
		}
		hasUsable, err := hasUsableCreditSubscriptionSQL(ctx, tx, sub.UserID)
		if err != nil {
			return nil, err
		}
		if hasUsable {
			return nil, service.ErrAlreadyHasUsableSubscription
		}
	}
	created, err := r.InsertCreditSubscription(ctx, tx, sub)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return created, nil
}

func hasUsableCreditSubscriptionSQL(ctx context.Context, q service.SQLExecer, userID int64) (bool, error) {
	var hasUsable bool
	err := q.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM user_subscriptions
	WHERE user_id = $1
		AND status = $2
		AND exhausted_at IS NULL
		AND expires_at > NOW()
		AND deleted_at IS NULL
)
`, userID, service.SubscriptionStatusActive).Scan(&hasUsable)
	return hasUsable, err
}

// GetRenewalEligibility 用户是否允许购买新订阅。
//
// 判定顺序（与 plan 设计原则 #8 / #9 对齐）：
//  1. 当前 active 订阅存在 → 看是否仍可消费：
//     - 已耗尽（exhausted_at != nil）→ Allowed=true, Reason=exhausted
//     - 已过期（expires_at <= now）→ Allowed=true, Reason=expired
//     - 仍可消费 → Allowed=false, Reason=not_exhausted
//  2. 没有 active 订阅 → Allowed=true, Reason=no_subscription
//
// "active 订阅" 包含 exhausted（status=active + exhausted_at != nil）和 expired
// （status=active + expires_at <= now，过期任务尚未推进 status='expired'）两种状态。
func (r *userSubscriptionRepository) GetRenewalEligibility(ctx context.Context, userID int64) (service.RenewalEligibility, error) {
	client := clientFromContext(ctx, r.client)
	// 取用户最新的 active 订阅（包含已耗尽 / 已到期但未推进状态的）
	m, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
		).
		WithUser().
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.RenewalEligibility{
				Allowed: true,
				Reason:  service.RenewalReasonNoSubscription,
			}, nil
		}
		return service.RenewalEligibility{}, err
	}
	sub := userSubscriptionEntityToService(m)
	now := time.Now()
	// 优先判断耗尽，再判断过期（与 plan 一致：触顶才允许提前重订）。
	// 兜底：即使扣费时 exhausted_at 漏盖（浮点精度边界，见
	// SubscriptionQuotaExhaustionEpsilonUSD），只要 quota_used 已在容差内达到
	// quota_limit，也按耗尽放行续购，避免用户被永久拦截。
	quotaExhausted := sub.QuotaLimitUSD > 0 &&
		sub.QuotaUsedUSD >= sub.QuotaLimitUSD-service.SubscriptionQuotaExhaustionEpsilonUSD
	if sub.ExhaustedAt != nil || quotaExhausted {
		return service.RenewalEligibility{
			Allowed:      true,
			Reason:       service.RenewalReasonExhausted,
			Subscription: sub,
		}, nil
	}
	if !sub.ExpiresAt.After(now) {
		return service.RenewalEligibility{
			Allowed:      true,
			Reason:       service.RenewalReasonExpired,
			Subscription: sub,
		}, nil
	}
	return service.RenewalEligibility{
		Allowed:      false,
		Reason:       service.RenewalReasonNotExhausted,
		Subscription: sub,
	}, nil
}

// LockUserForSubscriptionWrite 在事务内对 users 行加 FOR UPDATE 行锁。
// 用于订阅履约 / 余额扣费等关键路径，防止并发购买写入两条订阅。
//
// 调用方必须持有 *sql.Tx；锁会随事务提交/回滚自动释放。
func (r *userSubscriptionRepository) LockUserForSubscriptionWrite(ctx context.Context, tx *sql.Tx, userID int64) error {
	if tx == nil {
		return errors.New("tx is required for LockUserForSubscriptionWrite")
	}
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUserNotFound
	}
	return err
}

// InsertCreditSubscription 在调用方事务内插入一条额度池订阅。
func (r *userSubscriptionRepository) InsertCreditSubscription(ctx context.Context, tx *sql.Tx, sub *service.UserSubscription) (*service.UserSubscription, error) {
	if tx == nil {
		return nil, errors.New("tx is required for InsertCreditSubscription")
	}
	if sub == nil {
		return nil, service.ErrSubscriptionNilInput
	}
	scopeConfig := sub.ScopeConfig
	if scopeConfig == nil {
		scopeConfig = map[string]any{}
	}
	scopeType := strings.TrimSpace(sub.ScopeType)
	if scopeType == "" {
		scopeType = service.SubscriptionScopeAllAvailableGroups
	}
	scopeJSON, err := json.Marshal(scopeConfig)
	if err != nil {
		return nil, fmt.Errorf("encode subscription scope_config: %w", err)
	}
	assignedAt := sub.AssignedAt
	if assignedAt.IsZero() {
		assignedAt = time.Now().UTC()
	}
	var (
		id        int64
		createdAt time.Time
		updatedAt time.Time
	)
	err = tx.QueryRowContext(ctx, `
INSERT INTO user_subscriptions (
    user_id, group_id, plan_id, scope_type, scope_config,
    quota_limit_usd, quota_used_usd, daily_limit_usd, weekly_limit_usd,
    starts_at, expires_at, status, exhausted_at, expired_credit_logged_at,
    daily_window_start, weekly_window_start, daily_usage_usd, weekly_usage_usd,
    assigned_by, assigned_at, notes, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17, $18,
    $19, $20, $21, NOW(), NOW()
)
RETURNING id, created_at, updated_at
`,
		sub.UserID,
		nullableInt64Arg(sub.GroupID),
		nullableInt64Arg(sub.PlanID),
		scopeType,
		scopeJSON,
		sub.QuotaLimitUSD,
		sub.QuotaUsedUSD,
		nullableArg(sub.DailyLimitUSD),
		nullableArg(sub.WeeklyLimitUSD),
		sub.StartsAt,
		sub.ExpiresAt,
		sub.Status,
		nullableTimeArg(sub.ExhaustedAt),
		nullableTimeArg(sub.ExpiredCreditLoggedAt),
		nullableTimeArg(sub.DailyWindowStart),
		nullableTimeArg(sub.WeeklyWindowStart),
		sub.DailyUsageUSD,
		sub.WeeklyUsageUSD,
		nullableInt64Arg(sub.AssignedBy),
		assignedAt,
		sub.Notes,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert credit subscription: %w", err)
	}
	cp := *sub
	cp.ID = id
	cp.CreatedAt = createdAt
	cp.UpdatedAt = updatedAt
	cp.AssignedAt = assignedAt
	cp.ScopeType = scopeType
	cp.ScopeConfig = scopeConfig
	return &cp, nil
}

type expiredCreditSubscriptionRow struct {
	ID                    int64
	UserID                int64
	GroupID               *int64
	QuotaLimitUSD         float64
	QuotaUsedUSD          float64
	ExpiresAt             time.Time
	ExpiredCreditLoggedAt *time.Time
}

// ExpireCreditSubscriptions 推进已到期 active 订阅，并记录剩余额度销毁流水与通知 outbox。
//
// 事务内先 SELECT ... FOR UPDATE SKIP LOCKED 锁定待处理订阅，再逐条更新状态、
// 写 expire ledger、写 subscription_notify outbox、标记 expired_credit_logged_at。
func (r *userSubscriptionRepository) ExpireCreditSubscriptions(ctx context.Context) (int64, error) {
	if r.db == nil {
		return 0, service.ErrSQLDBUnavailable
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.QueryContext(ctx, `
SELECT id, user_id, group_id, quota_limit_usd, quota_used_usd, expires_at, expired_credit_logged_at
FROM user_subscriptions
WHERE deleted_at IS NULL
  AND status = $1
  AND expires_at <= NOW()
ORDER BY expires_at ASC, id ASC
FOR UPDATE SKIP LOCKED
`, service.SubscriptionStatusActive)
	if err != nil {
		return 0, err
	}
	expiredRows, err := scanExpiredCreditSubscriptionRows(rows)
	if err != nil {
		return 0, err
	}

	var updated int64
	for _, row := range expiredRows {
		affected, err := expireCreditSubscriptionRow(ctx, tx, row.ID)
		if err != nil {
			return 0, err
		}
		if affected == 0 {
			continue
		}
		updated += affected

		remaining := math.Max(row.QuotaLimitUSD-row.QuotaUsedUSD, 0)
		if remaining <= 0 || row.ExpiredCreditLoggedAt != nil {
			continue
		}
		if err := r.recordExpiredCredit(ctx, tx, row, remaining); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return updated, nil
}

func scanExpiredCreditSubscriptionRows(rows *sql.Rows) (_ []expiredCreditSubscriptionRow, err error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	out := make([]expiredCreditSubscriptionRow, 0)
	for rows.Next() {
		var (
			row                   expiredCreditSubscriptionRow
			groupID               sql.NullInt64
			expiredCreditLoggedAt sql.NullTime
		)
		if err = rows.Scan(
			&row.ID,
			&row.UserID,
			&groupID,
			&row.QuotaLimitUSD,
			&row.QuotaUsedUSD,
			&row.ExpiresAt,
			&expiredCreditLoggedAt,
		); err != nil {
			return nil, err
		}
		if groupID.Valid {
			v := groupID.Int64
			row.GroupID = &v
		}
		if expiredCreditLoggedAt.Valid {
			v := expiredCreditLoggedAt.Time
			row.ExpiredCreditLoggedAt = &v
		}
		out = append(out, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func expireCreditSubscriptionRow(ctx context.Context, tx *sql.Tx, id int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `
UPDATE user_subscriptions
SET status = $1,
    updated_at = NOW()
WHERE id = $2
  AND status = $3
`, service.SubscriptionStatusExpired, id, service.SubscriptionStatusActive)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *userSubscriptionRepository) recordExpiredCredit(ctx context.Context, tx *sql.Tx, row expiredCreditSubscriptionRow, remaining float64) error {
	eventKey := fmt.Sprintf("expire:%d:%s", row.ID, row.ExpiresAt.UTC().Format(time.RFC3339))
	created, err := NewSubscriptionCreditLedgerRepository(r.db).CreateLimitReachedEvent(ctx, tx, &service.SubscriptionCreditLedgerEntry{
		UserID:            row.UserID,
		SubscriptionID:    row.ID,
		GroupID:           row.GroupID,
		Type:              service.SubscriptionCreditLedgerExpire,
		DeltaUSD:          -remaining,
		RemainingAfterUSD: 0,
		Reason:            "subscription expired",
		EventKey:          &eventKey,
		Metadata: map[string]any{
			"quota_limit_usd": row.QuotaLimitUSD,
			"quota_used_usd":  row.QuotaUsedUSD,
			"remaining_usd":   remaining,
			"expires_at":      row.ExpiresAt.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return err
	}
	if created {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventSubscriptionNotify, nil, nil, map[string]any{
			"user_id":         row.UserID,
			"subscription_id": row.ID,
			"kind":            "expired",
		}); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
UPDATE user_subscriptions
SET expired_credit_logged_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND expired_credit_logged_at IS NULL
`, row.ID)
	return err
}

func nullableTimeArg(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}

// MarkExpiredCreditLogged 标记过期销毁 ledger 已写入。
func (r *userSubscriptionRepository) MarkExpiredCreditLogged(ctx context.Context, id int64, loggedAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserSubscription.UpdateOneID(id).
		SetExpiredCreditLoggedAt(loggedAt).
		Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionNotFound, nil)
}
