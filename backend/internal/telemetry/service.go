package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{repo: repo, now: now}
}

func (s *Service) Ingest(ctx context.Context, req ingestRequest, userID int64) error {
	eventName := req.Event
	if _, ok := allowedEvents[eventName]; !ok {
		return ErrUnknownEvent
	}
	now := s.now().UTC()
	props := sanitizeProps(req.Props)
	occurred := normalizeOccurredAt(req.TS, now)

	event := Event{
		Event:              eventName,
		OccurredAt:         occurred,
		ClientSource:       props.ClientSource,
		Route:              props.Route,
		AuthMethod:         props.AuthMethod,
		Platform:           props.Platform,
		Destination:        props.Destination,
		ErrorCode:          props.ErrorCode,
		AttributionID:      props.AttributionID,
		FirstTouchSource:   props.FirstTouchSource,
		FirstTouchMedium:   props.FirstTouchMedium,
		FirstTouchCampaign: props.FirstTouchCampaign,
		LastTouchSource:    props.LastTouchSource,
		LastTouchMedium:    props.LastTouchMedium,
		LastTouchCampaign:  props.LastTouchCampaign,
		IngestSource:       IngestSourceClient,
	}

	if userID > 0 {
		id := userID
		event.UserID = &id
		if _, inherit := inheritFirstTouchEvents[eventName]; inherit && firstTouchEmpty(props) {
			if attr, err := s.repo.GetAttribution(ctx, userID); err != nil {
				slog.Warn("telemetry attribution lookup failed", "error", err)
			} else if attr != nil {
				event.FirstTouchSource = attr.FirstTouchSource
				event.FirstTouchMedium = attr.FirstTouchMedium
				event.FirstTouchCampaign = attr.FirstTouchCampaign
			}
		}
		fillLastTouchFromFirst(&event)
		if err := s.repo.UpsertAttribution(ctx, attributionFromEvent(event, userID)); err != nil {
			slog.Warn("telemetry attribution upsert failed", "error", err)
		}
	} else {
		fillLastTouchFromFirst(&event)
	}

	return s.persistEvent(ctx, event)
}

func (s *Service) Stats(ctx context.Context, query StatsQuery) (StatsResult, error) {
	rows, err := s.repo.Aggregate(ctx, query)
	if err != nil {
		return StatsResult{}, err
	}
	if rows == nil {
		rows = []StatsRow{}
	}
	if query.Event != "" && len(rows) == 0 {
		rows = []StatsRow{{
			Event:                query.Event,
			EventCount:           0,
			UniqueAttributionIDs: 0,
			Measure:              MeasureEventCountAndUniqueAnonymousID,
		}}
	}
	filters := map[string]string{}
	if query.ClientSource != "" {
		filters["client_source"] = query.ClientSource
	}
	if query.Campaign != "" {
		filters["campaign"] = query.Campaign
	}
	if len(filters) == 0 {
		filters = nil
	}
	return StatsResult{
		Authority: AuthorityFirstPartyIngest,
		Range:     StatsRange{From: query.From, To: query.To},
		Filters:   filters,
		Rows:      rows,
	}, nil
}

func (s *Service) RecordServerAuthEvent(ctx context.Context, userID int64, eventName string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil
	}
	if _, ok := allowedEvents[eventName]; !ok {
		return nil
	}
	now := s.now().UTC()
	event := Event{
		Event:        eventName,
		OccurredAt:   now,
		ClientSource: ClientSourceUnknown,
		IngestSource: IngestSourceServer,
		UserID:       &userID,
	}
	if attr, err := s.repo.GetAttribution(ctx, userID); err != nil {
		slog.Warn("telemetry server attribution lookup failed", "error", err)
	} else if attr != nil {
		event.FirstTouchSource = attr.FirstTouchSource
		event.FirstTouchMedium = attr.FirstTouchMedium
		event.FirstTouchCampaign = attr.FirstTouchCampaign
		event.LastTouchSource = attr.LastTouchSource
		event.LastTouchMedium = attr.LastTouchMedium
		event.LastTouchCampaign = attr.LastTouchCampaign
		event.AttributionID = attr.LastAttributionID
		if event.AttributionID == "" {
			event.AttributionID = attr.FirstAttributionID
		}
	}
	fillLastTouchFromFirst(&event)
	return s.persistEvent(ctx, event)
}

func (s *Service) persistEvent(ctx context.Context, event Event) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if isSuccessEvent(event.Event) {
		existing, err := s.repo.FindSuccessMatch(ctx, event)
		if err != nil {
			slog.Warn("telemetry success match lookup failed", "error", err)
		} else if existing != nil {
			if err := s.repo.PatchEventObservability(ctx, existing.ID, event); err != nil {
				return err
			}
			return nil
		}
	}
	event.DedupKey = dedupKey(event)
	if err := s.repo.InsertEvent(ctx, event); err != nil {
		if errors.Is(err, ErrDuplicateEvent) {
			return nil
		}
		return err
	}
	return nil
}

func fillLastTouchFromFirst(event *Event) {
	if event == nil {
		return
	}
	if event.LastTouchSource == "" {
		event.LastTouchSource = event.FirstTouchSource
	}
	if event.LastTouchMedium == "" {
		event.LastTouchMedium = event.FirstTouchMedium
	}
	if event.LastTouchCampaign == "" {
		event.LastTouchCampaign = event.FirstTouchCampaign
	}
}

func isSuccessEvent(name string) bool {
	_, lifetime := lifetimeDedupEvents[name]
	_, windowed := windowedDedupEvents[name]
	return lifetime || windowed
}

func successMatchWindowStart(event Event) (time.Time, bool) {
	if _, ok := lifetimeDedupEvents[event.Event]; ok {
		return time.Time{}, false
	}
	if _, ok := windowedDedupEvents[event.Event]; ok {
		return event.OccurredAt.Add(-time.Duration(dedupWindowMillis) * time.Millisecond), true
	}
	return time.Time{}, false
}

func attributionFromEvent(event Event, userID int64) AccountAttribution {
	return AccountAttribution{
		UserID:             userID,
		FirstTouchSource:   event.FirstTouchSource,
		FirstTouchMedium:   event.FirstTouchMedium,
		FirstTouchCampaign: event.FirstTouchCampaign,
		FirstAttributionID: event.AttributionID,
		LastTouchSource:    event.LastTouchSource,
		LastTouchMedium:    event.LastTouchMedium,
		LastTouchCampaign:  event.LastTouchCampaign,
		LastAttributionID:  event.AttributionID,
		FirstTouchAt:       event.OccurredAt,
		LastTouchAt:        event.OccurredAt,
	}
}

func dedupKey(event Event) string {
	idPart := event.AttributionID
	if idPart == "" && event.UserID != nil && *event.UserID > 0 {
		idPart = "u:" + strconv.FormatInt(*event.UserID, 10)
	}
	if idPart == "" {
		return ""
	}
	if _, ok := lifetimeDedupEvents[event.Event]; ok {
		return event.Event + ":" + idPart
	}
	if _, ok := windowedDedupEvents[event.Event]; ok {
		bucket := event.OccurredAt.UnixMilli() / dedupWindowMillis
		return event.Event + ":" + idPart + ":" + strconv.FormatInt(bucket, 10)
	}
	return ""
}
