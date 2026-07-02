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
	"github.com/redis/go-redis/v9"
)

const (
	opsAccountErrorAlertJobName         = "ops_account_error_alert"
	opsAccountErrorAlertTimeout         = 45 * time.Second
	opsAccountErrorAlertDefaultBaseURL  = "https://api.telegram.org"
	opsAccountErrorAlertMessageMaxBytes = 3900
	opsAccountErrorAlertSkipLogInterval = time.Minute
)

var opsAccountErrorAlertReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

var accountErrorAlertFingerprintSpaceRE = regexp.MustCompile(`\s+`)

type OpsTelegramSender interface {
	SendMessage(ctx context.Context, botToken, chatID, text string) error
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

	redisClient *redis.Client
	cfg         *config.Config
	instanceID  string

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cooldownMu sync.Mutex
	cooldowns  map[string]time.Time

	skipLogMu sync.Mutex
	skipLogAt time.Time
}

func NewOpsAccountErrorAlertService(
	opsService *OpsService,
	opsRepo OpsRepository,
	sender OpsTelegramSender,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsAccountErrorAlertService {
	return &OpsAccountErrorAlertService{
		opsService:  opsService,
		opsRepo:     opsRepo,
		sender:      sender,
		redisClient: redisClient,
		cfg:         cfg,
		instanceID:  uuid.NewString(),
		cooldowns:   map[string]time.Time{},
		startOnce:   sync.Once{},
		stopOnce:    sync.Once{},
		cooldownMu:  sync.Mutex{},
		skipLogMu:   sync.Mutex{},
		skipLogAt:   time.Time{},
	}
}

func ProvideOpsAccountErrorAlertService(
	opsService *OpsService,
	opsRepo OpsRepository,
	sender OpsTelegramSender,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsAccountErrorAlertService {
	svc := NewOpsAccountErrorAlertService(opsService, opsRepo, sender, redisClient, cfg)
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
	candidates, err := s.opsRepo.ListAccountErrorAlertCandidates(ctx, &OpsAccountErrorAlertCandidateFilter{
		StartTime:     windowStart,
		EndTime:       windowEnd,
		MinErrorCount: cfg.MinErrorCount,
		Limit:         cfg.MaxAccountsPerAlert,
	})
	if err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_account_error_alert", "[OpsAccountErrorAlert] list candidates failed: %v", err)
		return
	}
	if len(candidates) == 0 {
		s.recordHeartbeatSuccess(runAt, time.Since(startedAt), "candidates=0 sent=0")
		return
	}

	eligible := s.filterCooldown(ctx, cfg, candidates)
	if len(eligible) == 0 {
		s.recordHeartbeatSuccess(runAt, time.Since(startedAt), fmt.Sprintf("candidates=%d sent=0 cooldown=all", len(candidates)))
		return
	}

	topUsers := []*OpsAccountErrorAlertTopUser{}
	if cfg.MaxUsersPerAlert > 0 {
		topUsers, err = s.opsRepo.ListAccountErrorAlertTopUsers(ctx, &OpsAccountErrorAlertTopUserFilter{
			StartTime:     windowStart,
			EndTime:       windowEnd,
			MinErrorCount: cfg.MinErrorCount,
			Limit:         cfg.MaxUsersPerAlert,
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
	if s.redisClient == nil {
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
	ok, err := s.redisClient.SetNX(ctx, key, s.instanceID, ttl).Result()
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
		_, _ = opsAccountErrorAlertReleaseScript.Run(releaseCtx, s.redisClient, []string{key}, s.instanceID).Result()
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
	if s.redisClient != nil {
		exists, err := s.redisClient.Exists(ctx, key).Result()
		if err == nil {
			return exists > 0
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
		if s.redisClient != nil {
			_ = s.redisClient.Set(ctx, key, "1", ttl).Err()
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
