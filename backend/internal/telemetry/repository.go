package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
)

type sqlRepository struct {
	db *sql.DB
}

func newSQLRepository(db *sql.DB) *sqlRepository {
	return &sqlRepository{db: db}
}

func (r *sqlRepository) InsertEvent(ctx context.Context, event Event) error {
	if r == nil || r.db == nil {
		return errors.New("telemetry repository is not configured")
	}
	var userID any
	if event.UserID != nil && *event.UserID > 0 {
		userID = *event.UserID
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO telemetry_events (
	event, occurred_at, client_source, route, auth_method, platform, destination,
	error_code, attribution_id,
	first_touch_source, first_touch_medium, first_touch_campaign,
	last_touch_source, last_touch_medium, last_touch_campaign,
	user_id, dedup_key, ingest_source
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		event.Event,
		event.OccurredAt.UTC(),
		event.ClientSource,
		event.Route,
		event.AuthMethod,
		event.Platform,
		event.Destination,
		event.ErrorCode,
		event.AttributionID,
		event.FirstTouchSource,
		event.FirstTouchMedium,
		event.FirstTouchCampaign,
		event.LastTouchSource,
		event.LastTouchMedium,
		event.LastTouchCampaign,
		userID,
		event.DedupKey,
		event.IngestSource,
	)
	if isUniqueViolation(err) {
		return ErrDuplicateEvent
	}
	return err
}

func (r *sqlRepository) FindSuccessMatch(ctx context.Context, event Event) (*Event, error) {
	if r == nil || r.db == nil || !isSuccessEvent(event.Event) {
		return nil, nil
	}
	attributionID := event.AttributionID
	var userID any
	if event.UserID != nil && *event.UserID > 0 {
		userID = *event.UserID
	}
	if attributionID == "" && userID == nil {
		return nil, nil
	}
	windowStart, windowed := successMatchWindowStart(event)
	var windowStartArg any
	if windowed {
		windowStartArg = windowStart.UTC()
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, event, occurred_at, client_source, route, auth_method, platform, destination,
	error_code, attribution_id,
	first_touch_source, first_touch_medium, first_touch_campaign,
	last_touch_source, last_touch_medium, last_touch_campaign,
	user_id, dedup_key, ingest_source
FROM telemetry_events
WHERE event = $1
	AND (
		($2 <> '' AND attribution_id = $2)
		OR ($3::bigint IS NOT NULL AND user_id = $3)
	)
	AND ($4::timestamptz IS NULL OR occurred_at >= $4)
ORDER BY id ASC
LIMIT 1`,
		event.Event, attributionID, userID, windowStartArg,
	)
	matched, err := scanTelemetryEvent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return matched, nil
}

func (r *sqlRepository) PatchEventObservability(ctx context.Context, id int64, incoming Event) error {
	if r == nil || r.db == nil || id <= 0 {
		return nil
	}
	var userID any
	if incoming.UserID != nil && *incoming.UserID > 0 {
		userID = *incoming.UserID
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE telemetry_events SET
	client_source = CASE WHEN client_source IN ('', 'unknown') AND $2 NOT IN ('', 'unknown') THEN $2 ELSE client_source END,
	route = CASE WHEN route = '' THEN $3 ELSE route END,
	auth_method = CASE WHEN auth_method = '' THEN $4 ELSE auth_method END,
	platform = CASE WHEN platform = '' THEN $5 ELSE platform END,
	destination = CASE WHEN destination = '' THEN $6 ELSE destination END,
	error_code = CASE WHEN error_code = '' THEN $7 ELSE error_code END,
	attribution_id = CASE WHEN attribution_id = '' THEN $8 ELSE attribution_id END,
	first_touch_source = CASE WHEN first_touch_source = '' THEN $9 ELSE first_touch_source END,
	first_touch_medium = CASE WHEN first_touch_medium = '' THEN $10 ELSE first_touch_medium END,
	first_touch_campaign = CASE WHEN first_touch_campaign = '' THEN $11 ELSE first_touch_campaign END,
	last_touch_source = CASE WHEN last_touch_source = '' THEN $12 ELSE last_touch_source END,
	last_touch_medium = CASE WHEN last_touch_medium = '' THEN $13 ELSE last_touch_medium END,
	last_touch_campaign = CASE WHEN last_touch_campaign = '' THEN $14 ELSE last_touch_campaign END,
	user_id = COALESCE(user_id, $15)
WHERE id = $1`,
		id,
		incoming.ClientSource,
		incoming.Route,
		incoming.AuthMethod,
		incoming.Platform,
		incoming.Destination,
		incoming.ErrorCode,
		incoming.AttributionID,
		incoming.FirstTouchSource,
		incoming.FirstTouchMedium,
		incoming.FirstTouchCampaign,
		incoming.LastTouchSource,
		incoming.LastTouchMedium,
		incoming.LastTouchCampaign,
		userID,
	)
	return err
}

func scanTelemetryEvent(row rowScanner) (*Event, error) {
	var (
		event  Event
		userID sql.NullInt64
	)
	err := row.Scan(
		&event.ID,
		&event.Event,
		&event.OccurredAt,
		&event.ClientSource,
		&event.Route,
		&event.AuthMethod,
		&event.Platform,
		&event.Destination,
		&event.ErrorCode,
		&event.AttributionID,
		&event.FirstTouchSource,
		&event.FirstTouchMedium,
		&event.FirstTouchCampaign,
		&event.LastTouchSource,
		&event.LastTouchMedium,
		&event.LastTouchCampaign,
		&userID,
		&event.DedupKey,
		&event.IngestSource,
	)
	if err != nil {
		return nil, err
	}
	if userID.Valid && userID.Int64 > 0 {
		id := userID.Int64
		event.UserID = &id
	}
	return &event, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *sqlRepository) GetAttribution(ctx context.Context, userID int64) (*AccountAttribution, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, nil
	}
	var (
		attr         AccountAttribution
		firstTouchAt sql.NullTime
		lastTouchAt  sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, `
SELECT user_id, first_touch_source, first_touch_medium, first_touch_campaign,
	first_attribution_id, last_touch_source, last_touch_medium, last_touch_campaign,
	last_attribution_id, first_touch_at, last_touch_at
FROM user_first_party_attribution
WHERE user_id = $1`, userID).Scan(
		&attr.UserID,
		&attr.FirstTouchSource,
		&attr.FirstTouchMedium,
		&attr.FirstTouchCampaign,
		&attr.FirstAttributionID,
		&attr.LastTouchSource,
		&attr.LastTouchMedium,
		&attr.LastTouchCampaign,
		&attr.LastAttributionID,
		&firstTouchAt,
		&lastTouchAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if firstTouchAt.Valid {
		attr.FirstTouchAt = firstTouchAt.Time
	}
	if lastTouchAt.Valid {
		attr.LastTouchAt = lastTouchAt.Time
	}
	return &attr, nil
}

func (r *sqlRepository) UpsertAttribution(ctx context.Context, attr AccountAttribution) error {
	if r == nil || r.db == nil || attr.UserID <= 0 {
		return nil
	}
	var firstTouchAt any
	if !attr.FirstTouchAt.IsZero() {
		firstTouchAt = attr.FirstTouchAt.UTC()
	}
	var lastTouchAt any
	if !attr.LastTouchAt.IsZero() {
		lastTouchAt = attr.LastTouchAt.UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_first_party_attribution (
	user_id,
	first_touch_source, first_touch_medium, first_touch_campaign, first_attribution_id,
	last_touch_source, last_touch_medium, last_touch_campaign, last_attribution_id,
	first_touch_at, last_touch_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (user_id) DO UPDATE SET
	first_touch_source = CASE WHEN user_first_party_attribution.first_touch_source = '' THEN EXCLUDED.first_touch_source ELSE user_first_party_attribution.first_touch_source END,
	first_touch_medium = CASE WHEN user_first_party_attribution.first_touch_medium = '' THEN EXCLUDED.first_touch_medium ELSE user_first_party_attribution.first_touch_medium END,
	first_touch_campaign = CASE WHEN user_first_party_attribution.first_touch_campaign = '' THEN EXCLUDED.first_touch_campaign ELSE user_first_party_attribution.first_touch_campaign END,
	first_attribution_id = CASE WHEN user_first_party_attribution.first_attribution_id = '' THEN EXCLUDED.first_attribution_id ELSE user_first_party_attribution.first_attribution_id END,
	first_touch_at = COALESCE(user_first_party_attribution.first_touch_at, EXCLUDED.first_touch_at),
	last_touch_source = CASE WHEN EXCLUDED.last_touch_source = '' THEN user_first_party_attribution.last_touch_source ELSE EXCLUDED.last_touch_source END,
	last_touch_medium = CASE WHEN EXCLUDED.last_touch_medium = '' THEN user_first_party_attribution.last_touch_medium ELSE EXCLUDED.last_touch_medium END,
	last_touch_campaign = CASE WHEN EXCLUDED.last_touch_campaign = '' THEN user_first_party_attribution.last_touch_campaign ELSE EXCLUDED.last_touch_campaign END,
	last_attribution_id = CASE WHEN EXCLUDED.last_attribution_id = '' THEN user_first_party_attribution.last_attribution_id ELSE EXCLUDED.last_attribution_id END,
	last_touch_at = COALESCE(EXCLUDED.last_touch_at, user_first_party_attribution.last_touch_at)`,
		attr.UserID,
		attr.FirstTouchSource,
		attr.FirstTouchMedium,
		attr.FirstTouchCampaign,
		attr.FirstAttributionID,
		attr.LastTouchSource,
		attr.LastTouchMedium,
		attr.LastTouchCampaign,
		attr.LastAttributionID,
		firstTouchAt,
		lastTouchAt,
	)
	return err
}

func (r *sqlRepository) Aggregate(ctx context.Context, query StatsQuery) ([]StatsRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("telemetry repository is not configured")
	}
	from := time.UnixMilli(query.From).UTC()
	to := time.UnixMilli(query.To).UTC()
	rows, err := r.db.QueryContext(ctx, `
SELECT event,
	COUNT(*)::bigint,
	COUNT(DISTINCT CASE WHEN attribution_id <> '' THEN attribution_id END)::bigint
FROM telemetry_events
WHERE occurred_at >= $1 AND occurred_at <= $2
	AND ($3 = '' OR client_source = $3)
	AND ($4 = '' OR first_touch_campaign = $4)
	AND ($5 = '' OR event = $5)
GROUP BY event
ORDER BY event`,
		from, to, query.ClientSource, query.Campaign, query.Event,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]StatsRow, 0)
	for rows.Next() {
		var row StatsRow
		if err := rows.Scan(&row.Event, &row.EventCount, &row.UniqueAttributionIDs); err != nil {
			return nil, err
		}
		row.Measure = MeasureEventCountAndUniqueAnonymousID
		out = append(out, row)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
