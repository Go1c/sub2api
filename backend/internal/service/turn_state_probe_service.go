package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	turnStateProbeStatusHolding      = "holding"
	turnStateProbeStatusFailed       = "failed"
	turnStateProbeStatusRunning      = "running"
	turnStateProbeStatusSkipped      = "skipped"
	turnStateProbeStatusCooldown     = "cooldown"
	turnStateProbeHarvestTimeout     = 45 * time.Second
	turnStateProbeLockTTL            = 2 * time.Minute
	turnStateProbeSIDLen             = 8
	turnStateProbeRetryInterval      = 45 * time.Second
	turnStateProbeHarvestConcurrency = 10
	turnStateProbeSIDAlphabet        = "abcdefghijklmnopqrstuvwxyz0123456789"
	turnStateProbePolicyCacheTTL     = 2 * time.Second
)

type cachedTurnStateProbePolicy struct {
	policy TurnStateProbePolicy
	at     time.Time
}

var ErrTurnStateProbeRateLimited = infraerrors.TooManyRequests("TURN_STATE_PROBE_RPM", "探测调用预算已用尽")

type TurnStateProbeService struct {
	groups       ProxyIPGroupRepository
	settingRepo  SettingRepository
	accountRepo  AccountRepository
	proxyRepo    ProxyRepository
	tickets      TurnStateTicketStore
	httpUpstream HTTPUpstream
	tokens       *OpenAITokenProvider
	policyCache  atomic.Value
	harvestSlots chan struct{}
}

func NewTurnStateProbeService(
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tickets TurnStateTicketStore,
	httpUpstream HTTPUpstream,
	tokens *OpenAITokenProvider,
) *TurnStateProbeService {
	return &TurnStateProbeService{
		settingRepo:  settingRepo,
		accountRepo:  accountRepo,
		proxyRepo:    proxyRepo,
		tickets:      tickets,
		httpUpstream: httpUpstream,
		tokens:       tokens,
		harvestSlots: make(chan struct{}, turnStateProbeHarvestConcurrency),
	}
}

func ProvideTurnStateProbeService(
	settingRepo SettingRepository,
	groups ProxyIPGroupRepository,
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tickets TurnStateTicketStore,
	httpUpstream HTTPUpstream,
	tokens *OpenAITokenProvider,
) *TurnStateProbeService {
	svc := NewTurnStateProbeService(settingRepo, accountRepo, proxyRepo, tickets, httpUpstream, tokens)
	svc.groups = groups
	return svc
}

func (s *TurnStateProbeService) GetPolicy(ctx context.Context) (TurnStateProbePolicy, error) {
	fallback, normErr := NormalizeTurnStateProbePolicy(DefaultTurnStateProbePolicy())
	if s == nil {
		return fallback, normErr
	}
	if cached, ok := s.policyCache.Load().(*cachedTurnStateProbePolicy); ok && cached != nil && time.Since(cached.at) < turnStateProbePolicyCacheTTL {
		return cached.policy, nil
	}
	if s.settingRepo == nil {
		return fallback, normErr
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyTurnStateProbePolicy)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			s.rememberPolicy(fallback)
			return fallback, normErr
		}
		return TurnStateProbePolicy{}, err
	}
	if strings.TrimSpace(raw) == "" {
		s.rememberPolicy(fallback)
		return fallback, normErr
	}
	policy, err := DecodeTurnStateProbePolicyJSON(raw)
	if err != nil {
		return policy, err
	}
	s.rememberPolicy(policy)
	return policy, nil
}

func (s *TurnStateProbeService) rememberPolicy(policy TurnStateProbePolicy) {
	if s == nil {
		return
	}
	s.policyCache.Store(&cachedTurnStateProbePolicy{policy: policy, at: time.Now()})
}

func (s *TurnStateProbeService) GetPolicyPublic(ctx context.Context) (TurnStateProbePolicy, error) {
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return TurnStateProbePolicy{}, err
	}
	return policy.Public(), nil
}

func (s *TurnStateProbeService) SavePolicy(ctx context.Context, next TurnStateProbePolicy) (TurnStateProbePolicy, error) {
	if s == nil || s.settingRepo == nil {
		return TurnStateProbePolicy{}, infraerrors.InternalServer("TURN_STATE_PROBE_UNAVAILABLE", "Turn-State 探测服务未就绪")
	}
	prev, err := s.GetPolicy(ctx)
	if err != nil {
		return TurnStateProbePolicy{}, err
	}
	next = MergeTurnStateProbePassword(next, prev)
	normalized, err := NormalizeTurnStateProbePolicy(next)
	if err != nil {
		return TurnStateProbePolicy{}, err
	}
	if TurnStateProbePolicyChanged(prev, normalized) {
		normalized.Revision = prev.Revision + 1
		if normalized.Revision < 1 {
			normalized.Revision = 1
		}
	} else {
		normalized.Revision = prev.Revision
	}
	normalized.UpdatedAt = time.Now()
	raw, err := json.Marshal(normalized)
	if err != nil {
		return TurnStateProbePolicy{}, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyTurnStateProbePolicy, string(raw)); err != nil {
		return TurnStateProbePolicy{}, err
	}
	s.rememberPolicy(normalized)
	return normalized.Public(), nil
}

func (s *TurnStateProbeService) GetOverview(ctx context.Context) (*TurnStateProbeOverview, error) {
	policy, err := s.GetPolicyPublic(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]TurnStateProbeAccountItem, 0)
	if s.accountRepo != nil {
		listed, listErr := s.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
		if listErr != nil {
			return nil, listErr
		}
		for i := range listed {
			account := listed[i]
			if !account.IsOpenAIOAuth() {
				continue
			}
			if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
				continue
			}
			item := TurnStateProbeAccountItem{
				AccountID: account.ID,
				Name:      account.Name,
				Enabled:   true,
			}
			if s.tickets != nil {
				ticket, ticketErr := s.tickets.Get(ctx, account.ID)
				if ticketErr == nil && ticket != nil {
					sum := ticket.Summary()
					item.Status = sum.Status
					item.StateHash = sum.StateHash
					item.StateLength = sum.StateLength
					item.Model = sum.Model
					item.PolicyRevision = sum.PolicyRevision
					item.RecheckAt = sum.RecheckAt
					item.LastError = sum.LastError
					item.LastProbedAt = sum.LastProbedAt
					item.StickyProxyID = sum.StickyProxyID
					item.StickyUntil = sum.StickyUntil
				}
			}
			accounts = append(accounts, item)
		}
	}
	return &TurnStateProbeOverview{Policy: policy, Accounts: accounts, ImportBatchRuntimeSuspended: OpenAIImportBatchRuntimeSuspended()}, nil
}

func (s *TurnStateProbeService) SetAccountEnabled(ctx context.Context, id int64, enabled bool) error {
	if s == nil || s.accountRepo == nil {
		return infraerrors.InternalServer("TURN_STATE_PROBE_UNAVAILABLE", "Turn-State 探测服务未就绪")
	}
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account == nil || !account.IsOpenAIOAuth() {
		return ErrTurnStateProbeDisabled
	}
	if err := s.accountRepo.UpdateExtra(ctx, id, map[string]any{
		TurnStateProbeExtraKey: TurnStateProbeSwitchMap(enabled),
	}); err != nil {
		return err
	}
	if !enabled && s.tickets != nil {
		if err := s.tickets.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *TurnStateProbeService) ClearTicket(ctx context.Context, id int64) error {
	if s == nil || s.tickets == nil {
		return nil
	}
	return s.tickets.Delete(ctx, id)
}

func (s *TurnStateProbeService) RunOne(ctx context.Context, id int64) error {
	return s.probeAccount(ctx, id, true)
}

func (s *TurnStateProbeService) ProbeAccount(ctx context.Context, accountID int64) error {
	return s.probeAccount(ctx, accountID, false)
}

func (s *TurnStateProbeService) probeAccount(ctx context.Context, accountID int64, resetAttempts bool) (probeErr error) {
	if s == nil || s.accountRepo == nil || s.tickets == nil || s.httpUpstream == nil {
		return infraerrors.InternalServer("TURN_STATE_PROBE_UNAVAILABLE", "Turn-State 探测服务未就绪")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil || !account.IsOpenAIOAuth() || !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return ErrTurnStateProbeDisabled
	}
	if account.Status != "" && account.Status != StatusActive {
		policy, polErr := s.GetPolicy(ctx)
		if polErr != nil {
			return polErr
		}
		_ = s.putHarvestTicket(ctx, account, policy, nil, turnStateProbeStatusSkipped, 0, "account_"+account.Status)
		return nil
	}
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return err
	}
	if !policy.Enabled {
		return ErrTurnStateProbeDisabled
	}
	stickyGroup, err := s.stickyGroup(ctx, account)
	if err != nil {
		return err
	}
	if stickyGroup == nil && !policy.HasProbeExit() {
		return ErrTurnStateProbeInvalid
	}
	existing, err := s.tickets.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if existing != nil && existing.State != "" {
		matches, configErr := s.stickyConfigurationMatches(ctx, account, existing)
		if configErr != nil {
			return configErr
		}
		if !matches {
			existing = nil
		}
	}
	if !resetAttempts && turnStateProbeSkipBlocks(existing, time.Now()) {
		return nil
	}
	if !resetAttempts && existing != nil && existing.Status == turnStateProbeStatusCooldown && existing.RecheckAt.After(time.Now()) {
		return nil
	}
	locked, err := s.tickets.TryLock(ctx, accountID, turnStateProbeLockTTL)
	if err != nil {
		return err
	}
	if !locked {
		return ErrTurnStateProbeBusy
	}
	defer func() {
		_ = s.tickets.Unlock(context.Background(), accountID)
	}()

	// Read under the account lock so a completed renewal cannot be overwritten
	// with the snapshot read before acquiring the lock.
	existing, err = s.tickets.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if existing != nil && existing.State != "" {
		matches, configErr := s.stickyConfigurationMatches(ctx, account, existing)
		if configErr != nil {
			return configErr
		}
		if !matches {
			existing = nil
		}
	}
	if !resetAttempts && existing != nil {
		if turnStateProbeSkipBlocks(existing, time.Now()) {
			return nil
		}
		if existing.Status == turnStateProbeStatusCooldown && existing.RecheckAt.After(time.Now()) {
			return nil
		}
	}
	if !resetAttempts && existing != nil && existing.Status == turnStateProbeStatusHolding && !s.ticketDue(ctx, accountID, policy, time.Now()) {
		return nil
	}
	if existing != nil && existing.StickyGroupID > 0 && stickyGroup != nil && existing.StickyGroupID == stickyGroup.ID && existing.StickyUntil.After(time.Now()) && existing.ExpiresAt.After(time.Now()) && existing.PolicyRevision == policy.Revision && existing.Identity == turnStateProbeTicketIdentity(account) {
		return nil
	}
	attempts := 0
	if existing != nil && !resetAttempts {
		attempts = existing.Attempts
	}
	authFailed := false
	lastReason := "probe_error"
	defer func() {
		if probeErr == nil || authFailed {
			return
		}
		// The upstream context may have timed out; still persist the retry time.
		retryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.putHarvestTicket(retryCtx, account, policy, existing, turnStateProbeStatusCooldown, attempts, lastReason); err != nil {
			probeErr = errors.Join(probeErr, err)
		}
	}()
	if err := s.putHarvestTicket(ctx, account, policy, existing, turnStateProbeStatusRunning, attempts, ""); err != nil {
		return err
	}
	useDynamic := turnStateProbeDynamicReady(policy.Dynamic)
	proxyURLs, err := s.probeProxyURLs(ctx, policy)
	if err != nil {
		return err
	}
	if stickyGroup == nil && !useDynamic && len(proxyURLs) == 0 {
		return ErrTurnStateProbeInvalid
	}
	ok, err := s.tickets.AllowRPM(ctx, "global", policy.RPM)
	if err != nil {
		return err
	}
	if !ok {
		lastReason = "rpm_budget"
		return ErrTurnStateProbeRateLimited
	}
	acctRPM := policy.RPM / 2
	if acctRPM < 1 {
		acctRPM = 1
	}
	ok, err = s.tickets.AllowRPM(ctx, "acct:"+strconv.FormatInt(accountID, 10), acctRPM)
	if err != nil {
		return err
	}
	if !ok {
		lastReason = "rpm_budget"
		return ErrTurnStateProbeRateLimited
	}
	proxyURL := ""
	var stickyRecord TurnStateTicketRecord
	if stickyGroup != nil {
		var exitErr error
		proxyURL, stickyRecord, exitErr = s.newStickyExit(ctx, stickyGroup, attempts)
		if exitErr != nil {
			return exitErr
		}
	} else if useDynamic {
		sid, err := randomTurnStateProbeSID()
		if err != nil {
			lastReason = "sid"
			return err
		}
		proxyURL, err = BuildTurnStateProbeDynamicProxyURL(policy.Dynamic, sid)
		if err != nil {
			lastReason = "dynamic_exit"
			return err
		}
	} else {
		proxyURL = proxyURLs[attempts%len(proxyURLs)]
	}
	// One attempt per scheduled run; persistence below applies the retry delay.
	attempts++
	harvestStarted := time.Now()
	result, authErr, harvestErr := s.harvestOnce(ctx, account, policy, proxyURL)
	if authErr != nil {
		authFailed = true
		return errors.Join(authErr, s.putHarvestTicket(ctx, account, policy, nil, turnStateProbeStatusSkipped, attempts, authErr.Error()))
	}
	if harvestErr != nil {
		lastReason = harvestErr.Error()
		return harvestErr
	}
	ok, reason := EvaluateTurnStateProbeAttempt(result, policy)
	if !ok {
		lastReason = reason
		return infraerrors.BadRequest("TURN_STATE_PROBE_FAILED", reason)
	}
	now := time.Now()
	generation := int64(1)
	if existing != nil && existing.Generation >= generation {
		generation = existing.Generation + 1
	}
	rec := TurnStateTicketRecord{
		AccountID: account.ID, Identity: turnStateProbeTicketIdentity(account),
		State: result.State, StateHash: TurnStateProbeStateHash(result.State), StateLength: len(strings.TrimSpace(result.State)),
		Model: firstNonEmpty(result.ObservedModel, policy.Model), PolicyRevision: policy.Revision,
		Status: turnStateProbeStatusHolding, Attempts: attempts,
		RecheckAt: harvestStarted.Add(policy.RecheckAfter()), UpdatedAt: now,
		HarvestedAt: harvestStarted, ExpiresAt: harvestStarted.Add(policy.RecheckAfter()),
		Generation: generation, ExitDigest: turnStateExitDigest(proxyURL),
	}
	if stickyGroup != nil {
		rec.copyStickyFrom(stickyRecord)
		rec.StickyUntil = now.Add(time.Duration(stickyGroup.StickyMinutes) * time.Minute)
		rec.RecheckAt = rec.StickyUntil
		if !rec.StickyExitExpiresAt.After(rec.StickyUntil) {
			return ErrTurnStateProbeInvalid
		}
		if rec.ExpiresAt.After(rec.StickyExitExpiresAt) {
			rec.ExpiresAt = rec.StickyExitExpiresAt
		}
	}
	if err := s.tickets.Put(ctx, rec); err != nil {
		return err
	}
	logger.LegacyPrintf("service.turn_state_probe", "harvested account_id=%d hash=%s length=%d model=%s attempts=%d", account.ID, rec.StateHash, rec.StateLength, rec.Model, rec.Attempts)
	return nil
}

// Old holding records have a trustworthy UpdatedAt (the successful harvest).
// Old running records do not: their UpdatedAt was rewritten on every attempt.
func turnStateProbeLifetime(rec *TurnStateTicketRecord) (harvested, expires time.Time) {
	if rec == nil {
		return
	}
	harvested, expires = rec.HarvestedAt, rec.ExpiresAt
	if harvested.IsZero() && rec.Status == turnStateProbeStatusHolding {
		harvested = rec.UpdatedAt
	}
	if harvested.IsZero() {
		return harvested, time.Time{}
	}
	// Usable until the same 20-minute recheck that schedules the next harvest.
	// A sticky hold can be longer and is stored on ExpiresAt at harvest time.
	if expires.IsZero() || expires.Before(harvested) {
		expires = harvested.Add(turnStateProbeDefaultRecheckMinutes * time.Minute)
	}
	return
}

func (s *TurnStateProbeService) putHarvestTicket(ctx context.Context, account *Account, policy TurnStateProbePolicy, previous *TurnStateTicketRecord, status string, attempts int, lastErr string) error {
	if s == nil || s.tickets == nil || account == nil {
		return nil
	}
	rec := TurnStateTicketRecord{
		AccountID:      account.ID,
		Identity:       turnStateProbeTicketIdentity(account),
		PolicyRevision: policy.Revision,
		Status:         status,
		Attempts:       attempts,
		LastError:      lastErr,
		UpdatedAt:      time.Now(),
	}
	if previous != nil && (status == turnStateProbeStatusRunning || status == turnStateProbeStatusCooldown) &&
		previous.PolicyRevision == policy.Revision && (previous.Identity == "" || previous.Identity == rec.Identity) {
		rec.copyStickyFrom(*previous)
		rec.ForbiddenRetries = previous.ForbiddenRetries
		rec.State = previous.State
		rec.StateHash = previous.StateHash
		rec.StateLength = previous.StateLength
		rec.Model = previous.Model
		rec.RecheckAt = previous.RecheckAt
		rec.HarvestedAt, rec.ExpiresAt = turnStateProbeLifetime(previous)
		rec.Generation = previous.Generation
		rec.ExitDigest = previous.ExitDigest
	}
	if status == turnStateProbeStatusCooldown {
		delay := turnStateProbeRetryInterval
		if lastErr == "http_403" {
			// Attempts is lifetime-wide; backoff only counts forbidden responses
			// since the last success. Keep this counter across RPM deferrals.
			if rec.ForbiddenRetries < 5 {
				rec.ForbiddenRetries++
			}
			delay = turnStateProbeForbiddenDelay(rec.ForbiddenRetries)
		}
		rec.RecheckAt = time.Now().Add(delay)
	}
	return s.tickets.Put(ctx, rec)
}

func (s *TurnStateProbeService) currentHoldingTicket(ctx context.Context, account *Account) (*TurnStateTicketRecord, bool) {
	if s == nil || s.tickets == nil || account == nil || !account.IsOpenAI() {
		return nil, false
	}
	if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return nil, false
	}
	policy, err := s.GetPolicy(ctx)
	if err != nil || !policy.Enabled {
		return nil, false
	}
	ticket, err := s.tickets.Get(ctx, account.ID)
	if err != nil || ticket == nil {
		return nil, false
	}
	if strings.TrimSpace(ticket.State) == "" || (ticket.Status != turnStateProbeStatusHolding && ticket.Status != turnStateProbeStatusRunning && ticket.Status != turnStateProbeStatusCooldown) {
		return nil, false
	}
	if ticket.PolicyRevision != policy.Revision {
		return nil, false
	}
	if matches, err := s.stickyConfigurationMatches(ctx, account, ticket); err != nil || !matches {
		return nil, false
	}
	if ticket.StickyGroupID > 0 {
		if account.ProxyIPGroupID == nil || *account.ProxyIPGroupID != ticket.StickyGroupID || !ticket.StickyExitExpiresAt.After(time.Now()) {
			return nil, false
		}
	}
	_, expires := turnStateProbeLifetime(ticket)
	if !expires.After(time.Now()) || (ticket.Identity != "" && ticket.Identity != turnStateProbeTicketIdentity(account)) {
		return nil, false
	}
	ticket.ExpiresAt = expires
	return ticket, true
}

func (s *TurnStateProbeService) HasHolding(ctx context.Context, account *Account) bool {
	_, ok := s.currentHoldingTicket(ctx, account)
	return ok
}

func (s *TurnStateProbeService) CurrentTicket(ctx context.Context, account *Account) (*TurnStateTicketRecord, bool) {
	return s.currentHoldingTicket(ctx, account)
}

func (s *TurnStateProbeService) BindCurrent(ctx context.Context, account *Account, identity, turnKey, model string) (string, bool) {
	ticket, ok := s.currentHoldingTicket(ctx, account)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(turnKey) == "" {
		return ticket.State, true
	}
	// Scope bindings to the current ticket generation. Old 2-hour bindings
	// cannot override a fresh harvest, even in an existing session.
	key := "v2:" + TurnStateProbeStateHash(ticket.State) + ":" + turnKey
	ttl := time.Until(ticket.ExpiresAt)
	if ttl <= 0 {
		return "", false
	}
	bound, err := s.tickets.BindTurn(ctx, account.ID, key, ticket.State, ttl)
	if !ticket.ExpiresAt.After(time.Now()) {
		return "", false
	}
	if err != nil || strings.TrimSpace(bound) == "" {
		return ticket.State, true
	}
	return bound, true
}

var _ TurnStateTicketLookup = (*TurnStateProbeService)(nil)

func (s *TurnStateProbeService) HarvestDue(ctx context.Context) error {
	if s == nil {
		return nil
	}
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return err
	}
	if !policy.Enabled {
		return nil
	}
	ids, err := s.listDueAccountIDs(ctx, policy)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if !s.tryStartHarvest(id) {
			break
		}
	}
	return nil
}

func (s *TurnStateProbeService) tryStartHarvest(accountID int64) bool {
	if s == nil {
		return false
	}
	if s.harvestSlots == nil {
		s.harvestSlots = make(chan struct{}, turnStateProbeHarvestConcurrency)
	}
	select {
	case s.harvestSlots <- struct{}{}:
		go func() {
			defer func() { <-s.harvestSlots }()
			ctx, cancel := context.WithTimeout(context.Background(), turnStateProbeLockTTL)
			defer cancel()
			err := s.ProbeAccount(ctx, accountID)
			if err == nil || errors.Is(err, ErrTurnStateProbeBusy) || errors.Is(err, ErrTurnStateProbeRateLimited) {
				return
			}
			logger.LegacyPrintf("service.turn_state_probe", "harvest account_id=%d: %v", accountID, err)
		}()
		return true
	default:
		return false
	}
}

func (s *TurnStateProbeService) listDueAccountIDs(ctx context.Context, policy TurnStateProbePolicy) ([]int64, error) {
	if s.accountRepo == nil {
		return nil, nil
	}
	listed, err := s.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	ids := make([]int64, 0)
	for i := range listed {
		account := listed[i]
		if !account.IsOpenAIOAuth() || !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
			continue
		}
		if s.tickets == nil {
			ids = append(ids, account.ID)
			continue
		}
		ticket, readErr := s.tickets.Get(ctx, account.ID)
		configurationChanged := false
		if readErr == nil && ticket != nil && ticket.State != "" {
			matches, configErr := s.stickyConfigurationMatches(ctx, &account, ticket)
			if configErr != nil {
				continue
			}
			configurationChanged = !matches
		}
		if configurationChanged || s.ticketDue(ctx, account.ID, policy, now) {
			ids = append(ids, account.ID)
		}
	}
	return ids, nil
}

func (s *TurnStateProbeService) ticketDue(ctx context.Context, accountID int64, policy TurnStateProbePolicy, now time.Time) bool {
	if s.tickets == nil {
		return true
	}
	ticket, err := s.tickets.Get(ctx, accountID)
	if err != nil || ticket == nil {
		return true
	}
	if ticket.Status == turnStateProbeStatusSkipped {
		return !turnStateProbeSkipBlocks(ticket, now)
	}
	if ticket.PolicyRevision != policy.Revision {
		return true
	}
	if ticket.Status == turnStateProbeStatusHolding && strings.TrimSpace(ticket.State) != "" {
		harvested, expires := turnStateProbeLifetime(ticket)
		if !expires.After(now) || !(func() time.Time {
			if ticket.StickyGroupID > 0 {
				return ticket.StickyUntil
			}
			return harvested.Add(policy.RecheckAfter())
		})().After(now) || ticket.RecheckAt.IsZero() || !ticket.RecheckAt.After(now) {
			return true
		}
		return false
	}
	if ticket.Status == turnStateProbeStatusCooldown {
		if !ticket.RecheckAt.IsZero() && ticket.RecheckAt.After(now) {
			return false
		}
		return true
	}
	if ticket.Status == turnStateProbeStatusRunning && now.Sub(ticket.UpdatedAt) < turnStateProbeLockTTL {
		return false
	}
	return true
}

func (s *TurnStateProbeService) probeProxyURLs(ctx context.Context, policy TurnStateProbePolicy) ([]string, error) {
	if turnStateProbeDynamicReady(policy.Dynamic) || s.proxyRepo == nil || len(policy.ProxyIDs) == 0 {
		return nil, nil
	}
	listed, err := s.proxyRepo.ListByIDs(ctx, policy.ProxyIDs)
	if err != nil {
		urls := make([]string, 0, len(policy.ProxyIDs))
		for _, id := range policy.ProxyIDs {
			proxy, getErr := s.proxyRepo.GetByID(ctx, id)
			if getErr != nil || proxy == nil {
				continue
			}
			if u := strings.TrimSpace(proxy.URL()); u != "" {
				urls = append(urls, u)
			}
		}
		return urls, nil
	}
	byID := make(map[int64]Proxy, len(listed))
	for i := range listed {
		byID[listed[i].ID] = listed[i]
	}
	urls := make([]string, 0, len(policy.ProxyIDs))
	for _, id := range policy.ProxyIDs {
		proxy, ok := byID[id]
		if !ok {
			continue
		}
		if u := strings.TrimSpace(proxy.URL()); u != "" {
			urls = append(urls, u)
		}
	}
	return urls, nil
}

func (s *TurnStateProbeService) harvestOnce(ctx context.Context, account *Account, policy TurnStateProbePolicy, proxyURL string) (TurnStateProbeAttempt, error, error) {
	attempt := TurnStateProbeAttempt{RequestedModel: policy.Model}
	token, err := s.probeAccessToken(ctx, account)
	if err != nil {
		return attempt, err, nil
	}
	payload, err := marshalTurnStateProbePayload(policy.Model, policy.Question)
	if err != nil {
		return attempt, nil, err
	}
	attemptCtx, cancel := context.WithTimeout(ctx, turnStateProbeHarvestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, chatgptCodexURL, bytes.NewReader(payload))
	if err != nil {
		return attempt, nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Host = "chatgpt.com"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	setTurnStateProbeCookie(req.Header, account)
	req.Header.Set("accept", "text/event-stream")
	canonical := resolveCodexOutboundIdentity("")
	req.Header.Set("Originator", canonical.originator)
	if customUA := strings.TrimSpace(account.GetOpenAIUserAgent()); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	} else {
		req.Header.Set("User-Agent", canonical.userAgent)
	}
	setOpenAIChatGPTAccountHeaders(req.Header, account)
	enforceCodexIdentityHeadersWithUA(req.Header, account.GetOpenAIUserAgent())
	req.Header.Del("OpenAI-Beta")

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return attempt, nil, errors.New("upstream_error")
	}
	if resp == nil {
		return attempt, nil, errors.New("upstream_error")
	}
	defer func() { _ = resp.Body.Close() }()
	attempt.StatusCode = resp.StatusCode
	attempt.State = extractOpenAICodexTurnState(resp.Header)
	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, resp.Body)
		return attempt, infraerrors.Unauthorized("TURN_STATE_PROBE_UNAUTHORIZED", "探测上游返回 401"), nil
	}
	if resp.StatusCode == http.StatusForbidden {
		diagnostic, accountFailure := inspectTurnStateProbeForbidden(resp)
		logger.LegacyPrintf("service.turn_state_probe", "forbidden account_id=%d %s state_length=%d state_hash=%s", account.ID, diagnostic, len(attempt.State), TurnStateProbeStateHash(attempt.State))
		if accountFailure {
			return attempt, infraerrors.Forbidden("TURN_STATE_PROBE_ACCOUNT_FORBIDDEN", "探测上游明确拒绝账号或凭据"), nil
		}
		return attempt, nil, errors.New("http_403")
	}
	sse := parseTurnStateProbeSSE(resp.Body)
	attempt.ObservedModel = sse.Model
	attempt.AnswerText = sse.Text
	return attempt, nil, nil
}

// setTurnStateProbeCookie forwards the account's explicitly stored browser
// cookie to the probe. The normal gateway deliberately strips inbound Cookie
// headers; this is a server-owned probe request and needs the credential that
// was configured for the account. Accept session_key as a legacy alias.
func setTurnStateProbeCookie(headers http.Header, account *Account) {
	if headers == nil || account == nil {
		return
	}
	for _, key := range []string{"cookie", "session_key"} {
		if value := strings.TrimSpace(account.GetCredential(key)); value != "" {
			headers.Set("Cookie", value)
			return
		}
	}
}

func (s *TurnStateProbeService) probeAccessToken(ctx context.Context, account *Account) (string, error) {
	if s.tokens != nil {
		token, err := s.tokens.GetAccessToken(ctx, account)
		if err != nil {
			return "", err
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return "", infraerrors.Unauthorized("TURN_STATE_PROBE_NO_TOKEN", "账号缺少 access token")
		}
		return token, nil
	}
	token := strings.TrimSpace(account.GetOpenAIAccessToken())
	if token == "" {
		return "", infraerrors.Unauthorized("TURN_STATE_PROBE_NO_TOKEN", "账号缺少 access token")
	}
	return token, nil
}

func turnStateProbeDynamicReady(exit TurnStateProbeDynamicExit) bool {
	return strings.TrimSpace(exit.Host) != "" &&
		strings.TrimSpace(exit.Username) != "" &&
		strings.TrimSpace(exit.Password) != ""
}

func turnStateProbeTicketIdentity(account *Account) string {
	if ns := strings.TrimSpace(codexAccountIdentityNamespace(account)); ns != "" {
		return ns
	}
	if account == nil || account.ID <= 0 {
		return ""
	}
	return "id:" + strconv.FormatInt(account.ID, 10)
}

func randomTurnStateProbeSID() (string, error) {
	raw := make([]byte, turnStateProbeSIDLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, turnStateProbeSIDLen)
	n := len(turnStateProbeSIDAlphabet)
	for i := range raw {
		out[i] = turnStateProbeSIDAlphabet[int(raw[i])%n]
	}
	return string(out), nil
}
