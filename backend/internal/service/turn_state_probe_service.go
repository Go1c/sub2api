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
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	turnStateProbeStatusHolding      = "holding"
	turnStateProbeStatusFailed       = "failed"
	turnStateProbeHarvestTimeout     = 45 * time.Second
	turnStateProbeLockTTL            = 2 * time.Minute
	turnStateProbeMaxAttempts        = 8
	turnStateProbeSIDLen             = 8
	turnStateProbeBindTTL            = 2 * time.Hour
	turnStateProbeHarvestConcurrency = 2
	turnStateProbeSIDAlphabet        = "abcdefghijklmnopqrstuvwxyz0123456789"
	turnStateProbePolicyCacheTTL     = 2 * time.Second
)

type cachedTurnStateProbePolicy struct {
	policy TurnStateProbePolicy
	at     time.Time
}

var ErrTurnStateProbeRateLimited = infraerrors.TooManyRequests("TURN_STATE_PROBE_RPM", "探测调用预算已用尽")

type TurnStateProbeService struct {
	settingRepo  SettingRepository
	accountRepo  AccountRepository
	proxyRepo    ProxyRepository
	tickets      TurnStateTicketStore
	httpUpstream HTTPUpstream
	tokens       *OpenAITokenProvider
	policyCache  atomic.Value
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
	}
}

func ProvideTurnStateProbeService(
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tickets TurnStateTicketStore,
	httpUpstream HTTPUpstream,
	tokens *OpenAITokenProvider,
) *TurnStateProbeService {
	return NewTurnStateProbeService(settingRepo, accountRepo, proxyRepo, tickets, httpUpstream, tokens)
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
				}
			}
			accounts = append(accounts, item)
		}
	}
	return &TurnStateProbeOverview{Policy: policy, Accounts: accounts}, nil
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
	return s.ProbeAccount(ctx, id)
}

func (s *TurnStateProbeService) BindCurrent(ctx context.Context, account *Account, identity, turnKey, model string) (string, bool) {
	if s == nil || s.tickets == nil || account == nil || !account.IsOpenAI() {
		return "", false
	}
	if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return "", false
	}
	policy, err := s.GetPolicy(ctx)
	if err != nil || !policy.Enabled {
		return "", false
	}
	ticket, err := s.tickets.Get(ctx, account.ID)
	if err != nil || ticket == nil {
		return "", false
	}
	if strings.TrimSpace(ticket.State) == "" || ticket.Status != turnStateProbeStatusHolding {
		return "", false
	}
	if ticket.PolicyRevision != policy.Revision {
		return "", false
	}
	_ = identity
	_ = model
	if strings.TrimSpace(turnKey) == "" {
		return ticket.State, true
	}
	bound, err := s.tickets.BindTurn(ctx, account.ID, turnKey, ticket.State, turnStateProbeBindTTL)
	if err != nil || strings.TrimSpace(bound) == "" {
		return ticket.State, true
	}
	return bound, true
}

func (s *TurnStateProbeService) ProbeAccount(ctx context.Context, accountID int64) error {
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
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return err
	}
	if !policy.Enabled {
		return ErrTurnStateProbeDisabled
	}
	if !policy.HasProbeExit() {
		return ErrTurnStateProbeInvalid
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

	ok, err := s.tickets.AllowRPM(ctx, "global", policy.RPM)
	if err != nil {
		return err
	}
	if !ok {
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
		return ErrTurnStateProbeRateLimited
	}

	useDynamic := turnStateProbeDynamicReady(policy.Dynamic)
	proxyURLs, err := s.probeProxyURLs(ctx, policy)
	if err != nil {
		return err
	}
	if !useDynamic && len(proxyURLs) == 0 {
		return ErrTurnStateProbeInvalid
	}

	var lastReason string
	for attempt := 0; attempt < turnStateProbeMaxAttempts; attempt++ {
		proxyURL := ""
		if useDynamic {
			sid, sidErr := randomTurnStateProbeSID()
			if sidErr != nil {
				lastReason = "sid"
				continue
			}
			proxyURL, err = BuildTurnStateProbeDynamicProxyURL(policy.Dynamic, sid)
			if err != nil {
				lastReason = "dynamic_exit"
				continue
			}
		} else {
			proxyURL = proxyURLs[attempt%len(proxyURLs)]
		}

		result, authErr, harvestErr := s.harvestOnce(ctx, account, policy, proxyURL)
		if authErr != nil {
			return authErr
		}
		if harvestErr != nil {
			lastReason = harvestErr.Error()
			continue
		}
		ok, reason := EvaluateTurnStateProbeAttempt(result, policy)
		if !ok {
			lastReason = reason
			continue
		}
		now := time.Now()
		rec := TurnStateTicketRecord{
			AccountID:      account.ID,
			Identity:       turnStateProbeTicketIdentity(account),
			State:          result.State,
			StateHash:      TurnStateProbeStateHash(result.State),
			StateLength:    len(strings.TrimSpace(result.State)),
			Model:          firstNonEmpty(result.ObservedModel, policy.Model),
			PolicyRevision: policy.Revision,
			Status:         turnStateProbeStatusHolding,
			RecheckAt:      now.Add(policy.RecheckAfter()),
			UpdatedAt:      now,
		}
		logger.LegacyPrintf("service.turn_state_probe", "harvested account_id=%d hash=%s length=%d model=%s", account.ID, rec.StateHash, rec.StateLength, rec.Model)
		return s.tickets.Put(ctx, rec)
	}

	fail := TurnStateTicketRecord{
		AccountID:      accountID,
		Identity:       turnStateProbeTicketIdentity(account),
		PolicyRevision: policy.Revision,
		Status:         turnStateProbeStatusFailed,
		LastError:      lastReason,
		UpdatedAt:      time.Now(),
	}
	_ = s.tickets.Put(ctx, fail)
	if lastReason == "" {
		lastReason = "probe_failed"
	}
	return infraerrors.BadRequest("TURN_STATE_PROBE_FAILED", lastReason)
}

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
	if len(ids) == 0 {
		return nil
	}
	sem := make(chan struct{}, turnStateProbeHarvestConcurrency)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(accountID int64) {
			defer wg.Done()
			defer func() { <-sem }()
			err := s.ProbeAccount(ctx, accountID)
			if err == nil || errors.Is(err, ErrTurnStateProbeBusy) || errors.Is(err, ErrTurnStateProbeRateLimited) {
				return
			}
			logger.LegacyPrintf("service.turn_state_probe", "harvest account_id=%d: %v", accountID, err)
		}(id)
	}
	wg.Wait()
	return nil
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
		if s.ticketDue(ctx, account.ID, policy, now) {
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
	if ticket.Status != turnStateProbeStatusHolding || strings.TrimSpace(ticket.State) == "" {
		return true
	}
	if ticket.PolicyRevision != policy.Revision {
		return true
	}
	if ticket.RecheckAt.IsZero() || !ticket.RecheckAt.After(now) {
		return true
	}
	return false
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
		_, _ = io.Copy(io.Discard, resp.Body)
		return attempt, infraerrors.Forbidden("TURN_STATE_PROBE_FORBIDDEN", "探测上游返回 403"), nil
	}
	sse := parseTurnStateProbeSSE(resp.Body)
	attempt.ObservedModel = sse.Model
	attempt.AnswerText = sse.Text
	return attempt, nil, nil
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
