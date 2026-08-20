package checkin

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

func adminRecordWhere(filter AdminRecordFilter) (string, []any) {
	where := []string{"TRUE"}
	args := make([]any, 0, 8)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.UserID > 0 {
		where = append(where, "user_id = "+addArgument(filter.UserID))
	}
	if filter.Search != "" {
		placeholder := addArgument("%" + filter.Search + "%")
		where = append(where, "(user_email ILIKE "+placeholder+" OR username ILIKE "+placeholder+")")
	}
	if filter.BusinessDate != nil {
		where = append(where, "business_date = "+addArgument(*filter.BusinessDate))
	} else {
		if filter.BusinessDateFrom != nil {
			where = append(where, "business_date >= "+addArgument(*filter.BusinessDateFrom))
		}
		if filter.BusinessDateTo != nil {
			where = append(where, "business_date <= "+addArgument(*filter.BusinessDateTo))
		}
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArgument(filter.Status))
	}
	return strings.Join(where, " AND "), args
}

func (r *sqlRepository) ListAdminRecords(ctx context.Context, rawFilter AdminRecordFilter) ([]Record, int64, error) {
	filter := normalizeAdminFilter(rawFilter)
	whereSQL, args := adminRecordWhere(filter)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM daily_checkin_records WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin check-in records: %w", err)
	}

	sortColumns := map[string]string{
		"business_date": "business_date",
		"checked_at":    "checked_at",
		"streak_days":   "streak_days",
		"actual_reward": "actual_reward",
		"balance_after": "balance_after",
	}
	sortColumn := sortColumns[filter.SortBy]
	if sortColumn == "" {
		sortColumn = "checked_at"
	}
	sortOrder := "DESC"
	if strings.EqualFold(filter.SortOrder, "asc") {
		sortOrder = "ASC"
	}
	limitPlaceholder := addArgument(filter.PageSize)
	offsetPlaceholder := addArgument((filter.Page - 1) * filter.PageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+recordSelectColumns+`
		FROM daily_checkin_records
		WHERE `+whereSQL+`
		ORDER BY `+sortColumn+` `+sortOrder+`, id `+sortOrder+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin check-in records: %w", err)
	}
	defer func() { _ = rows.Close() }()

	records := make([]Record, 0, filter.PageSize)
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan admin check-in record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate admin check-in records: %w", err)
	}
	return records, total, nil
}

func (r *sqlRepository) AdminStats(ctx context.Context, rawFilter AdminRecordFilter) (AdminStats, error) {
	filter := normalizeAdminFilter(rawFilter)
	whereSQL, args := adminRecordWhere(filter)
	row := r.db.QueryRowContext(ctx, `
		WITH filtered AS (
			SELECT user_id, actual_reward, status
			FROM daily_checkin_records
			WHERE `+whereSQL+`
		)
		SELECT
			COUNT(*)::bigint,
			COUNT(DISTINCT user_id)::bigint,
			COALESCE(SUM(actual_reward) FILTER (WHERE status = 'awarded'), 0)::text,
			COALESCE(AVG(actual_reward) FILTER (WHERE status = 'awarded'), 0)::text,
			COALESCE((
				SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY actual_reward)
				FROM filtered
				WHERE status = 'awarded'
			), 0)::text,
			COALESCE((
				SELECT percentile_cont(0.9) WITHIN GROUP (ORDER BY actual_reward)
				FROM filtered
				WHERE status = 'awarded'
			), 0)::text,
			COALESCE(MAX(actual_reward) FILTER (WHERE status = 'awarded'), 0)::text
		FROM filtered`, args...)

	var (
		stats                           AdminStats
		totalAmount, avgAmount          string
		p50Amount, p90Amount, maxAmount string
	)
	if err := row.Scan(
		&stats.CheckInCount,
		&stats.UniqueUsers,
		&totalAmount,
		&avgAmount,
		&p50Amount,
		&p90Amount,
		&maxAmount,
	); err != nil {
		return AdminStats{}, fmt.Errorf("admin check-in stats: %w", err)
	}
	parsed, err := parseStatsAmounts(totalAmount, avgAmount, p50Amount, p90Amount, maxAmount)
	if err != nil {
		return AdminStats{}, err
	}
	stats.TotalAmount, stats.AvgAmount, stats.P50Amount, stats.P90Amount, stats.MaxAmount = parsed[0], parsed[1], parsed[2], parsed[3], parsed[4]
	return stats, nil
}

func parseStatsAmounts(values ...string) ([]decimal.Decimal, error) {
	out := make([]decimal.Decimal, 0, len(values))
	for _, raw := range values {
		amount, err := decimal.NewFromString(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("parse admin check-in stats amount %q: %w", raw, err)
		}
		out = append(out, amount)
	}
	return out, nil
}
