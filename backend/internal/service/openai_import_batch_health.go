package service

import (
	"context"
	"sync"
	"time"
)

// Runtime-only, like the existing scheduler EWMA. Each gateway independently
// learns account failures. Persistent credential/quota blocks remain authoritative.
type importBatchHealth struct {
	mu                sync.Mutex
	outcomes          []importBatchOutcome
	degraded          bool
	nextProbe         time.Time
	serial            uint64
	pending           uint64
	active            uint64
	recoverySuccesses int
}

type importBatchOutcome struct {
	at      time.Time
	success bool
}

func (h *importBatchHealth) prune(now time.Time) {
	first := 0
	for first < len(h.outcomes) && now.Sub(h.outcomes[first].at) > 10*time.Minute {
		first++
	}
	h.outcomes = h.outcomes[first:]
}

func (s *openAIAccountRuntimeStats) batchState(id int64) *importBatchHealth {
	if s == nil || id <= 0 {
		return nil
	}
	return &s.loadOrCreate(id).batch
}

func (s *openAIAccountRuntimeStats) batchDegraded(id int64) bool {
	h := s.batchState(id)
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.degraded
}

func (s *openAIAccountRuntimeStats) batchErrorRate(id int64, now time.Time) float64 {
	h := s.batchState(id)
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prune(now)
	if len(h.outcomes) == 0 {
		return 0
	}
	failures := 0
	for _, v := range h.outcomes {
		if !v.success {
			failures++
		}
	}
	return float64(failures) / float64(len(h.outcomes))
}

func (s *openAIAccountRuntimeStats) recordBatchOutcome(id int64, token uint64, success bool, now time.Time) {
	h := s.batchState(id)
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.degraded {
		// Completions from requests already running before demotion cannot heal it.
		if token == 0 || token != h.pending {
			return
		}
		h.pending = 0
		if success {
			h.recoverySuccesses++
		} else {
			h.recoverySuccesses = 0
		}
		h.nextProbe = now.Add(2 * time.Minute)
		if h.recoverySuccesses >= 3 {
			h.degraded = false
			h.outcomes = nil
			h.recoverySuccesses = 0
		}
		return
	}
	h.prune(now)
	h.outcomes = append(h.outcomes, importBatchOutcome{now, success})
	if len(h.outcomes) > 20 {
		h.outcomes = h.outcomes[len(h.outcomes)-20:]
	}
	failures, consecutive := 0, 0
	for _, v := range h.outcomes {
		if !v.success {
			failures++
			consecutive++
		} else {
			consecutive = 0
		}
	}
	if consecutive >= 5 || (len(h.outcomes) >= 10 && failures*2 >= len(h.outcomes)) {
		h.degraded = true
		h.nextProbe = now.Add(2 * time.Minute)
		h.recoverySuccesses = 0
	}
}

func (s *openAIAccountRuntimeStats) reserveBatchProbe(id int64, now time.Time) (uint64, bool) {
	h := s.batchState(id)
	if h == nil {
		return 0, true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.degraded {
		return 0, true
	}
	if h.active != 0 || now.Before(h.nextProbe) {
		return 0, false
	}
	h.serial++
	h.active = h.serial
	h.pending = h.serial
	h.nextProbe = now.Add(2 * time.Minute)
	return h.serial, true
}

func (s *openAIAccountRuntimeStats) cancelBatchProbe(id int64, token uint64) {
	if token == 0 {
		return
	}
	h := s.batchState(id)
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active == token {
		h.active = 0
		h.pending = 0
		h.nextProbe = time.Time{}
	}
}

func (s *openAIAccountRuntimeStats) releaseBatchProbe(id int64, token uint64) {
	if token == 0 {
		return
	}
	h := s.batchState(id)
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active == token {
		h.active = 0
	}
}

func (s *OpenAIGatewayService) reportImportBatchOutcome(account *Account, success bool, observedErr ...error) {
	if s == nil || account == nil || account.Platform != PlatformOpenAI || s.openAIImportBatchMinutes(context.Background()) == 0 {
		return
	}
	// Initialize through sync.Once before accessing runtime stats.
	if s.getOpenAIAccountScheduler(context.Background()) == nil {
		return
	}
	if !success {
		if len(observedErr) == 0 {
			return
		}
		if _, _, eligible := classifyOpenAIAPIKeyHealthFailure(observedErr[0]); !eligible {
			return
		}
	}
	s.openaiAccountStats.recordBatchOutcome(account.ID, account.importBatchProbeToken, success, time.Now())
}
