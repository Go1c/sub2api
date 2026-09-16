package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	opsAccountErrorAlertJobName         = "ops_account_error_alert"
	opsAccountErrorAlertTimeout         = 45 * time.Second
	opsAccountErrorAlertDefaultBaseURL  = "https://api.telegram.org"
	opsAccountErrorAlertMessageMaxBytes = 3900
	opsAccountErrorAlertSkipLogInterval = time.Minute
)

var accountErrorAlertFingerprintSpaceRE = regexp.MustCompile(`\s+`)

type OpsTelegramSender interface {
	SendMessage(ctx context.Context, botToken, chatID, text string) error
}

type OpsAccountErrorAlertLockStore interface {
	Acquire(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, value string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetCooldown(ctx context.Context, key string, ttl time.Duration) error
	GetInt(ctx context.Context, key string) (int64, error)
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type telegramOpsSender struct {
	client  *http.Client
	baseURL string
}

func NewTelegramOpsSender() *telegramOpsSender {
	return &telegramOpsSender{
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: opsAccountErrorAlertDefaultBaseURL,
	}
}

func (s *telegramOpsSender) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	botToken = strings.TrimSpace(botToken)
	chatID = strings.TrimSpace(chatID)
	text = strings.TrimSpace(text)
	if botToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}
	if chatID == "" {
		return fmt.Errorf("telegram chat id is required")
	}
	if text == "" {
		return fmt.Errorf("telegram message is required")
	}

	baseURL := strings.TrimRight(s.baseURL, "/")
	if baseURL == "" {
		baseURL = opsAccountErrorAlertDefaultBaseURL
	}
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/bot"+botToken+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := s.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("telegram sendMessage failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

type OpsAccountErrorAlertService struct {
	opsService *OpsService
	opsRepo    OpsRepository
	sender     OpsTelegramSender

	lockStore  OpsAccountErrorAlertLockStore
	cfg        *config.Config
	instanceID string

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cooldownMu sync.Mutex
	cooldowns  map[string]time.Time

	skipLogMu sync.Mutex
	skipLogAt time.Time

	sendMu     sync.Mutex
	sendCounts map[string]sendWindowCounter
}

type sendWindowCounter struct {
	until time.Time
	n     int64
}

func NewOpsAccountErrorAlertService(
	opsService *OpsService,
	opsRepo OpsRepository,
	sender OpsTelegramSender,
	lockStore OpsAccountErrorAlertLockStore,
	cfg *config.Config,
) *OpsAccountErrorAlertService {
	return &OpsAccountErrorAlertService{
		opsService: opsService,
		opsRepo:    opsRepo,
		sender:     sender,
		lockStore:  lockStore,
		cfg:        cfg,
		instanceID: uuid.NewString(),
		cooldowns:  map[string]time.Time{},
		sendCounts: map[string]sendWindowCounter{},
		startOnce:  sync.Once{},
		stopOnce:   sync.Once{},
		cooldownMu: sync.Mutex{},
		skipLogMu:  sync.Mutex{},
		skipLogAt:  time.Time{},
	}
}

func ProvideOpsAccountErrorAlertService(
	opsService *OpsService,
	opsRepo OpsRepository,
	sender OpsTelegramSender,
	lockStore OpsAccountErrorAlertLockStore,
	cfg *config.Config,
) *OpsAccountErrorAlertService {
	svc := NewOpsAccountErrorAlertService(opsService, opsRepo, sender, lockStore, cfg)
	svc.Start()
	return svc
}

func (s *OpsAccountErrorAlertService) Start() {
	if s == nil {
		return
	}
	if s.cfg != nil && !s.cfg.Ops.Enabled {
		return
	}
	if s.opsService == nil || s.opsRepo == nil || s.sender == nil {
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

func (s *OpsAccountErrorAlertService) Stop() {
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

func (s *OpsAccountErrorAlertService) run() {
	defer s.wg.Done()

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			interval := s.getInterval()
			s.runOnce()
			timer.Reset(interval)
		case <-s.stopCh:
			return
		}
	}
}

func (s *OpsAccountErrorAlertService) getInterval() time.Duration {
	cfg := s.loadConfig(2 * time.Second)
	if cfg == nil || cfg.IntervalMinutes <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(cfg.IntervalMinutes) * time.Minute
}

func (s *OpsAccountErrorAlertService) runOnce() {
	if s == nil || s.opsService == nil || s.opsRepo == nil || s.sender == nil {
		return
	}
	if s.cfg != nil && !s.cfg.Ops.Enabled {
		return
	}

	startedAt := time.Now().UTC()
	runAt := startedAt

	ctx, cancel := context.WithTimeout(context.Background(), opsAccountErrorAlertTimeout)
	defer cancel()

	if !s.opsService.IsMonitoringEnabled(ctx) {
		return
	}

	cfg, err := s.opsService.GetOpsAccountErrorAlertConfig(ctx)
	if err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] load config failed: %v", err)
		return
	}
	normalizeOpsAccountErrorAlertConfig(cfg)
	if err := validateOpsAccountErrorAlertConfig(cfg); err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] invalid config: %v", err)
		return
	}
	if !cfg.Enabled {
		return
	}

	release, ok := s.tryAcquireLeaderLock(ctx, cfg.DistributedLock)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	windowEnd := time.Now().UTC().Truncate(time.Minute)
	if windowEnd.IsZero() {
		windowEnd = time.Now().UTC()
	}
	windowStart := windowEnd.Add(-time.Duration(cfg.WindowMinutes) * time.Minute)
	candidates, err := s.collectAccountErrorAlertItems(ctx, cfg, windowStart, windowEnd)
	if err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] list candidates failed: %v", err)
		return
	}
	if len(candidates) == 0 {
		s.recordHeartbeatSuccess(runAt, time.Since(startedAt), "candidates=0 sent=0")
		return
	}

	eligible, sendKeys := s.filterAccountErrorAlertItems(ctx, cfg, candidates)
	if len(eligible) == 0 {
		s.recordHeartbeatSuccess(runAt, time.Since(startedAt), fmt.Sprintf("candidates=%d sent=0 cooldown=all", len(candidates)))
		return
	}

	topUsers := []*OpsAccountErrorAlertTopUser{}
	if cfg.MaxUsersPerAlert > 0 {
		topUsers, err = s.opsRepo.ListAccountErrorAlertTopUsers(ctx, &OpsAccountErrorAlertTopUserFilter{
			StartTime:          windowStart,
			EndTime:            windowEnd,
			MinErrorCount:      cfg.MinErrorCount,
			Limit:              cfg.MaxUsersPerAlert,
			UseAccountKeywords: true,
		})
		if err != nil {
			s.recordHeartbeatError(runAt, time.Since(startedAt), err)
			logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] list top users failed: %v", err)
			return
		}
	}

	message := buildOpsAccountErrorAlertMessage(windowStart, windowEnd, cfg.MinErrorCount, cfg.CooldownMinutes, eligible, topUsers)
	if err := s.sender.SendMessage(ctx, cfg.TelegramBotToken, cfg.TelegramChatID, message); err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] send telegram failed: %v", err)
		return
	}

	s.markCooldown(ctx, cfg, eligible)
	s.markSends(ctx, sendKeys)
	result := truncateString(fmt.Sprintf("candidates=%d sent=%d top_users=%d window=%s..%s", len(candidates), len(eligible), len(topUsers), windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339)), 2048)
	s.recordHeartbeatSuccess(runAt, time.Since(startedAt), result)
}

func (s *OpsAccountErrorAlertService) loadConfig(timeout time.Duration) *OpsAccountErrorAlertConfig {
	if s == nil || s.opsService == nil {
		return defaultOpsAccountErrorAlertConfig()
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cfg, err := s.opsService.GetOpsAccountErrorAlertConfig(ctx)
	if err != nil || cfg == nil {
		return defaultOpsAccountErrorAlertConfig()
	}
	normalizeOpsAccountErrorAlertConfig(cfg)
	return cfg
}

func (s *OpsAccountErrorAlertService) tryAcquireLeaderLock(ctx context.Context, lock OpsDistributedLockSettings) (func(), bool) {
	if !lock.Enabled {
		return nil, true
	}
	if s.lockStore == nil {
		return nil, true
	}
	key := strings.TrimSpace(lock.Key)
	if key == "" {
		key = opsAccountErrorAlertLeaderLockKeyDefault
	}
	ttl := time.Duration(lock.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = opsAccountErrorAlertLeaderLockTTLDefault
	}
	ok, err := s.lockStore.Acquire(ctx, key, s.instanceID, ttl)
	if err != nil {
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] leader lock SetNX failed; skipping this cycle: %v", err)
		return nil, false
	}
	if !ok {
		s.maybeLogSkip(key)
		return nil, false
	}
	return func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer releaseCancel()
		_ = s.lockStore.Release(releaseCtx, key, s.instanceID)
	}, true
}

func (s *OpsAccountErrorAlertService) maybeLogSkip(key string) {
	s.skipLogMu.Lock()
	defer s.skipLogMu.Unlock()
	now := time.Now()
	if !s.skipLogAt.IsZero() && now.Sub(s.skipLogAt) < opsAccountErrorAlertSkipLogInterval {
		return
	}
	s.skipLogAt = now
	logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] another instance holds leader lock %s; skipping", key)
}

type accountErrorAlertSendMark struct {
	key string
	ttl time.Duration
}

func (s *OpsAccountErrorAlertService) collectAccountErrorAlertItems(ctx context.Context, cfg *OpsAccountErrorAlertConfig, windowStart, windowEnd time.Time) ([]*OpsAccountErrorAlertCandidate, error) {
	if s == nil || s.opsRepo == nil || cfg == nil {
		return nil, nil
	}
	defaultItems, err := s.opsRepo.ListAccountErrorAlertCandidates(ctx, &OpsAccountErrorAlertCandidateFilter{
		StartTime:          windowStart,
		EndTime:            windowEnd,
		MinErrorCount:      cfg.MinErrorCount,
		Limit:              cfg.MaxAccountsPerAlert,
		UseAccountKeywords: true,
	})
	if err != nil {
		return nil, err
	}

	ruleAccountIDs, err := s.opsRepo.ListAccountIDsWithErrorAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	ids := append([]int64{}, ruleAccountIDs...)
	for _, item := range defaultItems {
		if item != nil && item.AccountID > 0 {
			ids = append(ids, item.AccountID)
		}
	}
	settingsByID := map[int64]AccountErrorAlertSettings{}
	if len(ids) > 0 {
		settingsByID, err = s.opsRepo.GetAccountErrorAlertSettings(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	merged := make([]*OpsAccountErrorAlertCandidate, 0, len(defaultItems)+len(ruleAccountIDs))
	seen := map[int64]struct{}{}

	for _, accountID := range ruleAccountIDs {
		settings := settingsByID[accountID]
		if !settings.Enabled {
			continue
		}
		for _, rule := range settings.Rules {
			ruleWindow := resolveAccountErrorAlertRuleWindow(cfg, rule)
			ruleMin := resolveAccountErrorAlertRuleMinCount(cfg, rule)
			ruleStartAt := windowEnd.Add(-time.Duration(ruleWindow) * time.Minute)
			if !ruleStartAt.Before(windowEnd) {
				ruleStartAt = windowEnd.Add(-time.Minute)
			}
			items, listErr := s.opsRepo.ListAccountErrorAlertCandidates(ctx, &OpsAccountErrorAlertCandidateFilter{
				StartTime:     ruleStartAt,
				EndTime:       windowEnd,
				MinErrorCount: ruleMin,
				Limit:         cfg.MaxAccountsPerAlert,
				AccountID:     accountID,
				Keyword:       strings.TrimSpace(rule.Keyword),
			})
			if listErr != nil {
				return nil, listErr
			}
			for _, item := range items {
				if item == nil || item.AccountID <= 0 {
					continue
				}
				if _, ok := seen[item.AccountID]; ok {
					continue
				}
				seen[item.AccountID] = struct{}{}
				merged = append(merged, item)
			}
		}
	}

	for _, item := range defaultItems {
		if item == nil || item.AccountID <= 0 {
			continue
		}
		if _, ok := seen[item.AccountID]; ok {
			continue
		}
		if coveredByAccountErrorAlertRule(settingsByID[item.AccountID], item) {
			continue
		}
		seen[item.AccountID] = struct{}{}
		merged = append(merged, item)
	}

	if cfg.MaxAccountsPerAlert > 0 && len(merged) > cfg.MaxAccountsPerAlert {
		merged = merged[:cfg.MaxAccountsPerAlert]
	}
	return merged, nil
}

func coveredByAccountErrorAlertRule(settings AccountErrorAlertSettings, item *OpsAccountErrorAlertCandidate) bool {
	if !settings.Enabled || item == nil {
		return false
	}
	for _, rule := range settings.Rules {
		if accountErrorAlertRuleCoversItem(rule, item) {
			return true
		}
	}
	return false
}

func (s *OpsAccountErrorAlertService) filterAccountErrorAlertItems(ctx context.Context, cfg *OpsAccountErrorAlertConfig, items []*OpsAccountErrorAlertCandidate) ([]*OpsAccountErrorAlertCandidate, []accountErrorAlertSendMark) {
	if len(items) == 0 {
		return items, nil
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item != nil && item.AccountID > 0 {
			ids = append(ids, item.AccountID)
		}
	}
	settingsByID := map[int64]AccountErrorAlertSettings{}
	if s.opsRepo != nil && len(ids) > 0 {
		if loaded, err := s.opsRepo.GetAccountErrorAlertSettings(ctx, ids); err == nil {
			settingsByID = loaded
		}
	}

	out := make([]*OpsAccountErrorAlertCandidate, 0, len(items))
	marks := make([]accountErrorAlertSendMark, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		settings := settingsByID[item.AccountID]
		matchedRule, ok := matchingAccountErrorAlertRule(settings, item)
		if ok {
			maxSends := resolveAccountErrorAlertRuleMaxSends(matchedRule)
			ttl := time.Duration(resolveAccountErrorAlertRuleWindow(cfg, matchedRule)) * time.Minute
			key := accountErrorAlertSendKey(item.AccountID, matchedRule.Keyword)
			if !s.canSend(ctx, key, maxSends) {
				continue
			}
			out = append(out, item)
			marks = append(marks, accountErrorAlertSendMark{key: key, ttl: ttl})
			continue
		}
		if s.isCoolingDown(ctx, cfg, item) {
			continue
		}
		out = append(out, item)
	}
	return out, marks
}

func matchingAccountErrorAlertRule(settings AccountErrorAlertSettings, item *OpsAccountErrorAlertCandidate) (AccountErrorAlertRule, bool) {
	if !settings.Enabled || item == nil {
		return AccountErrorAlertRule{}, false
	}
	for _, rule := range settings.Rules {
		if accountErrorAlertRuleCoversItem(rule, item) {
			return rule, true
		}
	}
	return AccountErrorAlertRule{}, false
}

func accountErrorAlertSendKey(accountID int64, keyword string) string {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		kw = "*"
	}
	return fmt.Sprintf("ops:account_error_alert:sends:%d:%s", accountID, kw)
}

func (s *OpsAccountErrorAlertService) canSend(ctx context.Context, key string, maxSends int) bool {
	if maxSends <= 0 {
		maxSends = 1
	}
	if key == "" {
		return true
	}
	if s.lockStore != nil {
		n, err := s.lockStore.GetInt(ctx, key)
		if err == nil {
			return n < int64(maxSends)
		}
	}
	now := time.Now().UTC()
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.sendCounts == nil {
		s.sendCounts = map[string]sendWindowCounter{}
	}
	cur, ok := s.sendCounts[key]
	if !ok || now.After(cur.until) {
		return true
	}
	return cur.n < int64(maxSends)
}

func (s *OpsAccountErrorAlertService) markSends(ctx context.Context, marks []accountErrorAlertSendMark) {
	if len(marks) == 0 {
		return
	}
	now := time.Now().UTC()
	for _, mark := range marks {
		if mark.key == "" {
			continue
		}
		ttl := mark.ttl
		if ttl <= 0 {
			ttl = 10 * time.Minute
		}
		if s.lockStore != nil {
			_, _ = s.lockStore.Incr(ctx, mark.key, ttl)
			continue
		}
		s.sendMu.Lock()
		if s.sendCounts == nil {
			s.sendCounts = map[string]sendWindowCounter{}
		}
		cur := s.sendCounts[mark.key]
		if now.After(cur.until) {
			cur = sendWindowCounter{until: now.Add(ttl), n: 0}
		}
		cur.n++
		s.sendCounts[mark.key] = cur
		s.sendMu.Unlock()
	}
}

func (s *OpsAccountErrorAlertService) filterCooldown(ctx context.Context, cfg *OpsAccountErrorAlertConfig, items []*OpsAccountErrorAlertCandidate) []*OpsAccountErrorAlertCandidate {
	if s == nil || cfg == nil || len(items) == 0 || cfg.CooldownMinutes <= 0 {
		return items
	}
	out := make([]*OpsAccountErrorAlertCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if !s.isCoolingDown(ctx, cfg, item) {
			out = append(out, item)
		}
	}
	return out
}

func (s *OpsAccountErrorAlertService) isCoolingDown(ctx context.Context, cfg *OpsAccountErrorAlertConfig, item *OpsAccountErrorAlertCandidate) bool {
	key := accountErrorAlertCooldownKey(item)
	if key == "" {
		return false
	}
	if s.lockStore != nil {
		exists, err := s.lockStore.Exists(ctx, key)
		if err == nil {
			return exists
		}
	}

	now := time.Now().UTC()
	s.cooldownMu.Lock()
	defer s.cooldownMu.Unlock()
	until, ok := s.cooldowns[key]
	if !ok {
		return false
	}
	if now.Before(until) {
		return true
	}
	delete(s.cooldowns, key)
	return false
}

func (s *OpsAccountErrorAlertService) markCooldown(ctx context.Context, cfg *OpsAccountErrorAlertConfig, items []*OpsAccountErrorAlertCandidate) {
	if s == nil || cfg == nil || cfg.CooldownMinutes <= 0 || len(items) == 0 {
		return
	}
	ttl := time.Duration(cfg.CooldownMinutes) * time.Minute
	now := time.Now().UTC()
	for _, item := range items {
		if item == nil {
			continue
		}
		key := accountErrorAlertCooldownKey(item)
		if key == "" {
			continue
		}
		if s.lockStore != nil {
			_ = s.lockStore.SetCooldown(ctx, key, ttl)
			continue
		}
		s.cooldownMu.Lock()
		s.cooldowns[key] = now.Add(ttl)
		s.cooldownMu.Unlock()
	}
}

func accountErrorAlertCooldownKey(item *OpsAccountErrorAlertCandidate) string {
	if item == nil || item.AccountID <= 0 {
		return ""
	}
	fp := accountErrorAlertFingerprint(item)
	if fp == "" {
		return ""
	}
	return "ops:account_error_alert:cooldown:" + fp
}

func accountErrorAlertFingerprint(item *OpsAccountErrorAlertCandidate) string {
	if item == nil || item.AccountID <= 0 {
		return ""
	}
	msg := strings.ToLower(strings.TrimSpace(item.ErrorMessage))
	msg = accountErrorAlertFingerprintSpaceRE.ReplaceAllString(msg, " ")
	raw := fmt.Sprintf("%d|%d|%s", item.AccountID, item.StatusCode, msg)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func buildOpsAccountErrorAlertMessage(start, end time.Time, minErrorCount int, cooldownMinutes int, items []*OpsAccountErrorAlertCandidate, topUsers []*OpsAccountErrorAlertTopUser) string {
	windowMinutes := int(end.Sub(start).Minutes())
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	loc := time.Local
	startLocal := start.In(loc)
	endLocal := end.In(loc)

	var b strings.Builder
	fmt.Fprintf(&b, "[账号异常] 最近 %d 分钟有 %d 个账号异常\n\n", windowMinutes, len(items))
	fmt.Fprintf(&b, "时间窗口：%s - %s\n", startLocal.Format("15:04"), endLocal.Format("15:04"))
	fmt.Fprintf(&b, "触发条件：单账号异常 >= %d 次\n\n", minErrorCount)
	fmt.Fprintf(&b, "%-24s %-6s %-6s %s\n", "账号", "错误", "次数", "最近时间")
	for _, item := range items {
		if item == nil {
			continue
		}
		name := truncateString(strings.TrimSpace(item.AccountName), 24)
		if name == "" {
			name = fmt.Sprintf("Account #%d", item.AccountID)
		}
		fmt.Fprintf(&b, "%-24s %-6d %-6d %s\n", name, item.StatusCode, item.ErrorCount, item.LatestAt.In(loc).Format("15:04:05"))
	}

	fmt.Fprintf(&b, "\n主要错误信息：\n")
	for _, item := range items {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(item.AccountName)
		if name == "" {
			name = fmt.Sprintf("Account #%d", item.AccountID)
		}
		msg := strings.TrimSpace(item.ErrorMessage)
		if msg == "" {
			msg = "无错误信息"
		}
		fmt.Fprintf(&b, "%s：%s\n", truncateString(name, 32), truncateString(msg, 180))
	}
	emailUsers := make([]*OpsAccountErrorAlertTopUser, 0, len(topUsers))
	for _, user := range topUsers {
		if user == nil {
			continue
		}
		if strings.TrimSpace(user.UserEmail) == "" {
			continue
		}
		emailUsers = append(emailUsers, user)
	}
	if len(emailUsers) > 0 {
		fmt.Fprintf(&b, "\n影响用户邮箱 Top %d：\n", len(emailUsers))
		fmt.Fprintf(&b, "%-32s %s\n", "邮箱", "次数")
		for _, user := range emailUsers {
			label := strings.TrimSpace(user.UserEmail)
			fmt.Fprintf(&b, "%-32s %d\n", truncateString(label, 32), user.ErrorCount)
		}
	}
	if cooldownMinutes > 0 {
		fmt.Fprintf(&b, "\n降噪：同账号同错误 %d 分钟内不重复推送。", cooldownMinutes)
	}
	return truncateString(b.String(), opsAccountErrorAlertMessageMaxBytes)
}

func (s *OpsAccountErrorAlertService) recordHeartbeatSuccess(runAt time.Time, duration time.Duration, result string) {
	if s == nil || s.opsRepo == nil {
		return
	}
	successAt := time.Now().UTC()
	durationMs := duration.Milliseconds()
	result = truncateString(result, 2048)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsAccountErrorAlertJobName,
		LastRunAt:      &runAt,
		LastSuccessAt:  &successAt,
		LastDurationMs: &durationMs,
		LastResult:     &result,
	})
}

func (s *OpsAccountErrorAlertService) recordHeartbeatError(runAt time.Time, duration time.Duration, err error) {
	if s == nil || s.opsRepo == nil || err == nil {
		return
	}
	errorAt := time.Now().UTC()
	durationMs := duration.Milliseconds()
	msg := truncateString(err.Error(), 2048)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsAccountErrorAlertJobName,
		LastRunAt:      &runAt,
		LastErrorAt:    &errorAt,
		LastError:      &msg,
		LastDurationMs: &durationMs,
	})
}
