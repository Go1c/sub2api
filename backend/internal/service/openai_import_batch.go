package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The override is independent of the advanced-score switch. Empty inherits the
// server setting; zero is an explicit rollback to the previous scheduler.
func (s *OpenAIGatewayService) openAIImportBatchMinutes(ctx context.Context) int {
	raw := strings.TrimSpace(s.openAIAdvancedSchedulerRuntimeSettings(ctx).batchMinutes)
	if raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && (n == 0 || n == 10 || n == 30) {
			return n
		}
	}
	if s.cfg != nil {
		return s.cfg.Gateway.OpenAIScheduler.ImportBatchMinutes
	}
	return 0
}

func importBatchNumber(createdAt time.Time, minutes int) int64 {
	// Missing timestamps must not make an account artificially older than every
	// imported account. Real persisted accounts always have a creation time.
	if createdAt.IsZero() {
		return 1<<63 - 1
	}
	seconds := int64(minutes) * 60
	stamp := createdAt.Unix()
	q := stamp / seconds
	if stamp < 0 && stamp%seconds != 0 {
		q--
	}
	return q
}

func (s *defaultOpenAIAccountScheduler) importBatchOrder(ctx context.Context, req OpenAIAccountScheduleRequest, accounts []*Account, loads map[int64]*AccountLoadInfo) []openAIAccountCandidateScore {
	order := make([]openAIAccountCandidateScore, 0, len(accounts))
	holding := make(map[int64]bool, len(accounts))
	for _, a := range accounts {
		load, known := loads[a.ID]
		if load == nil {
			load = &AccountLoadInfo{AccountID: a.ID}
			known = false
		}
		errorRate := s.stats.batchErrorRate(a.ID, time.Now())
		order = append(order, openAIAccountCandidateScore{account: a, loadInfo: load, loadKnown: known, errorRate: errorRate})
		holding[a.ID] = s.service.accountHasTurnStateHolding(ctx, a)
	}
	// Randomize exact ties, then apply strict lexicographic ordering. Scoring,
	// subscription/cost overrides and Top-K cannot move accounts across batches.
	rng := newOpenAISelectionRNG(deriveOpenAISelectionSeed(req))
	for i := len(order) - 1; i > 0; i-- {
		j := int(rng.nextUint64() % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.account.Priority != b.account.Priority {
			return a.account.Priority < b.account.Priority
		}
		ab, bb := importBatchNumber(a.account.CreatedAt, req.ImportBatchMinutes), importBatchNumber(b.account.CreatedAt, req.ImportBatchMinutes)
		if ab != bb {
			return ab < bb
		}
		if holding[a.account.ID] != holding[b.account.ID] {
			return holding[a.account.ID]
		}
		if req.RequireCompact {
			at, bt := openAICompactSupportTier(a.account), openAICompactSupportTier(b.account)
			if at != bt {
				return at > bt
			}
		}
		if a.loadInfo.LoadRate != b.loadInfo.LoadRate {
			return a.loadInfo.LoadRate < b.loadInfo.LoadRate
		}
		return a.errorRate < b.errorRate
	})
	return order
}

func (s *defaultOpenAIAccountScheduler) selectByImportBatch(ctx context.Context, req OpenAIAccountScheduleRequest, accounts []*Account, loads map[int64]*AccountLoadInfo, filters openAISelectionFilterStats) (*AccountSelectionResult, int, int, float64, error) {
	budget := newOpenAISelectionProbeBudget()
	order := s.importBatchOrder(ctx, req, accounts, loads)
	compactBlocked := false
	for pass := 0; pass < 2; pass++ {
		for _, candidate := range order {
			token, allowed := s.stats.reserveBatchProbe(candidate.account.ID, time.Now())
			if !allowed {
				continue
			}
			attemptReq := req
			if token > 0 {
				attemptReq.PreserveStickyBinding = true
			}
			result, blocked, err := s.tryAcquireOpenAISelectionOrderWithBudget(ctx, attemptReq, []openAIAccountCandidateScore{candidate}, budget)
			compactBlocked = compactBlocked || blocked
			if result == nil || err != nil {
				s.stats.cancelBatchProbe(candidate.account.ID, token)
			}
			if err != nil {
				return nil, len(accounts), 0, 0, err
			}
			if result != nil {
				if token > 0 {
					copy := *result.Account
					copy.importBatchProbeToken = token
					result.Account = &copy
					release := result.ReleaseFunc
					result.ReleaseFunc = func() {
						if release != nil {
							release()
						}
						s.stats.releaseBatchProbe(copy.ID, token)
					}
				}
				return result, len(accounts), 0, 0, nil
			}
		}
		if pass == 0 && s.service.concurrencyService != nil {
			fresh, err := s.service.concurrencyService.GetAccountsLoadBatchFresh(ctx, buildOpenAIAccountLoadRequest(accounts))
			if err == nil {
				order = s.importBatchOrder(ctx, req, accounts, fresh)
				continue
			}
		}
		break
	}
	// Never queue normal traffic behind a degraded account or an active probe.
	waiting := make([]openAIAccountCandidateScore, 0, len(order))
	for _, candidate := range order {
		if !s.stats.batchDegraded(candidate.account.ID) {
			waiting = append(waiting, candidate)
		}
	}
	return s.finishLoadBalanceSelectionFallback(ctx, req, openAIAccountLoadSelectionAttempt{selectionOrder: waiting, candidateCount: len(accounts), compactBlocked: compactBlocked}, budget, filters)
}
