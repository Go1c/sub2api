package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// DefaultWebsocketBalanceAlertThreshold is used when the user leaves WS threshold empty.
const DefaultWebsocketBalanceAlertThreshold = 10.0

// UserWebsocket event types pushed to the browser.
const (
	UserWSEventTest         = "test"
	UserWSEventBalanceLow   = "balance_low"
	UserWSEventSiteMessage  = "site_message"
	UserWSEventAnnouncement = "announcement"
)

// UserWSEvent is the JSON payload sent over the user notification WebSocket.
type UserWSEvent struct {
	Type      string         `json:"type"`
	Title     string         `json:"title,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Timestamp int64          `json:"timestamp"`
}

// UserWSConn is a single client connection that can receive JSON events.
type UserWSConn interface {
	SendJSON(v any) error
	Close() error
}

// UserWebsocketHub keeps per-user WebSocket connections and fans out events.
type UserWebsocketHub struct {
	mu    sync.RWMutex
	conns map[int64]map[UserWSConn]struct{}
}

// NewUserWebsocketHub creates an empty hub.
func NewUserWebsocketHub() *UserWebsocketHub {
	return &UserWebsocketHub{
		conns: make(map[int64]map[UserWSConn]struct{}),
	}
}

// Register adds a connection for userID.
func (h *UserWebsocketHub) Register(userID int64, conn UserWSConn) {
	if h == nil || conn == nil || userID <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[UserWSConn]struct{})
	}
	h.conns[userID][conn] = struct{}{}
}

// Unregister removes a connection for userID.
func (h *UserWebsocketHub) Unregister(userID int64, conn UserWSConn) {
	if h == nil || conn == nil || userID <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.conns[userID]
	if set == nil {
		return
	}
	delete(set, conn)
	if len(set) == 0 {
		delete(h.conns, userID)
	}
}

// OnlineCount returns how many live connections the user currently has.
func (h *UserWebsocketHub) OnlineCount(userID int64) int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID])
}

// OnlineUserIDs returns user IDs that currently have at least one live connection.
func (h *UserWebsocketHub) OnlineUserIDs() []int64 {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]int64, 0, len(h.conns))
	for id := range h.conns {
		out = append(out, id)
	}
	return out
}

// Publish sends an event to all connections of userID. Returns number of successful sends.
func (h *UserWebsocketHub) Publish(userID int64, event UserWSEvent) int {
	if h == nil || userID <= 0 {
		return 0
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	h.mu.RLock()
	set := h.conns[userID]
	conns := make([]UserWSConn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	ok := 0
	var stale []UserWSConn
	for _, c := range conns {
		if err := c.SendJSON(event); err != nil {
			slog.Debug("user websocket publish failed", "user_id", userID, "type", event.Type, "err", err)
			stale = append(stale, c)
			continue
		}
		ok++
	}
	for _, c := range stale {
		h.Unregister(userID, c)
		_ = c.Close()
	}
	return ok
}

// PublishJSON is a helper when callers already have a map payload.
func (h *UserWebsocketHub) PublishJSON(userID int64, eventType, title, message string, data map[string]any) int {
	return h.Publish(userID, UserWSEvent{
		Type:    eventType,
		Title:   title,
		Message: message,
		Data:    data,
	})
}

// ResolveWebsocketBalanceThreshold returns the effective WS balance threshold for a user.
func ResolveWebsocketBalanceThreshold(user *User) float64 {
	if user == nil {
		return 0
	}
	if user.WebsocketBalanceAlertThreshold != nil && *user.WebsocketBalanceAlertThreshold > 0 {
		return *user.WebsocketBalanceAlertThreshold
	}
	return DefaultWebsocketBalanceAlertThreshold
}

// ShouldPushWebsocketBalanceAlert reports whether a balance cross should push WS.
func ShouldPushWebsocketBalanceAlert(user *User) bool {
	if user == nil {
		return false
	}
	return user.WebsocketNotifyEnabled && user.WebsocketBalanceAlertEnabled && ResolveWebsocketBalanceThreshold(user) > 0
}

// ShouldPushWebsocketSiteMessage reports whether a new site message should push WS.
func ShouldPushWebsocketSiteMessage(user *User) bool {
	if user == nil {
		return false
	}
	return user.WebsocketNotifyEnabled && user.WebsocketSiteMessageNotifyEnabled
}

// ShouldPushWebsocketAnnouncement reports whether a new announcement should push WS.
func ShouldPushWebsocketAnnouncement(user *User) bool {
	if user == nil {
		return false
	}
	return user.WebsocketNotifyEnabled && user.WebsocketAnnouncementNotifyEnabled
}

// MustMarshalEvent is used by tests / debugging.
func MustMarshalEvent(event UserWSEvent) []byte {
	b, err := json.Marshal(event)
	if err != nil {
		return []byte(`{"type":"error"}`)
	}
	return b
}

// ErrUserWebsocketUnavailable indicates the hub/service is not wired.
var ErrUserWebsocketUnavailable = fmt.Errorf("USER_WEBSOCKET_UNAVAILABLE")

// ErrUserWebsocketDisabled indicates the user has not enabled WebSocket notifications.
var ErrUserWebsocketDisabled = fmt.Errorf("USER_WEBSOCKET_DISABLED")

// ErrUserWebsocketNotConnected indicates no active browser connection for the user.
var ErrUserWebsocketNotConnected = fmt.Errorf("USER_WEBSOCKET_NOT_CONNECTED")

// UserWebsocketNotifier is the narrow port used by billing / site-message / announcement hooks.
type UserWebsocketNotifier interface {
	NotifyBalanceLow(ctx context.Context, user *User, newBalance, threshold float64)
	NotifySiteMessage(ctx context.Context, recipientID int64, messageID int64, subject string)
	NotifyAnnouncement(ctx context.Context, userID int64, announcementID int64, title string)
	NotifyAnnouncementPublished(ctx context.Context, a *Announcement)
	NotifyTest(ctx context.Context, userID int64) (sent int, err error)
}

// UserWebsocketUserReader loads users for preference checks.
type UserWebsocketUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

// UserWebsocketNotifyService implements UserWebsocketNotifier using the hub + user repo.
type UserWebsocketNotifyService struct {
	hub      *UserWebsocketHub
	userRepo UserWebsocketUserReader
}

// NewUserWebsocketNotifyService constructs the notify service.
func NewUserWebsocketNotifyService(hub *UserWebsocketHub, userRepo UserWebsocketUserReader) *UserWebsocketNotifyService {
	return &UserWebsocketNotifyService{hub: hub, userRepo: userRepo}
}

// Hub exposes the hub for the HTTP upgrade handler.
func (s *UserWebsocketNotifyService) Hub() *UserWebsocketHub {
	if s == nil {
		return nil
	}
	return s.hub
}

func (s *UserWebsocketNotifyService) NotifyBalanceLow(_ context.Context, user *User, newBalance, threshold float64) {
	if s == nil || s.hub == nil || !ShouldPushWebsocketBalanceAlert(user) {
		return
	}
	s.hub.PublishJSON(user.ID, UserWSEventBalanceLow, "余额不足提醒", "账户余额已低于设定阈值", map[string]any{
		"balance":   newBalance,
		"threshold": threshold,
	})
}

func (s *UserWebsocketNotifyService) NotifySiteMessage(ctx context.Context, recipientID int64, messageID int64, subject string) {
	if s == nil || s.hub == nil || recipientID <= 0 {
		return
	}
	user, err := s.loadUser(ctx, recipientID)
	if err != nil || !ShouldPushWebsocketSiteMessage(user) {
		return
	}
	s.hub.PublishJSON(recipientID, UserWSEventSiteMessage, "新站内信", subject, map[string]any{
		"message_id": messageID,
		"subject":    subject,
	})
}

func (s *UserWebsocketNotifyService) NotifyAnnouncement(ctx context.Context, userID int64, announcementID int64, title string) {
	if s == nil || s.hub == nil || userID <= 0 {
		return
	}
	user, err := s.loadUser(ctx, userID)
	if err != nil || !ShouldPushWebsocketAnnouncement(user) {
		return
	}
	s.hub.PublishJSON(userID, UserWSEventAnnouncement, "新公告", title, map[string]any{
		"announcement_id": announcementID,
		"title":           title,
	})
}

func (s *UserWebsocketNotifyService) NotifyAnnouncementPublished(ctx context.Context, a *Announcement) {
	if s == nil || s.hub == nil || a == nil || a.Status != AnnouncementStatusActive {
		return
	}
	online := s.hub.OnlineUserIDs()
	if len(online) == 0 {
		return
	}
	for _, userID := range online {
		user, err := s.loadUser(ctx, userID)
		if err != nil || !ShouldPushWebsocketAnnouncement(user) {
			continue
		}
		// Targeting: if we cannot evaluate groups cheaply, still push when targeting is empty/all.
		// Full group matching is best-effort via balance-only when no subscription repo here.
		if !announcementLikelyVisible(user, a) {
			continue
		}
		s.hub.PublishJSON(userID, UserWSEventAnnouncement, "新公告", a.Title, map[string]any{
			"announcement_id": a.ID,
			"title":           a.Title,
			"notify_mode":     a.NotifyMode,
		})
	}
}

// announcementLikelyVisible is a lightweight gate for WS fan-out.
// Empty targeting => visible to all. Subscription-only targeting is skipped for offline-group
// evaluation and still notified (user will see correct list on fetch); balance rules are applied.
func announcementLikelyVisible(user *User, a *Announcement) bool {
	if user == nil || a == nil {
		return false
	}
	t := domain.AnnouncementTargeting(a.Targeting)
	if len(t.AnyOf) == 0 {
		return true
	}
	// Balance-only evaluation with empty group set: subscription conditions won't match,
	// which is acceptable (user still gets announcements via poll).
	return t.Matches(user.Balance, map[int64]struct{}{})
}

func (s *UserWebsocketNotifyService) NotifyTest(ctx context.Context, userID int64) (int, error) {
	if s == nil || s.hub == nil {
		return 0, ErrUserWebsocketUnavailable
	}
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil || !user.WebsocketNotifyEnabled {
		return 0, ErrUserWebsocketDisabled
	}
	if s.hub.OnlineCount(userID) == 0 {
		return 0, ErrUserWebsocketNotConnected
	}
	sent := s.hub.PublishJSON(userID, UserWSEventTest, "测试通知", "这是一条 WebSocket 测试消息", map[string]any{
		"ok": true,
	})
	return sent, nil
}

func (s *UserWebsocketNotifyService) loadUser(ctx context.Context, userID int64) (*User, error) {
	if s.userRepo == nil {
		return nil, ErrUserNotFound
	}
	return s.userRepo.GetByID(ctx, userID)
}

// CheckWebsocketBalanceAfterDeduction pushes WS when balance first crosses below the WS threshold.
// Independent from email BalanceNotifyService.
func (s *UserWebsocketNotifyService) CheckWebsocketBalanceAfterDeduction(ctx context.Context, user *User, oldBalance, cost float64) {
	if s == nil || !ShouldPushWebsocketBalanceAlert(user) {
		return
	}
	threshold := ResolveWebsocketBalanceThreshold(user)
	if threshold <= 0 {
		return
	}
	newBalance := oldBalance - cost
	if !crossedDownward(oldBalance, newBalance, threshold) {
		return
	}
	s.NotifyBalanceLow(ctx, user, newBalance, threshold)
}
