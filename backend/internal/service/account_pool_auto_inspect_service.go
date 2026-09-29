package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	accountPoolAutoInspectTick        = 30 * time.Second
	accountPoolAutoInspectRunTimeout  = 3 * time.Minute
	accountPoolAutoInspectAskTimeout  = 180 * time.Second
	accountPoolAutoInspectMaxPerRun   = 20
	accountPoolIQPauseReason          = "pool auto inspect: group switch cooldown"
)

type poolAutoInspectAccounts interface {
	ListAllWithFilters(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, error)
	BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
	SetError(ctx context.Context, id int64, errorMsg string) error
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type poolAutoInspectSettings interface {
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type poolAutoInspectGroups interface {
	GetByIDLite(ctx context.Context, id int64) (*Group, error)
}

// poolAutoInspectQuiz asks one account the configured question on its own exit.
type poolAutoInspectQuiz interface {
	Ask(ctx context.Context, account *Account, model, question string) (text string, askErr error)
}

type AccountPoolAutoInspectService struct {
	settings   poolAutoInspectSettings
	accounts   poolAutoInspectAccounts
	groups     poolAutoInspectGroups
	quiz       poolAutoInspectQuiz
	lockStore  OpsAccountErrorAlertLockStore
	instanceID string

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu         sync.Mutex
	lastRunAt  time.Time
	lastResult string
	lastError  string

	dueMu sync.Mutex
	dueAt map[int64]time.Time
}

func ProvideAccountPoolAutoInspectService(
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	upstream HTTPUpstream,
	tokens *OpenAITokenProvider,
	lockStore OpsAccountErrorAlertLockStore,
) *AccountPoolAutoInspectService {
	svc := NewAccountPoolAutoInspectService(
		settingRepo,
		accountRepo,
		groupRepo,
		&accountPoolAutoInspectHTTPQuiz{upstream: upstream, tokens: tokens},
		lockStore,
	)
	svc.Start()
	return svc
}

func NewAccountPoolAutoInspectService(
	settings poolAutoInspectSettings,
	accounts poolAutoInspectAccounts,
	groups poolAutoInspectGroups,
	quiz poolAutoInspectQuiz,
	lockStore OpsAccountErrorAlertLockStore,
) *AccountPoolAutoInspectService {
	return &AccountPoolAutoInspectService{
		settings:   settings,
		accounts:   accounts,
		groups:     groups,
		quiz:       quiz,
		lockStore:  lockStore,
		instanceID: uuid.NewString(),
		dueAt:      map[int64]time.Time{},
	}
}

func (s *AccountPoolAutoInspectService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if s.stopCh == nil {
			s.stopCh = make(chan struct{})
		}
		s.wg.Add(1)
		go s.run()
	})
}

func (s *AccountPoolAutoInspectService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
	s.wg.Wait()
}

func (s *AccountPoolAutoInspectService) run() {
	defer s.wg.Done()
	timer := time.NewTimer(accountPoolAutoInspectTick)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			s.RunOnce(context.Background(), false)
			timer.Reset(accountPoolAutoInspectTick)
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountPoolAutoInspectService) GetConfig(ctx context.Context) (*AccountPoolAutoInspectStatus, error) {
	cfg, err := s.GetStoredConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s.statusFrom(cfg), nil
}

func (s *AccountPoolAutoInspectService) statusFrom(cfg *AccountPoolAutoInspectConfig) *AccountPoolAutoInspectStatus {
	if cfg == nil {
		cfg = defaultAccountPoolAutoInspectConfig()
	}
	status := &AccountPoolAutoInspectStatus{AccountPoolAutoInspectConfig: *cfg}
	s.mu.Lock()
	if !s.lastRunAt.IsZero() {
		runAt := s.lastRunAt
		status.LastRunAt = &runAt
	}
	status.LastResult = s.lastResult
	status.LastError = s.lastError
	s.mu.Unlock()
	return status
}

func (s *AccountPoolAutoInspectService) GetStoredConfig(ctx context.Context) (*AccountPoolAutoInspectConfig, error) {
	if s == nil || s.settings == nil {
		return defaultAccountPoolAutoInspectConfig(), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := s.settings.GetValue(ctx, SettingKeyAccountPoolAutoInspectConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			cfg := defaultAccountPoolAutoInspectConfig()
			if encoded, mErr := marshalAccountPoolAutoInspectConfig(cfg); mErr == nil {
				_ = s.settings.Set(ctx, SettingKeyAccountPoolAutoInspectConfig, encoded)
			}
			return cfg, nil
		}
		return nil, err
	}
	return parseAccountPoolAutoInspectConfig(raw), nil
}

func (s *AccountPoolAutoInspectService) UpdateConfig(ctx context.Context, cfg *AccountPoolAutoInspectConfig) (*AccountPoolAutoInspectConfig, error) {
	if s == nil || s.settings == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("invalid config")
	}
	normalizeAccountPoolAutoInspectConfig(cfg)
	if err := validateAccountPoolAutoInspectConfig(cfg); err != nil {
		return nil, err
	}
	if err := s.validateIQGroups(ctx, cfg); err != nil {
		return nil, err
	}
	raw, err := marshalAccountPoolAutoInspectConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, SettingKeyAccountPoolAutoInspectConfig, raw); err != nil {
		return nil, err
	}
	return parseAccountPoolAutoInspectConfig(raw), nil
}

func (s *AccountPoolAutoInspectService) validateIQGroups(ctx context.Context, cfg *AccountPoolAutoInspectConfig) error {
	if cfg == nil || s == nil || s.groups == nil {
		return nil
	}
	for _, id := range []int64{cfg.CorrectGroupID, cfg.IncorrectGroupID} {
		if id <= 0 {
			continue
		}
		if _, err := s.groups.GetByIDLite(ctx, id); err != nil {
			return fmt.Errorf("unknown group id %d", id)
		}
	}
	return nil
}

// RunOnce asks due accounts. force=true (the manual run) asks every eligible
// account now; the timer only asks accounts whose jittered due time has passed.
func (s *AccountPoolAutoInspectService) RunOnce(ctx context.Context, force bool) *AccountPoolAutoInspectStatus {
	startedAt := time.Now().UTC()
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, accountPoolAutoInspectRunTimeout)
	defer cancel()

	cfg, err := s.GetStoredConfig(runCtx)
	if err != nil {
		s.recordRun(startedAt, "", err)
		return s.statusFrom(cfg)
	}
	normalizeAccountPoolAutoInspectConfig(cfg)
	if !accountPoolAutoInspectReady(cfg) {
		return s.statusFrom(cfg)
	}
	if s.accounts == nil || s.quiz == nil {
		err := errors.New("auto inspect dependencies missing")
		s.recordRun(startedAt, "", err)
		return s.statusFrom(cfg)
	}
	if !force {
		release, ok := s.tryAcquireLeaderLock(runCtx)
		if !ok {
			return s.statusFrom(cfg)
		}
		if release != nil {
			defer release()
		}
	}

	accounts, err := s.accounts.ListAllWithFilters(runCtx, "", "", "", "", 0, "")
	if err != nil {
		s.recordRun(startedAt, "", err)
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] list accounts failed: %v", err)
		return s.statusFrom(cfg)
	}

	groups := s.loadIQGroups(runCtx, cfg)
	now := time.Now().UTC()
	var asked, correct, incorrect, untestable, moved, disabled int
	for i := range accounts {
		if asked >= accountPoolAutoInspectMaxPerRun {
			break
		}
		account := accounts[i]
		if !accountEligibleForPoolIQ(account, cfg, groups, now) {
			continue
		}
		if !force && !s.claimDue(runCtx, account.ID, cfg, now) {
			continue
		}
		asked++
		result := s.askAndMove(runCtx, &account, cfg, now)
		switch result {
		case AccountPoolIQResultCorrect:
			correct++
		case AccountPoolIQResultIncorrect:
			incorrect++
		default:
			untestable++
		}
		switch {
		case result == AccountPoolIQResultIncorrect && cfg.DisableFirstImportOnIncorrect && accountPoolIQFirstImport(&account):
			if s.disableFirstImport(runCtx, &account, now) {
				disabled++
			}
		case result == AccountPoolIQResultCorrect || result == AccountPoolIQResultIncorrect:
			target := cfg.IncorrectGroupID
			if result == AccountPoolIQResultCorrect {
				target = cfg.CorrectGroupID
			}
			if s.moveAccountGroup(runCtx, &account, cfg, target, now) {
				moved++
			}
		}
		if result == AccountPoolIQResultCorrect || result == AccountPoolIQResultIncorrect {
			s.markAccountPoolIQChecked(runCtx, &account, now)
		}
		if force {
			s.scheduleNext(runCtx, account.ID, cfg, now)
		}
	}

	result := fmt.Sprintf("asked=%d correct=%d incorrect=%d untestable=%d moved=%d disabled=%d", asked, correct, incorrect, untestable, moved, disabled)
	s.recordRun(startedAt, result, nil)
	return s.statusFrom(cfg)
}

func (s *AccountPoolAutoInspectService) askAndMove(ctx context.Context, account *Account, cfg *AccountPoolAutoInspectConfig, now time.Time) string {
	askCtx, cancel := context.WithTimeout(ctx, accountPoolAutoInspectAskTimeout)
	defer cancel()
	text, err := s.quiz.Ask(askCtx, account, cfg.Model, cfg.Question)
	if err != nil || strings.TrimSpace(text) == "" {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] account=%d untestable: %v", account.ID, err)
		return AccountPoolIQResultUntestable
	}
	if accountPoolIQAnswerMatches(text, cfg.Answer, cfg.FuzzyMatch) {
		return AccountPoolIQResultCorrect
	}
	return AccountPoolIQResultIncorrect
}

func (s *AccountPoolAutoInspectService) moveAccountGroup(ctx context.Context, account *Account, cfg *AccountPoolAutoInspectConfig, target int64, now time.Time) bool {
	next, changed := planAccountPoolIQGroups(account.GroupIDs, cfg.CorrectGroupID, cfg.IncorrectGroupID, target)
	if !changed {
		return false
	}
	if err := s.accounts.BindGroups(ctx, account.ID, next); err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] bind account=%d failed: %v", account.ID, err)
		return false
	}
	account.GroupIDs = next
	until := now.Add(time.Duration(cfg.PauseMinutes) * time.Minute)
	if err := s.accounts.SetTempUnschedulable(ctx, account.ID, until, accountPoolIQPauseReason); err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] pause account=%d failed: %v", account.ID, err)
	}
	return true
}

// accountPoolIQFirstImport is an imported account that has not completed a quiz.
func accountPoolIQFirstImport(account *Account) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(account.GetExtraString("imported_at")) == "" {
		return false
	}
	return strings.TrimSpace(account.GetExtraString(accountPoolAutoInspectCheckedAtExtra)) == ""
}

// disableFirstImport sets status=error and schedulable=false. It does not move groups.
func (s *AccountPoolAutoInspectService) disableFirstImport(ctx context.Context, account *Account, now time.Time) bool {
	if s == nil || s.accounts == nil || account == nil {
		return false
	}
	if err := s.accounts.SetError(ctx, account.ID, accountPoolIQFirstImportError); err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] disable account=%d failed: %v", account.ID, err)
		return false
	}
	account.Status = StatusError
	account.Schedulable = false
	account.ErrorMessage = accountPoolIQFirstImportError
	s.markAccountPoolIQChecked(ctx, account, now)
	return true
}

func (s *AccountPoolAutoInspectService) markAccountPoolIQChecked(ctx context.Context, account *Account, now time.Time) {
	if s == nil || s.accounts == nil || account == nil {
		return
	}
	if strings.TrimSpace(account.GetExtraString(accountPoolAutoInspectCheckedAtExtra)) != "" {
		return
	}
	stamp := now.UTC().Format(time.RFC3339)
	if err := s.accounts.UpdateExtra(ctx, account.ID, map[string]any{
		accountPoolAutoInspectCheckedAtExtra: stamp,
	}); err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] mark checked account=%d failed: %v", account.ID, err)
		return
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra[accountPoolAutoInspectCheckedAtExtra] = stamp
}

func accountEligibleForPoolIQ(account Account, cfg *AccountPoolAutoInspectConfig, groups map[int64]Group, now time.Time) bool {
	if account.ID <= 0 || account.IsCredentialShadow() {
		return false
	}
	if account.Status != StatusActive || !account.Schedulable {
		return false
	}
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now) {
		return false
	}
	if !account.IsOpenAIOAuthLike() {
		return false
	}
	if len(groups) == 0 {
		return true
	}
	for _, id := range []int64{cfg.CorrectGroupID, cfg.IncorrectGroupID} {
		group, ok := groups[id]
		if !ok {
			return false
		}
		if !groupAcceptsAccountPlatform(group.Platform, account.Platform) {
			return false
		}
	}
	return true
}

func (s *AccountPoolAutoInspectService) loadIQGroups(ctx context.Context, cfg *AccountPoolAutoInspectConfig) map[int64]Group {
	out := map[int64]Group{}
	if s == nil || s.groups == nil || cfg == nil {
		return out
	}
	for _, id := range []int64{cfg.CorrectGroupID, cfg.IncorrectGroupID} {
		if id <= 0 {
			continue
		}
		group, err := s.groups.GetByIDLite(ctx, id)
		if err != nil || group == nil {
			continue
		}
		out[id] = *group
	}
	return out
}

// claimDue reports whether this account should be asked now. A stored cooldown
// key means the previous jittered interval has not elapsed. Claiming writes the
// next interval immediately so two instances do not both ask.
func (s *AccountPoolAutoInspectService) claimDue(ctx context.Context, accountID int64, cfg *AccountPoolAutoInspectConfig, now time.Time) bool {
	if s.lockStore != nil {
		key := accountPoolAutoInspectDueKey(accountID)
		exists, err := s.lockStore.Exists(ctx, key)
		if err != nil || exists {
			return false
		}
		return s.storeDue(ctx, accountID, now.Add(accountPoolAutoInspectNextDelay(cfg.IntervalMinutes, cfg.JitterSeconds)))
	}
	s.dueMu.Lock()
	defer s.dueMu.Unlock()
	if due, ok := s.dueAt[accountID]; ok && now.Before(due) {
		return false
	}
	s.dueAt[accountID] = now.Add(accountPoolAutoInspectNextDelay(cfg.IntervalMinutes, cfg.JitterSeconds))
	return true
}

func (s *AccountPoolAutoInspectService) scheduleNext(ctx context.Context, accountID int64, cfg *AccountPoolAutoInspectConfig, now time.Time) {
	next := now.Add(accountPoolAutoInspectNextDelay(cfg.IntervalMinutes, cfg.JitterSeconds))
	if s.lockStore != nil {
		_ = s.storeDue(ctx, accountID, next)
		return
	}
	s.dueMu.Lock()
	s.dueAt[accountID] = next
	s.dueMu.Unlock()
}

func (s *AccountPoolAutoInspectService) storeDue(ctx context.Context, accountID int64, next time.Time) bool {
	ttl := time.Until(next)
	if ttl < time.Second {
		ttl = time.Second
	}
	if err := s.lockStore.SetCooldown(ctx, accountPoolAutoInspectDueKey(accountID), ttl); err != nil {
		return false
	}
	return true
}

func accountPoolAutoInspectDueKey(accountID int64) string {
	return accountPoolAutoInspectDueKeyPrefix + fmt.Sprintf("%d", accountID)
}

func (s *AccountPoolAutoInspectService) tryAcquireLeaderLock(ctx context.Context) (func(), bool) {
	if s == nil || s.lockStore == nil {
		return nil, true
	}
	ok, err := s.lockStore.Acquire(ctx, accountPoolAutoInspectLeaderLockKey, s.instanceID, accountPoolAutoInspectRunTimeout)
	if err != nil || !ok {
		if err != nil {
			logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] leader lock failed: %v", err)
		}
		return nil, false
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.lockStore.Release(releaseCtx, accountPoolAutoInspectLeaderLockKey, s.instanceID)
	}, true
}

func (s *AccountPoolAutoInspectService) recordRun(at time.Time, result string, err error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRunAt = at
	s.lastResult = result
	if err != nil {
		s.lastError = err.Error()
	} else {
		s.lastError = ""
	}
}

// accountPoolAutoInspectHTTPQuiz sends the candy question through the account's
// Codex responses endpoint and returns the completed answer text.
type accountPoolAutoInspectHTTPQuiz struct {
	upstream HTTPUpstream
	tokens   openAIAccessTokenReader
}

type openAIAccessTokenReader interface {
	GetAccessToken(ctx context.Context, account *Account) (string, error)
}

func (q *accountPoolAutoInspectHTTPQuiz) Ask(ctx context.Context, account *Account, model, question string) (string, error) {
	if q == nil || q.upstream == nil || account == nil {
		return "", errors.New("quiz not configured")
	}
	token := ""
	var err error
	if q.tokens != nil {
		token, err = q.tokens.GetAccessToken(ctx, account)
		if err != nil {
			return "", err
		}
	} else {
		token = account.GetOpenAIAccessToken()
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("missing access token")
	}
	payload, err := marshalTurnStateProbePayload(model, question)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatgptCodexURL, strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Host = "chatgpt.com"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("accept", "text/event-stream")
	setTurnStateProbeCookie(req.Header, account)
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

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := q.upstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("empty upstream response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	parsed := parseTurnStateProbeSSE(resp.Body)
	if strings.TrimSpace(parsed.Text) == "" {
		return "", errors.New("empty upstream text")
	}
	return parsed.Text, nil
}
