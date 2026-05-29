package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type subscriptionWasteStatsRepository struct {
	db *sql.DB
}

func NewSubscriptionWasteStatsRepository(db *sql.DB) service.SubscriptionWasteStatsRepository {
	return &subscriptionWasteStatsRepository{db: db}
}

func (r *subscriptionWasteStatsRepository) GetWasteStats(ctx context.Context, query service.WasteStatsQuery) (service.WasteStatsResult, error) {
	var result service.WasteStatsResult
	if query.Projection == service.WasteStatsProjectionByPlan {
		byPlan, err := r.queryByPlan(ctx, query)
		if err != nil {
			return result, err
		}
		result.ByPlan = byPlan
		return result, nil
	}
	if query.Projection == service.WasteStatsProjectionTimeSeries {
		timeSeries, err := r.queryTimeSeries(ctx, query)
		if err != nil {
			return result, err
		}
		result.TimeSeries = timeSeries
		return result, nil
	}
	summary, err := r.querySummary(ctx, query, "")
	if err != nil {
		return result, err
	}
	result = summary

	byPlan, err := r.queryByPlan(ctx, query)
	if err != nil {
		return result, err
	}
	result.ByPlan = byPlan

	timeSeries, err := r.queryTimeSeries(ctx, query)
	if err != nil {
		return result, err
	}
	result.TimeSeries = timeSeries
	return result, nil
}

func (r *subscriptionWasteStatsRepository) querySummary(ctx context.Context, query service.WasteStatsQuery, groupBy string) (service.WasteStatsResult, error) {
	where, args := wasteStatsWhere(query)
	windowResetPredicate := wasteStatsWindowResetPredicate(query.Window)
	totalWastePredicate := wasteStatsTotalWastePredicate(query.Window)
	sqlText := fmt.Sprintf(`
SELECT
	COALESCE(SUM(CASE WHEN l.type = 'purchase' THEN l.delta_usd ELSE 0 END), 0)::double precision AS purchased_usd,
	COALESCE(SUM(CASE WHEN l.type = 'consume' THEN ABS(l.delta_usd) ELSE 0 END), 0)::double precision AS consumed_usd,
	COALESCE(SUM(CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE 0 END), 0)::double precision AS expired_wasted_usd,
	COALESCE(SUM(CASE WHEN %s THEN %s ELSE 0 END), 0)::double precision AS window_wasted_usd,
	COALESCE(SUM(CASE WHEN %s THEN
		CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE %s END
	ELSE 0 END), 0)::double precision AS total_wasted_usd,
	COALESCE(AVG(CASE WHEN %s THEN %s ELSE NULL END), 0)::double precision AS average_waste_ratio,
	COUNT(*) FILTER (WHERE %s) AS reset_count,
	COUNT(DISTINCT CASE WHEN l.type = 'purchase' THEN l.subscription_id ELSE NULL END) AS purchase_count,
	COUNT(*) FILTER (WHERE l.type = 'window_reset' AND l.metadata->>'window' = 'daily') AS daily_reset_count,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'daily' THEN %s ELSE NULL END), 0)::double precision AS daily_average_waste_ratio,
	COALESCE(SUM(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'daily' THEN %s ELSE 0 END), 0)::double precision AS daily_total_wasted_usd,
	COUNT(*) FILTER (WHERE l.type = 'window_reset' AND l.metadata->>'window' = 'weekly') AS weekly_reset_count,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'weekly' THEN %s ELSE NULL END), 0)::double precision AS weekly_average_waste_ratio,
	COALESCE(SUM(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'weekly' THEN %s ELSE 0 END), 0)::double precision AS weekly_total_wasted_usd
FROM subscription_credit_ledger l
JOIN user_subscriptions us ON us.id = l.subscription_id
LEFT JOIN subscription_plans sp ON sp.id = us.plan_id
WHERE %s
%s
`, windowResetPredicate, metadataNumber("wasted_usd"), totalWastePredicate, metadataNumber("wasted_usd"), windowResetPredicate, metadataNumber("wasted_ratio"), windowResetPredicate, metadataNumber("wasted_ratio"), metadataNumber("wasted_usd"), metadataNumber("wasted_ratio"), metadataNumber("wasted_usd"), where, groupBy)

	row := r.db.QueryRowContext(ctx, sqlText, args...)
	var out service.WasteStatsResult
	if err := row.Scan(
		&out.PurchasedUSD,
		&out.ConsumedUSD,
		&out.ExpiredWastedUSD,
		&out.WindowWastedUSD,
		&out.TotalWastedUSD,
		&out.AverageWasteRatio,
		&out.ResetCount,
		&out.TotalSubscriptionsPurchased,
		&out.DailyResetCount,
		&out.DailyAverageWasteRatio,
		&out.DailyTotalWastedUSD,
		&out.WeeklyResetCount,
		&out.WeeklyAverageWasteRatio,
		&out.WeeklyTotalWastedUSD,
	); err != nil {
		return service.WasteStatsResult{}, err
	}
	out.TotalWastedRatio = safeWasteRatio(out.TotalWastedUSD, out.PurchasedUSD)
	out.WasteRatio = out.TotalWastedRatio
	return out, nil
}

func (r *subscriptionWasteStatsRepository) queryByPlan(ctx context.Context, query service.WasteStatsQuery) ([]service.PlanWasteBucket, error) {
	where, args := wasteStatsWhere(query)
	windowResetPredicate := wasteStatsWindowResetPredicate(query.Window)
	totalWastePredicate := wasteStatsTotalWastePredicate(query.Window)
	sqlText := fmt.Sprintf(`
SELECT
	us.plan_id,
	COALESCE(sp.name, ''),
	COALESCE(SUM(CASE WHEN l.type = 'purchase' THEN l.delta_usd ELSE 0 END), 0)::double precision AS purchased_usd,
	COALESCE(SUM(CASE WHEN l.type = 'consume' THEN ABS(l.delta_usd) ELSE 0 END), 0)::double precision AS consumed_usd,
	COALESCE(SUM(CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE 0 END), 0)::double precision AS expired_wasted_usd,
	COALESCE(SUM(CASE WHEN %s THEN %s ELSE 0 END), 0)::double precision AS window_wasted_usd,
	COALESCE(SUM(CASE WHEN %s THEN
		CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE %s END
	ELSE 0 END), 0)::double precision AS total_wasted_usd,
	COALESCE(AVG(CASE WHEN %s THEN %s ELSE NULL END), 0)::double precision AS average_waste_ratio,
	COUNT(*) FILTER (WHERE %s) AS reset_count,
	COUNT(DISTINCT CASE WHEN l.type = 'purchase' THEN l.subscription_id ELSE NULL END) AS purchase_count,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'daily' THEN %s ELSE NULL END), 0)::double precision AS average_daily_waste_ratio,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'weekly' THEN %s ELSE NULL END), 0)::double precision AS average_weekly_waste_ratio
FROM subscription_credit_ledger l
JOIN user_subscriptions us ON us.id = l.subscription_id
LEFT JOIN subscription_plans sp ON sp.id = us.plan_id
WHERE %s
GROUP BY us.plan_id, sp.name
ORDER BY total_wasted_usd DESC, purchased_usd DESC, us.plan_id NULLS LAST
`, windowResetPredicate, metadataNumber("wasted_usd"), totalWastePredicate, metadataNumber("wasted_usd"), windowResetPredicate, metadataNumber("wasted_ratio"), windowResetPredicate, metadataNumber("wasted_ratio"), metadataNumber("wasted_ratio"), where)
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.PlanWasteBucket
	for rows.Next() {
		var (
			b      service.PlanWasteBucket
			planID sql.NullInt64
		)
		if err := rows.Scan(
			&planID,
			&b.PlanName,
			&b.PurchasedUSD,
			&b.ConsumedUSD,
			&b.ExpiredWastedUSD,
			&b.WindowWastedUSD,
			&b.TotalWastedUSD,
			&b.AverageWasteRatio,
			&b.ResetCount,
			&b.PurchaseCount,
			&b.AverageDailyWasteRatio,
			&b.AverageWeeklyWasteRatio,
		); err != nil {
			return nil, err
		}
		if planID.Valid {
			v := planID.Int64
			b.PlanID = &v
		}
		b.TotalWastedRatio = safeWasteRatio(b.TotalWastedUSD, b.PurchasedUSD)
		b.WasteRatio = b.TotalWastedRatio
		b.TotalQuotaWastedRatio = safeWasteRatio(b.ExpiredWastedUSD, b.PurchasedUSD)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *subscriptionWasteStatsRepository) queryTimeSeries(ctx context.Context, query service.WasteStatsQuery) ([]service.WasteTimeBucket, error) {
	where, args := wasteStatsWhere(query)
	grain, interval := wasteStatsBucketGrain(query.Window)
	totalWastePredicate := wasteStatsTotalWastePredicate(query.Window)
	sqlText := fmt.Sprintf(`
SELECT
	date_trunc('%s', l.created_at)::timestamptz AS bucket_start,
	(date_trunc('%s', l.created_at) + interval '%s')::timestamptz AS bucket_end,
	COALESCE(SUM(CASE WHEN %s THEN
		CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE %s END
	ELSE 0 END), 0)::double precision AS wasted_usd,
	COALESCE(SUM(CASE WHEN l.type = 'expire' THEN ABS(l.delta_usd) ELSE 0 END), 0)::double precision AS expired_wasted_usd,
	COALESCE(SUM(CASE WHEN l.type = 'window_reset' THEN %s ELSE 0 END), 0)::double precision AS window_wasted_usd,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' THEN %s ELSE NULL END), 0)::double precision AS average_waste_ratio,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'daily' THEN %s ELSE NULL END), 0)::double precision AS daily_average_waste_ratio,
	COALESCE(AVG(CASE WHEN l.type = 'window_reset' AND l.metadata->>'window' = 'weekly' THEN %s ELSE NULL END), 0)::double precision AS weekly_average_waste_ratio,
	COUNT(*) FILTER (WHERE l.type = 'window_reset') AS reset_count,
	COALESCE(SUM(CASE WHEN l.type = 'purchase' THEN l.delta_usd ELSE 0 END), 0)::double precision AS purchased_usd
FROM subscription_credit_ledger l
JOIN user_subscriptions us ON us.id = l.subscription_id
LEFT JOIN subscription_plans sp ON sp.id = us.plan_id
WHERE %s
GROUP BY bucket_start, bucket_end
ORDER BY bucket_start
`, grain, grain, interval, totalWastePredicate, metadataNumber("wasted_usd"), metadataNumber("wasted_usd"), metadataNumber("wasted_ratio"), metadataNumber("wasted_ratio"), metadataNumber("wasted_ratio"), where)
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.WasteTimeBucket
	for rows.Next() {
		var purchased float64
		var b service.WasteTimeBucket
		if err := rows.Scan(&b.BucketStart, &b.BucketEnd, &b.WastedUSD, &b.ExpiredWastedUSD, &b.WindowWastedUSD, &b.AverageWasteRatio, &b.DailyAverageWasteRatio, &b.WeeklyAverageWasteRatio, &b.ResetCount, &purchased); err != nil {
			return nil, err
		}
		b.Window = query.Window
		b.TotalWastedUSD = b.WastedUSD
		b.TotalWastedRatio = safeWasteRatio(b.WastedUSD, purchased)
		out = append(out, b)
	}
	return out, rows.Err()
}

func wasteStatsWhere(query service.WasteStatsQuery) (string, []any) {
	args := []any{query.StartTime, query.EndTime, service.SubscriptionStatusRevoked}
	parts := []string{"l.created_at >= $1", "l.created_at < $2", "us.deleted_at IS NULL", "us.status <> $3"}
	if query.PlanID != nil {
		args = append(args, *query.PlanID)
		parts = append(parts, fmt.Sprintf("us.plan_id = $%d", len(args)))
	}
	if query.UserID != nil {
		args = append(args, *query.UserID)
		parts = append(parts, fmt.Sprintf("l.user_id = $%d", len(args)))
	}
	return strings.Join(parts, " AND "), args
}

func wasteStatsWindowResetPredicate(window string) string {
	switch window {
	case service.WasteStatsWindowDaily:
		return "l.type = 'window_reset' AND l.metadata->>'window' = 'daily'"
	case service.WasteStatsWindowWeekly:
		return "l.type = 'window_reset' AND l.metadata->>'window' = 'weekly'"
	case service.WasteStatsWindowTotal:
		return "FALSE"
	default:
		return "l.type = 'window_reset'"
	}
}

func wasteStatsTotalWastePredicate(window string) string {
	switch window {
	case service.WasteStatsWindowDaily:
		return "l.type = 'window_reset' AND l.metadata->>'window' = 'daily'"
	case service.WasteStatsWindowWeekly:
		return "l.type = 'window_reset' AND l.metadata->>'window' = 'weekly'"
	case service.WasteStatsWindowTotal:
		return "l.type = 'expire'"
	default:
		return "l.type IN ('expire', 'window_reset')"
	}
}

func wasteStatsBucketGrain(window string) (string, string) {
	if window == service.WasteStatsWindowDaily {
		return "day", "1 day"
	}
	return "week", "1 week"
}

func metadataNumber(key string) string {
	return fmt.Sprintf("CASE WHEN jsonb_typeof(l.metadata->'%[1]s') = 'number' THEN (l.metadata->>'%[1]s')::double precision ELSE 0 END", key)
}

func safeWasteRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

var _ service.SubscriptionWasteStatsRepository = (*subscriptionWasteStatsRepository)(nil)
var _ time.Time
