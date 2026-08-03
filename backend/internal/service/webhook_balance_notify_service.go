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

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

const (
	// DefaultWebhookBalanceNotifyThreshold is used when the user leaves webhook threshold empty.
	DefaultWebhookBalanceNotifyThreshold = 10.0

	webhookBalanceNotifyHTTPTimeout = 10 * time.Second
	webhookBalanceNotifyTestRate    = 30 * time.Second
	webhookBalanceNotifyMaxBody     = 2048
)

// ErrWebhookBalanceNotifyDisabled means the user has not enabled webhook notify.
var ErrWebhookBalanceNotifyDisabled = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_DISABLED")

// ErrWebhookBalanceNotifyURLInvalid means the webhook URL is missing or not allowed.
var ErrWebhookBalanceNotifyURLInvalid = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_URL_INVALID")

// ErrWebhookBalanceNotifySendFailed means the remote webhook rejected the request.
var ErrWebhookBalanceNotifySendFailed = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_SEND_FAILED")

// ErrWebhookBalanceNotifyRateLimited means test is called too frequently.
var ErrWebhookBalanceNotifyRateLimited = fmt.Errorf("WEBHOOK_BALANCE_NOTIFY_RATE_LIMITED")

// WebhookUserReader loads users for preference checks.
type WebhookUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

// WebhookAnnouncementUserLister lists users eligible for announcement webhook fan-out.
type WebhookAnnouncementUserLister interface {
	ListIDsWithWebhookAnnouncementNotify(ctx context.Context) ([]int64, error)
}

// WebhookBalanceNotifyService sends HTTPS webhook notifications (balance / site-message / announcement).
type WebhookBalanceNotifyService struct {
	client   *http.Client
	userRepo WebhookUserReader

	testMu   sync.Mutex
	testLast map[int64]time.Time
}

// NewWebhookBalanceNotifyService constructs the service.
func NewWebhookBalanceNotifyService(userRepo WebhookUserReader) *WebhookBalanceNotifyService {
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

func webhookReady(user *User) bool {
	if user == nil || !user.WebhookBalanceNotifyEnabled {
		return false
	}
	return strings.TrimSpace(user.WebhookBalanceNotifyURL) != ""
}

// ShouldPushWebhookBalanceAlert reports whether a balance cross should notify webhook.
func ShouldPushWebhookBalanceAlert(user *User) bool {
	if !webhookReady(user) {
		return false
	}
	return ResolveWebhookBalanceThreshold(user) > 0
}

// ShouldPushWebhookSiteMessage reports whether a new site message should notify webhook.
func ShouldPushWebhookSiteMessage(user *User) bool {
	return webhookReady(user) && user.WebhookSiteMessageNotifyEnabled
}

// ShouldPushWebhookAnnouncement reports whether a new announcement should notify webhook.
func ShouldPushWebhookAnnouncement(user *User) bool {
	return webhookReady(user) && user.WebhookAnnouncementNotifyEnabled
}

// ValidateWebhookURL validates user-supplied webhook URLs (https only, no localhost).
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

// IsWeComStyleWebhook reports whether URL uses WeCom robot path (affects JSON payload shape only).
func IsWeComStyleWebhook(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "qyapi.weixin.qq.com" && strings.Contains(u.Path, "/cgi-bin/webhook/send")
}

// CheckWebhookBalanceAfterDeduction posts when balance first crosses below threshold.
func (s *WebhookBalanceNotifyService) CheckWebhookBalanceAfterDeduction(_ context.Context, user *User, oldBalance, cost float64) {
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in webhook balance notify", "recover", r)
			}
		}()
		notifyCtx, cancel := context.WithTimeout(context.Background(), webhookBalanceNotifyHTTPTimeout)
		defer cancel()
		text := buildWebhookBalanceText(user, newBalance, threshold, false)
		if err := s.postWebhook(notifyCtx, user.WebhookBalanceNotifyURL, text); err != nil {
			slog.Warn("webhook balance notify failed", "user_id", user.ID, "err", err)
		}
	}()
}

// NotifySiteMessage posts when a user receives a new site message.
func (s *WebhookBalanceNotifyService) NotifySiteMessage(ctx context.Context, recipientID int64, messageID int64, subject string) {
	if s == nil || recipientID <= 0 {
		return
	}
	user, err := s.loadUser(ctx, recipientID)
	if err != nil || !ShouldPushWebhookSiteMessage(user) {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in webhook site-message notify", "recover", r)
			}
		}()
		notifyCtx, cancel := context.WithTimeout(context.Background(), webhookBalanceNotifyHTTPTimeout)
		defer cancel()
		subj := strings.TrimSpace(subject)
		if subj == "" {
			subj = "(no subject)"
		}
		text := fmt.Sprintf("【新站内信】\n账户: %s\n主题: %s\n消息ID: %d",
			webhookUserLabel(user), subj, messageID)
		if err := s.postWebhook(notifyCtx, user.WebhookBalanceNotifyURL, text); err != nil {
			slog.Warn("webhook site-message notify failed", "user_id", recipientID, "err", err)
		}
	}()
}

// NotifyAnnouncementPublished fans out to users with webhook announcement notify enabled.
func (s *WebhookBalanceNotifyService) NotifyAnnouncementPublished(ctx context.Context, a *Announcement) {
	if s == nil || s.userRepo == nil || a == nil || a.Status != AnnouncementStatusActive {
		return
	}
	lister, ok := s.userRepo.(WebhookAnnouncementUserLister)
	if !ok {
		return
	}
	ids, err := lister.ListIDsWithWebhookAnnouncementNotify(ctx)
	if err != nil {
		slog.Warn("webhook announcement list users failed", "err", err)
		return
	}
	for _, id := range ids {
		user, err := s.loadUser(ctx, id)
		if err != nil || !ShouldPushWebhookAnnouncement(user) {
			continue
		}
		if !announcementLikelyVisible(user, a) {
			continue
		}
		uid, title, aid := id, a.Title, a.ID
		u := user
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in webhook announcement notify", "recover", r)
				}
			}()
			notifyCtx, cancel := context.WithTimeout(context.Background(), webhookBalanceNotifyHTTPTimeout)
			defer cancel()
			t := strings.TrimSpace(title)
			if t == "" {
				t = "(untitled)"
			}
			text := fmt.Sprintf("【新公告】\n账户: %s\n标题: %s\n公告ID: %d",
				webhookUserLabel(u), t, aid)
			if err := s.postWebhook(notifyCtx, u.WebhookBalanceNotifyURL, text); err != nil {
				slog.Warn("webhook announcement notify failed", "user_id", uid, "err", err)
			}
		}()
	}
}

func announcementLikelyVisible(user *User, a *Announcement) bool {
	if user == nil || a == nil {
		return false
	}
	t := domain.AnnouncementTargeting(a.Targeting)
	if len(t.AnyOf) == 0 {
		return true
	}
	return t.Matches(user.Balance, map[int64]struct{}{})
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
	text := buildWebhookBalanceText(user, user.Balance, threshold, true)
	return s.postWebhook(ctx, user.WebhookBalanceNotifyURL, text)
}

func (s *WebhookBalanceNotifyService) loadUser(ctx context.Context, userID int64) (*User, error) {
	if s.userRepo == nil {
		return nil, ErrUserNotFound
	}
	return s.userRepo.GetByID(ctx, userID)
}

func webhookUserLabel(user *User) string {
	if user == nil {
		return "unknown"
	}
	email := strings.TrimSpace(user.Email)
	if email != "" {
		return email
	}
	return fmt.Sprintf("user#%d", user.ID)
}

func buildWebhookBalanceText(user *User, balance, threshold float64, isTest bool) string {
	prefix := "【余额不足提醒】"
	if isTest {
		prefix = "【Webhook 测试】"
	}
	return fmt.Sprintf("%s\n账户: %s\n当前余额: $%.4f\n告警阈值: $%.4f\n请及时充值以免影响使用。",
		prefix, webhookUserLabel(user), balance, threshold)
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
	if IsWeComStyleWebhook(webhookURL) {
		payload = map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": text,
			},
		}
	} else {
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
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, webhookBalanceNotifyMaxBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status=%d body=%s", ErrWebhookBalanceNotifySendFailed, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if IsWeComStyleWebhook(webhookURL) {
		var wecom struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(raw, &wecom); err == nil && wecom.ErrCode != 0 {
			return fmt.Errorf("%w: webhook errcode=%d errmsg=%s", ErrWebhookBalanceNotifySendFailed, wecom.ErrCode, wecom.ErrMsg)
		}
	}
	return nil
}
