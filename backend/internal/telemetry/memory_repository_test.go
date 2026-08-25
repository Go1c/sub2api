//go:build unit

package telemetry

import (
	"context"
	"sync"
	"time"
)

type memoryRepository struct {
	mu           sync.Mutex
	events       []Event
	attributions map[int64]AccountAttribution
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{attributions: make(map[int64]AccountAttribution)}
}

func (r *memoryRepository) InsertEvent(_ context.Context, event Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.DedupKey != "" {
		for _, existing := range r.events {
			if existing.DedupKey == event.DedupKey {
				return ErrDuplicateEvent
			}
		}
	}
	cloned := cloneEvent(event)
	cloned.ID = int64(len(r.events) + 1)
	r.events = append(r.events, cloned)
	return nil
}

func (r *memoryRepository) FindSuccessMatch(_ context.Context, event Event) (*Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !isSuccessEvent(event.Event) {
		return nil, nil
	}
	windowStart, windowed := successMatchWindowStart(event)
	for i := range r.events {
		existing := r.events[i]
		if existing.Event != event.Event {
			continue
		}
		if windowed && existing.OccurredAt.Before(windowStart) {
			continue
		}
		sameAID := event.AttributionID != "" && existing.AttributionID == event.AttributionID
		sameUser := event.UserID != nil && existing.UserID != nil && *event.UserID == *existing.UserID
		if sameAID || sameUser {
			copied := cloneEvent(existing)
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *memoryRepository) PatchEventObservability(_ context.Context, id int64, incoming Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.events {
		if r.events[i].ID != id {
			continue
		}
		existing := r.events[i]
		if (existing.ClientSource == "" || existing.ClientSource == ClientSourceUnknown) &&
			incoming.ClientSource != "" && incoming.ClientSource != ClientSourceUnknown {
			existing.ClientSource = incoming.ClientSource
		}
		if existing.Route == "" {
			existing.Route = incoming.Route
		}
		if existing.AuthMethod == "" {
			existing.AuthMethod = incoming.AuthMethod
		}
		if existing.Platform == "" {
			existing.Platform = incoming.Platform
		}
		if existing.Destination == "" {
			existing.Destination = incoming.Destination
		}
		if existing.ErrorCode == "" {
			existing.ErrorCode = incoming.ErrorCode
		}
		if existing.AttributionID == "" {
			existing.AttributionID = incoming.AttributionID
		}
		if existing.FirstTouchSource == "" {
			existing.FirstTouchSource = incoming.FirstTouchSource
		}
		if existing.FirstTouchMedium == "" {
			existing.FirstTouchMedium = incoming.FirstTouchMedium
		}
		if existing.FirstTouchCampaign == "" {
			existing.FirstTouchCampaign = incoming.FirstTouchCampaign
		}
		if existing.LastTouchSource == "" {
			existing.LastTouchSource = incoming.LastTouchSource
		}
		if existing.LastTouchMedium == "" {
			existing.LastTouchMedium = incoming.LastTouchMedium
		}
		if existing.LastTouchCampaign == "" {
			existing.LastTouchCampaign = incoming.LastTouchCampaign
		}
		if existing.UserID == nil && incoming.UserID != nil {
			idCopy := *incoming.UserID
			existing.UserID = &idCopy
		}
		r.events[i] = existing
		return nil
	}
	return nil
}

func cloneEvent(event Event) Event {
	cloned := event
	if event.UserID != nil {
		id := *event.UserID
		cloned.UserID = &id
	}
	return cloned
}

func (r *memoryRepository) GetAttribution(_ context.Context, userID int64) (*AccountAttribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attr, ok := r.attributions[userID]
	if !ok {
		return nil, nil
	}
	copied := attr
	return &copied, nil
}

func (r *memoryRepository) UpsertAttribution(_ context.Context, incoming AccountAttribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.attributions[incoming.UserID]
	if !ok {
		r.attributions[incoming.UserID] = incoming
		return nil
	}
	if existing.FirstTouchSource == "" {
		existing.FirstTouchSource = incoming.FirstTouchSource
	}
	if existing.FirstTouchMedium == "" {
		existing.FirstTouchMedium = incoming.FirstTouchMedium
	}
	if existing.FirstTouchCampaign == "" {
		existing.FirstTouchCampaign = incoming.FirstTouchCampaign
	}
	if existing.FirstAttributionID == "" {
		existing.FirstAttributionID = incoming.FirstAttributionID
	}
	if existing.FirstTouchAt.IsZero() {
		existing.FirstTouchAt = incoming.FirstTouchAt
	}
	if incoming.LastTouchSource != "" {
		existing.LastTouchSource = incoming.LastTouchSource
	}
	if incoming.LastTouchMedium != "" {
		existing.LastTouchMedium = incoming.LastTouchMedium
	}
	if incoming.LastTouchCampaign != "" {
		existing.LastTouchCampaign = incoming.LastTouchCampaign
	}
	if incoming.LastAttributionID != "" {
		existing.LastAttributionID = incoming.LastAttributionID
	}
	if !incoming.LastTouchAt.IsZero() {
		existing.LastTouchAt = incoming.LastTouchAt
	}
	r.attributions[incoming.UserID] = existing
	return nil
}

func (r *memoryRepository) Aggregate(_ context.Context, query StatsQuery) ([]StatsRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	from := unixMilli(query.From)
	to := unixMilli(query.To)
	counts := map[string]int64{}
	unique := map[string]map[string]struct{}{}
	for _, event := range r.events {
		if event.OccurredAt.Before(from) || event.OccurredAt.After(to) {
			continue
		}
		if query.ClientSource != "" && event.ClientSource != query.ClientSource {
			continue
		}
		if query.Campaign != "" && event.FirstTouchCampaign != query.Campaign {
			continue
		}
		if query.Event != "" && event.Event != query.Event {
			continue
		}
		counts[event.Event]++
		if unique[event.Event] == nil {
			unique[event.Event] = map[string]struct{}{}
		}
		if event.AttributionID != "" {
			unique[event.Event][event.AttributionID] = struct{}{}
		}
	}
	rows := make([]StatsRow, 0, len(counts))
	for eventName, count := range counts {
		rows = append(rows, StatsRow{
			Event:                eventName,
			EventCount:           count,
			UniqueAttributionIDs: int64(len(unique[eventName])),
			Measure:              MeasureEventCountAndUniqueAnonymousID,
		})
	}
	return rows, nil
}

func unixMilli(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}
