package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	accountPoolAutoInspectTimeout         = 2 * time.Minute
	accountPoolAutoInspectHealthBatch     = 200
	accountPoolAutoInspectMaxActions      = 50
	accountPoolAutoInspectMessageMaxBytes = 3900
)

type poolAutoInspectAccounts interface {
	ListAllWithFilters(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, error)
	BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error
	Update(ctx context.Context, account *Account) error
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type poolAutoInspectSettings interface {
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type poolAutoInspectHealthReader interface {
	ListForAccounts(ctx context.Context, accountIDs []int64, window int) ([]AccountRequestHealthDTO, error)
}

type poolAutoInspectTelegramFallback interface {
	GetOpsAccountErrorAlertConfig(ctx context.Context) (*OpsAccountErrorAlertConfig, error)
}

type poolAutoInspectGroups interface {
	GetByIDLite(ctx context.Context, id int64) (*Group, error)
}

type AccountPoolAutoInspectService struct {
	settings   poolAutoInspectSettings
	accounts   poolAutoInspectAccounts
	health     poolAutoInspectHealthReader
	sender     OpsTelegramSender
	fallback   poolAutoInspectTelegramFallback
	lockStore  OpsAccountErrorAlertLockStore
	groups     poolAutoInspectGroups
	cfg        *config.Config
	instanceID string

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu         sync.Mutex
	lastRunAt  time.Time
	lastResult string
	lastError  string

	cooldownMu sync.Mutex
	cooldowns  map[string]time.Time
}

func NewAccountPoolAutoInspectService(
	settings poolAutoInspectSettings,
	accounts poolAutoInspectAccounts,
	health poolAutoInspectHealthReader,
	sender OpsTelegramSender,
	fallback poolAutoInspectTelegramFallback,
	lockStore OpsAccountErrorAlertLockStore,
	cfg *config.Config,
) *AccountPoolAutoInspectService {
	return &AccountPoolAutoInspectService{
		settings:   settings,
		accounts:   accounts,
		health:     health,
		sender:     sender,
		fallback:   fallback,
		lockStore:  lockStore,
		cfg:        cfg,
		instanceID: uuid.NewString(),
		cooldowns:  map[string]time.Time{},
	}
}

func ProvideAccountPoolAutoInspectService(
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	adminService AdminService,
	redisClient *redis.Client,
	concurrencyCache ConcurrencyCache,
	sender OpsTelegramSender,
	opsService *OpsService,
	lockStore OpsAccountErrorAlertLockStore,
	cfg *config.Config,
) *AccountPoolAutoInspectService {
	health := NewAccountRequestHealthService(
		NewAccountRequestHealthStore(redisClient, concurrencyCache),
		adminService,
	)
	svc := NewAccountPoolAutoInspectService(settingRepo, accountRepo, health, sender, opsService, lockStore, cfg)
	svc.groups = groupRepo
	svc.Start()
	return svc
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

	timer := time.NewTimer(s.getInterval())
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			s.RunOnce(context.Background(), false)
			timer.Reset(s.getInterval())
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountPoolAutoInspectService) getInterval() time.Duration {
	cfg := s.loadConfig(2 * time.Second)
	if cfg == nil || cfg.IntervalMinutes <= 0 {
		return time.Duration(AccountPoolAutoInspectDefaultInterval) * time.Minute
	}
	return time.Duration(cfg.IntervalMinutes) * time.Minute
}

func (s *AccountPoolAutoInspectService) GetConfig(ctx context.Context) (*AccountPoolAutoInspectStatus, error) {
	cfg, err := s.GetStoredConfig(ctx)
	if err != nil {
		return nil, err
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
	fallback := s.loadTelegramFallback(ctx)
	token, chatID := resolveAccountPoolAutoInspectTelegram(cfg, fallback)
	status.TelegramReady = token != "" && chatID != ""
	return status, nil
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
	cfg := parseAccountPoolAutoInspectConfig(raw)
	return cfg, nil
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
	if err := s.validateAddGroups(ctx, cfg.AddGroupIDs); err != nil {
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

func (s *AccountPoolAutoInspectService) RunOnce(ctx context.Context, force bool) *AccountPoolAutoInspectStatus {
	startedAt := time.Now().UTC()
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, accountPoolAutoInspectTimeout)
	defer cancel()

	cfg, err := s.GetStoredConfig(runCtx)
	if err != nil {
		s.recordRun(startedAt, "", err)
		return s.statusSnapshot(cfg, err)
	}
	normalizeAccountPoolAutoInspectConfig(cfg)
	if err := validateAccountPoolAutoInspectConfig(cfg); err != nil {
		s.recordRun(startedAt, "", err)
		return s.statusSnapshot(cfg, err)
	}
	if !force && !cfg.Enabled {
		return s.statusSnapshot(cfg, nil)
	}
	if s.accounts == nil || s.health == nil {
		err := errors.New("auto inspect dependencies missing")
		s.recordRun(startedAt, "", err)
		return s.statusSnapshot(cfg, err)
	}

	if !force {
		release, ok := s.tryAcquireLeaderLock(runCtx, cfg.IntervalMinutes)
		if !ok {
			s.recordRun(startedAt, "skipped_lock", nil)
			return s.statusSnapshot(cfg, nil)
		}
		if release != nil {
			defer release()
		}
	}

	accounts, err := s.accounts.ListAllWithFilters(runCtx, "", "", "", "", 0, "")
	if err != nil {
		s.recordRun(startedAt, "", err)
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] list accounts failed: %v", err)
		return s.statusSnapshot(cfg, err)
	}

	healthByID, err := s.loadHealth(runCtx, accounts)
	if err != nil {
		s.recordRun(startedAt, "", err)
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] load health failed: %v", err)
		return s.statusSnapshot(cfg, err)
	}

	var degraded, oauth401Notified int
	var degradeLines, oauth401Lines []string
	now := time.Now().UTC()
	addGroups := s.loadAddGroups(runCtx, cfg.AddGroupIDs)
	haveGroupDirectory := s.groups != nil
	for i := range accounts {
		account := accounts[i]
		if account.ID <= 0 || account.IsCredentialShadow() {
			continue
		}

		if cfg.NotifyOAuth401 {
			if stopped, reason := accountStoppedByOAuth401(account, now); stopped {
				if s.shouldNotifyOAuth401(runCtx, cfg, account.ID) {
					oauth401Lines = append(oauth401Lines, formatOAuth401Line(account, reason))
					oauth401Notified++
					s.markOAuth401Notified(runCtx, cfg, account.ID)
				}
			}
		}

		if degraded >= accountPoolAutoInspectMaxActions {
			continue
		}
		if !shouldInspectAccountHealth(account, now) {
			continue
		}
		dto, ok := healthByID[account.ID]
		if !ok {
			continue
		}
		verdict := evaluateAccountPoolAutoInspectHealth(dto, cfg.SuccessRateThreshold, cfg.MinSamples)
		if !verdict.Unhealthy {
			continue
		}
		plan := planAccountPoolAutoInspectRemediationWithGroups(account, cfg, addGroups, haveGroupDirectory)
		if !plan.HasWork {
			continue
		}
		if err := s.applyRemediation(runCtx, account, plan); err != nil {
			logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] remediate account=%d failed: %v", account.ID, err)
			continue
		}
		degraded++
		degradeLines = append(degradeLines, formatDegradeLine(account, verdict, plan))
	}

	s.notifyTelegram(runCtx, cfg, degradeLines, oauth401Lines)
	result := fmt.Sprintf("accounts=%d degraded=%d oauth401=%d", len(accounts), degraded, oauth401Notified)
	s.recordRun(startedAt, result, nil)
	return s.statusSnapshot(cfg, nil)
}

func shouldInspectAccountHealth(account Account, now time.Time) bool {
	if account.IsCredentialShadow() {
		return false
	}
	if account.Status != StatusActive {
		return false
	}
	if !account.Schedulable {
		return false
	}
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now) && looksLikeOAuth401(account.TempUnschedulableReason) {
		return false
	}
	return true
}

func (s *AccountPoolAutoInspectService) loadHealth(ctx context.Context, accounts []Account) (map[int64]AccountRequestHealthDTO, error) {
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account.ID > 0 && account.Status == StatusActive && !account.IsCredentialShadow() {
			ids = append(ids, account.ID)
		}
	}
	out := make(map[int64]AccountRequestHealthDTO, len(ids))
	for start := 0; start < len(ids); start += accountPoolAutoInspectHealthBatch {
		end := start + accountPoolAutoInspectHealthBatch
		if end > len(ids) {
			end = len(ids)
		}
		items, err := s.health.ListForAccounts(ctx, ids[start:end], DefaultRequestHealthWindow)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			out[item.AccountID] = item
		}
	}
	return out, nil
}

func (s *AccountPoolAutoInspectService) validateAddGroups(ctx context.Context, ids []int64) error {
	if s == nil || s.groups == nil || len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if _, err := s.groups.GetByIDLite(ctx, id); err != nil {
			return fmt.Errorf("unknown group id %d", id)
		}
	}
	return nil
}

func (s *AccountPoolAutoInspectService) loadAddGroups(ctx context.Context, ids []int64) map[int64]Group {
	out := make(map[int64]Group, len(ids))
	if s == nil || s.groups == nil {
		return out
	}
	for _, id := range ids {
		group, err := s.groups.GetByIDLite(ctx, id)
		if err != nil || group == nil {
			continue
		}
		out[id] = *group
	}
	return out
}

func (s *AccountPoolAutoInspectService) applyRemediation(ctx context.Context, account Account, plan poolAutoInspectRemediation) error {
	if plan.GroupsChanged {
		if err := s.accounts.BindGroups(ctx, account.ID, plan.GroupIDs); err != nil {
			return err
		}
	}
	if plan.MappingChanged {
		if account.IsCredentialShadow() {
			return nil
		}
		creds := cloneAccountCredentials(account.Credentials)
		creds["model_mapping"] = plan.Mapping
		if updater, ok := any(s.accounts).(accountCredentialsUpdater); ok {
			if err := updater.UpdateCredentials(ctx, account.ID, creds); err != nil {
				return err
			}
		} else {
			account.Credentials = creds
			if err := s.accounts.Update(ctx, &account); err != nil {
				return err
			}
		}
	}
	if plan.Close429Exemption {
		if err := s.accounts.UpdateExtra(ctx, account.ID, map[string]any{OAuth429CooldownEnforcedExtraKey: true}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AccountPoolAutoInspectService) notifyTelegram(ctx context.Context, cfg *AccountPoolAutoInspectConfig, degradeLines, oauth401Lines []string) {
	if s == nil || s.sender == nil {
		return
	}
	fallback := s.loadTelegramFallback(ctx)
	token, chatID := resolveAccountPoolAutoInspectTelegram(cfg, fallback)
	if token == "" || chatID == "" {
		return
	}
	text := buildPoolAutoInspectTelegram(degradeLines, oauth401Lines)
	if text == "" {
		return
	}
	if err := s.sender.SendMessage(ctx, token, chatID, text); err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] telegram send failed: %v", err)
	}
}

func buildPoolAutoInspectTelegram(degradeLines, oauth401Lines []string) string {
	var b strings.Builder
	if len(degradeLines) > 0 {
		b.WriteString("号池自动巡检：成功率过低，已降级\n")
		for _, line := range degradeLines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if len(oauth401Lines) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("号池自动巡检：401 已停止调度\n")
		for _, line := range oauth401Lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	text := strings.TrimSpace(b.String())
	if len(text) > accountPoolAutoInspectMessageMaxBytes {
		text = text[:accountPoolAutoInspectMessageMaxBytes]
	}
	return text
}

func formatDegradeLine(account Account, verdict poolAutoInspectHealthVerdict, plan poolAutoInspectRemediation) string {
	parts := []string{fmt.Sprintf("- %s (#%d) 成功率 %.0f%% (%d/%d)", account.Name, account.ID, verdict.Rate*100, verdict.OK, verdict.Samples)}
	if len(plan.AddedGroupIDs) > 0 {
		parts = append(parts, "加入分组 "+joinInt64(plan.AddedGroupIDs))
	}
	if len(plan.RemovedModels) > 0 {
		parts = append(parts, "移除 "+strings.Join(plan.RemovedModels, ","))
	}
	if plan.Close429Exemption {
		parts = append(parts, "关闭 429 豁免")
	}
	return strings.Join(parts, "；")
}

func formatOAuth401Line(account Account, reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 180 {
		reason = reason[:180]
	}
	if reason == "" {
		reason = "oauth_401"
	}
	return fmt.Sprintf("- %s (#%d) %s", account.Name, account.ID, reason)
}

func joinInt64(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ",")
}

func (s *AccountPoolAutoInspectService) shouldNotifyOAuth401(ctx context.Context, cfg *AccountPoolAutoInspectConfig, accountID int64) bool {
	key := accountPoolAutoInspectOAuth401CooldownKey + fmt.Sprintf("%d", accountID)
	if s.lockStore != nil {
		exists, err := s.lockStore.Exists(ctx, key)
		if err == nil {
			return !exists
		}
	}
	now := time.Now().UTC()
	s.cooldownMu.Lock()
	defer s.cooldownMu.Unlock()
	until, ok := s.cooldowns[key]
	return !ok || !now.Before(until)
}

func (s *AccountPoolAutoInspectService) markOAuth401Notified(ctx context.Context, cfg *AccountPoolAutoInspectConfig, accountID int64) {
	if cfg == nil {
		return
	}
	ttl := time.Duration(cfg.OAuth401CooldownMinutes) * time.Minute
	if ttl <= 0 {
		ttl = time.Hour
	}
	key := accountPoolAutoInspectOAuth401CooldownKey + fmt.Sprintf("%d", accountID)
	if s.lockStore != nil {
		_ = s.lockStore.SetCooldown(ctx, key, ttl)
		return
	}
	s.cooldownMu.Lock()
	s.cooldowns[key] = time.Now().UTC().Add(ttl)
	s.cooldownMu.Unlock()
}

func (s *AccountPoolAutoInspectService) tryAcquireLeaderLock(ctx context.Context, intervalMinutes int) (func(), bool) {
	if s == nil || s.lockStore == nil {
		return nil, true
	}
	ttl := time.Duration(intervalMinutes) * time.Minute
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}
	ok, err := s.lockStore.Acquire(ctx, accountPoolAutoInspectLeaderLockKey, s.instanceID, ttl)
	if err != nil {
		logger.LegacyPrintf("service.pool_auto_inspect", "[PoolAutoInspect] leader lock failed: %v", err)
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.lockStore.Release(releaseCtx, accountPoolAutoInspectLeaderLockKey, s.instanceID)
	}, true
}

func (s *AccountPoolAutoInspectService) loadConfig(timeout time.Duration) *AccountPoolAutoInspectConfig {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cfg, err := s.GetStoredConfig(ctx)
	if err != nil || cfg == nil {
		return defaultAccountPoolAutoInspectConfig()
	}
	return cfg
}

func (s *AccountPoolAutoInspectService) loadTelegramFallback(ctx context.Context) *OpsAccountErrorAlertConfig {
	if s == nil || s.fallback == nil {
		return nil
	}
	cfg, err := s.fallback.GetOpsAccountErrorAlertConfig(ctx)
	if err != nil {
		return nil
	}
	return cfg
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

func (s *AccountPoolAutoInspectService) statusSnapshot(cfg *AccountPoolAutoInspectConfig, err error) *AccountPoolAutoInspectStatus {
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
	if err != nil && status.LastError == "" {
		status.LastError = err.Error()
	}
	fallback := s.loadTelegramFallback(context.Background())
	token, chatID := resolveAccountPoolAutoInspectTelegram(cfg, fallback)
	status.TelegramReady = token != "" && chatID != ""
	return status
}
