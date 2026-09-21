package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"sort"
	"strings"
	"time"
)

func isTurnStateOverload(ev RequestHealthEvent) bool {
	if ev.Slot != RequestHealthSlotFail {
		return false
	}
	// Do not treat a generic 503 or rate limit as an overloaded response.
	msg := strings.ToLower(strings.TrimSpace(ev.Message))
	return isOpenAICapacityShedMessage(msg) || msg == "overloaded" || msg == "server_is_overloaded" || strings.Contains(msg, "server overloaded")
}

// Reuse the account-wide health history, which already combines all egress IPs.
// The periodic runner picks up the due ticket on its next (one-second) tick.
func (s *TurnStateProbeService) refreshAfterOverload(ctx context.Context, ev RequestHealthEvent, history RequestHealthStore) error {
	if s == nil || s.tickets == nil || s.accountRepo == nil || !isTurnStateOverload(ev) {
		return nil
	}
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return err
	}
	if !policy.Enabled || policy.OverloadThreshold <= 0 {
		return nil
	}
	events, err := history.List(ctx, ev.AccountID, 0, RequestHealthMaxEvents)
	if err != nil {
		return err
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].OccurredAt.Before(events[j].OccurredAt) })
	if len(events) < policy.OverloadThreshold {
		return nil
	}
	tail := events[len(events)-policy.OverloadThreshold:]
	for _, item := range tail {
		if !isTurnStateOverload(item) {
			return nil
		}
	}
	account, err := s.accountRepo.GetByID(ctx, ev.AccountID)
	if err != nil {
		return err
	}
	if account == nil || !account.IsOpenAIOAuth() || !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return nil
	}
	locked, err := s.tickets.TryLock(ctx, ev.AccountID, turnStateProbeLockTTL)
	if err != nil || !locked {
		return err
	}
	defer func() { _ = s.tickets.Unlock(context.Background(), ev.AccountID) }()
	ticket, err := s.tickets.Get(ctx, ev.AccountID)
	if err != nil || ticket == nil {
		return err
	}
	// A pending refresh/retry absorbs further overload notifications. Never revive
	// a credential failure, or let delayed old errors invalidate a fresh ticket.
	if ticket.Status != turnStateProbeStatusHolding || ticket.PolicyRevision != policy.Revision {
		return nil
	}
	if ticket.StickyGroupID > 0 && ticket.StickyUntil.After(time.Now()) {
		return nil
	}
	harvested, _ := turnStateProbeLifetime(ticket)
	if ticket.UpdatedAt.After(harvested) {
		harvested = ticket.UpdatedAt
	}
	if !tail[0].OccurredAt.After(harvested) {
		return nil
	}
	ticket.HarvestedAt, ticket.ExpiresAt = turnStateProbeLifetime(ticket)
	ticket.Status = turnStateProbeStatusCooldown
	ticket.RecheckAt = time.Now()
	ticket.LastError = "consecutive_overloaded"
	if err := s.tickets.Put(ctx, *ticket); err != nil {
		return err
	}
	logger.LegacyPrintf("service.turn_state_probe", "renewal_requested account_id=%d reason=consecutive_overloaded threshold=%d", ev.AccountID, policy.OverloadThreshold)
	return nil
}
