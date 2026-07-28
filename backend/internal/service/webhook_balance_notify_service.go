package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultWebhookBalanceNotifyThreshold is used when the user leaves webhook threshold empty.
	DefaultWebhookBalanceNotifyThreshold = 10.0

	webhookBalanceNotifyHTTPTimeout = 10 * time.Second
	webhookBalanceNotifyTestRate    = 30 * time.Second
	webhookBalanceNotifyMaxBody     = 2048
)

// ErrWebhookBalanceNotifyDisabled means the user has not enabled webhook balance alerts.
var ErrWebhookBalanceNotifyDisabled = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_DISABLED")

// ErrWebhookBalanceNotifyURLInvalid means the webhook URL is missing or not allowed.
var ErrWebhookBalanceNotifyURLInvalid = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_URL_INVALID")

// ErrWebhookBalanceNotifySendFailed means the remote webhook rejected the request.
var ErrWebhookBalanceNotifySendFailed = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_SEND_FAILED")

// ErrWebhookBalanceNotifyRateLimited means test is called too frequently.
var ErrWebhookBalanceNotifyRateLimited = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_RATE_LIMITED")

// WebhookBalanceNotifyService sends balance-low alerts to external robots (WeCom primary).
type WebhookBalanceNotifyService struct {
	client *http.Client
	userRepo UserWebsocketUserReader // reuse narrow GetByID reader

	testMu   sync.Mutex
	testLast map[int64]time.Time
}

// NewWebhookBalanceNotifyService constructs the service.
func NewWebhookBalanceNotifyService(userRepo UserWebsocketUserReader) *WebhookBalanceNotifyService {
	return &WebhookBalanceNotifyService{
		client:   &http.Client{Timeout: webhookBalanceNotifyHTTPTimeout},
		userRepo: userRepo,
		testLast: make(map[int64]time.Time),
	}
}

// ResolveWebhookBalanceThreshold returns the effective threshold.
func ResolveWebhookBalanceThreshold(user *User) float64 {
	if user == nil {
		return 0
	}
	if user.WebhookBalanceNotifyThreshold != nil && *user.WebhookBalanceNotifyThreshold > 0 {
		return *user.WebhookBalanceNotifyThreshold
	}
	return DefaultWebhookBalanceNotifyThreshold
}

// ShouldPushWebhookBalanceAlert reports whether a balance cross should notify webhook.
func ShouldPushWebhookBalanceAlert(user *User) bool {
	if user == nil || !user.WebhookBalanceNotifyEnabled {
		return false
	}
	if strings.TrimSpace(user.WebhookBalanceNotifyURL) == "" {
		return false
	}
	return ResolveWebhookBalanceThreshold(user) > 0
}

// ValidateWebhookURL validates user-supplied webhook URLs.
// Allows WeCom robot URLs and generic https webhooks (no http, no localhost).
func ValidateWebhookURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrWebhookBalanceNotifyURLInvalid
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrWebhookBalanceNotifyURLInvalid
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return ErrWebhookBalanceNotifyURLInvalid
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local") {
		return ErrWebhookBalanceNotifyURLInvalid
	}
	return nil
}

// IsWeComWebhook reports whether URL looks like a WeCom group robot webhook.
func IsWeComWebhook(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "qyapi.weixin.qq.com" && strings.Contains(u.Path, "/cgi-bin/webhook/send")
}

// CheckWebhookBalanceAfterDeduction pushes webhook when balance first crosses below threshold.
// Independent from email BalanceNotifyService and browser WebSocket.
func (s *WebhookBalanceNotifyService) CheckWebhookBalanceAfterDeduction(ctx context.Context, user *User, oldBalance, cost float64) {
	if s == nil || !ShouldPushWebhookBalanceAlert(user) {
		return
	}
	threshold := ResolveWebhookBalanceThreshold(user)
	if threshold <= 0 {
		return
	}
	newBalance := oldBalance - cost
	if !crossedDownward(oldBalance, newBalance, threshold) {
		return
	}
	// fire-and-forget so billing path is not blocked
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in webhook balance notify", "recover", r)
			}
		}()
		notifyCtx, cancel := context.WithTimeout(context.Background(), webhookBalanceNotifyHTTPTimeout)
		defer cancel()
		if err := s.sendBalanceLow(notifyCtx, user, newBalance, threshold, false); err != nil {
			slog.Warn("webhook balance notify failed",
				"user_id", user.ID, "err", err)
		}
	}()
}

// SendTest sends a test message to the user's configured webhook (rate limited).
func (s *WebhookBalanceNotifyService) SendTest(ctx context.Context, userID int64) error {
	if s == nil || s.userRepo == nil {
		return ErrWebhookBalanceNotifyURLInvalid
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || !user.WebhookBalanceNotifyEnabled {
		return ErrWebhookBalanceNotifyDisabled
	}
	if err := ValidateWebhookURL(user.WebhookBalanceNotifyURL); err != nil {
		return err
	}

	s.testMu.Lock()
	last := s.testLast[userID]
	if time.Since(last) < webhookBalanceNotifyTestRate {
		s.testMu.Unlock()
		return ErrWebhookBalanceNotifyRateLimited
	}
	s.testLast[userID] = time.Now()
	s.testMu.Unlock()

	threshold := ResolveWebhookBalanceThreshold(user)
	return s.sendBalanceLow(ctx, user, user.Balance, threshold, true)
}

func (s *WebhookBalanceNotifyService) sendBalanceLow(ctx context.Context, user *User, balance, threshold float64, isTest bool) error {
	if user == nil {
		return ErrWebhookBalanceNotifyDisabled
	}
	if err := ValidateWebhookURL(user.WebhookBalanceNotifyURL); err != nil {
		return err
	}
	text := buildWebhookBalanceText(user, balance, threshold, isTest)
	return s.postWebhook(ctx, user.WebhookBalanceNotifyURL, text)
}

func buildWebhookBalanceText(user *User, balance, threshold float64, isTest bool) string {
	prefix := "【余额不足提醒】"
	if isTest {
		prefix = "【余额不足提醒·测试】"
	}
	email := ""
	if user != nil {
		email = strings.TrimSpace(user.Email)
	}
	if email == "" {
		email = fmt.Sprintf("user#%d", user.ID)
	}
	return fmt.Sprintf("%s\n账户: %s\n当前余额: $%.4f\n告警阈值: $%.4f\n请及时充值以免影响使用。",
		prefix, email, balance, threshold)
}

func (s *WebhookBalanceNotifyService) postWebhook(ctx context.Context, webhookURL, text string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	webhookURL = strings.TrimSpace(webhookURL)
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty webhook message")
	}

	var payload any
	if IsWeComWebhook(webhookURL) {
		payload = map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": text,
			},
		}
	} else {
		// Generic JSON webhook for custom receivers / future robots.
		payload = map[string]any{
			"msgtype": "text",
			"text":    text,
			"content": text,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := s.client
	if client == nil {
		client = &http.Client{Timeout: webhookBalanceNotifyHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookBalanceNotifySendFailed, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, webhookBalanceNotifyMaxBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status=%d body=%s", ErrWebhookBalanceNotifySendFailed, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	// WeCom returns HTTP 200 with errcode in JSON body.
	if IsWeComWebhook(webhookURL) {
		var wecom struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(raw, &wecom); err == nil && wecom.ErrCode != 0 {
			return fmt.Errorf("%w: wecom errcode=%d errmsg=%s", ErrWebhookBalanceNotifySendFailed, wecom.ErrCode, wecom.ErrMsg)
		}
	}
	return nil
}
