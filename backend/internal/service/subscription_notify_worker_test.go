package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

// ─── Mocks ───

// stubSubNotifySettings 是一个本地最小 SettingRepository 实现，避免依赖
// `//go:build unit` 标签下的 mockSettingRepo。worker 只用 GetValue，
// 其它方法返回零值即可。
type stubSubNotifySettings struct {
	mu   sync.Mutex
	data map[string]string
}

func newStubSubNotifySettings() *stubSubNotifySettings {
	return &stubSubNotifySettings{data: make(map[string]string)}
}

func (s *stubSubNotifySettings) set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}

func (s *stubSubNotifySettings) GetValue(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[key]; ok {
		return v, nil
	}
	return "", nil
}

type stubSubNotifyFrontendURLResolver struct {
	url string
}

func (s *stubSubNotifyFrontendURLResolver) GetFrontendURL(_ context.Context) string {
	return s.url
}

type mockSubscriptionNotifyUserReader struct {
	mu     sync.Mutex
	users  map[int64]*User
	getErr error
	calls  int
}

func newMockSubNotifyUsers() *mockSubscriptionNotifyUserReader {
	return &mockSubscriptionNotifyUserReader{users: make(map[int64]*User)}
}

func (m *mockSubscriptionNotifyUserReader) GetByID(_ context.Context, id int64) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

type capturedSiteMessage struct {
	UserID  int64
	Subject string
	Content string
}

type mockSubscriptionNotifyMessenger struct {
	mu       sync.Mutex
	sendErr  error
	captured []capturedSiteMessage
}

func (m *mockSubscriptionNotifyMessenger) SendSubscriptionMessage(_ context.Context, userID int64, subject, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.captured = append(m.captured, capturedSiteMessage{
		UserID:  userID,
		Subject: subject,
		Content: content,
	})
	return nil
}

func (m *mockSubscriptionNotifyMessenger) calls() []capturedSiteMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]capturedSiteMessage, len(m.captured))
	copy(out, m.captured)
	return out
}

type capturedEmail struct {
	To      string
	Subject string
	Body    string
}

type mockSubscriptionNotifyEmailer struct {
	mu       sync.Mutex
	sendErr  error
	captured []capturedEmail
}

func (m *mockSubscriptionNotifyEmailer) SendSubscriptionEmail(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.captured = append(m.captured, capturedEmail{To: to, Subject: subject, Body: body})
	return nil
}

func (m *mockSubscriptionNotifyEmailer) calls() []capturedEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]capturedEmail, len(m.captured))
	copy(out, m.captured)
	return out
}

// ─── Helpers ───

func newSubNotifyHarness(t *testing.T) (*SubscriptionNotifyService, *mockSubscriptionNotifyUserReader, *mockSubscriptionNotifyMessenger, *mockSubscriptionNotifyEmailer, *stubSubNotifySettings) {
	t.Helper()
	users := newMockSubNotifyUsers()
	users.users[42] = &User{ID: 42, Email: "user42@example.com"}
	users.users[7] = &User{ID: 7, Email: "user7@example.com"}
	messenger := &mockSubscriptionNotifyMessenger{}
	emailer := &mockSubscriptionNotifyEmailer{}
	settings := newStubSubNotifySettings()
	svc := NewSubscriptionNotifyService(users, messenger, emailer, settings, "")
	return svc, users, messenger, emailer, settings
}

func mustMarshalPayload(t *testing.T, p SubscriptionNotifyPayload) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

// ─── Tests ───

func TestSubscriptionNotifyService_Handle_AllKindsSendBothChannelsWhenEmailEnabled(t *testing.T) {
	kinds := []struct {
		kind          string
		expectSubject string
	}{
		{"limit_reached_total", "订阅额度已用完"},
		{"limit_reached_daily", "订阅已到达日限额"},
		{"limit_reached_weekly", "订阅已到达周限额"},
		{"expired", "订阅已过期"},
	}

	for _, tc := range kinds {
		t.Run(tc.kind, func(t *testing.T) {
			svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
			settings.set(SettingKeySubscriptionCreditPoolRepurchaseURL, "https://example.com/buy")
			settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

			payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
				UserID:         42,
				SubscriptionID: 123,
				Kind:           tc.kind,
			})

			if err := svc.Handle(context.Background(), payload); err != nil {
				t.Fatalf("Handle returned error: %v", err)
			}

			msgs := messenger.calls()
			if len(msgs) != 1 {
				t.Fatalf("site message expected 1 call, got %d", len(msgs))
			}
			if msgs[0].UserID != 42 {
				t.Errorf("site message user id = %d, want 42", msgs[0].UserID)
			}
			if msgs[0].Subject != tc.expectSubject {
				t.Errorf("site message subject = %q, want %q", msgs[0].Subject, tc.expectSubject)
			}
			if !strings.Contains(msgs[0].Content, "https://example.com/buy") {
				t.Errorf("site message should contain repurchase URL, got: %s", msgs[0].Content)
			}
			if !strings.Contains(msgs[0].Content, "#123") {
				t.Errorf("site message should contain subscription id, got: %s", msgs[0].Content)
			}

			emails := emailer.calls()
			if len(emails) != 1 {
				t.Fatalf("email expected 1 call, got %d", len(emails))
			}
			if emails[0].To != "user42@example.com" {
				t.Errorf("email to = %q, want user42@example.com", emails[0].To)
			}
			if emails[0].Subject != tc.expectSubject {
				t.Errorf("email subject = %q, want %q", emails[0].Subject, tc.expectSubject)
			}
			if !strings.Contains(emails[0].Body, "https://example.com/buy") {
				t.Errorf("email body should contain repurchase URL")
			}
		})
	}
}

func TestSubscriptionNotifyService_Handle_LimitWindowCopyUsesConfiguredResetSchedule(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		expected string
	}{
		{
			name:     "daily",
			kind:     "limit_reached_daily",
			expected: "日限会在 UTC+8 4 点重置",
		},
		{
			name:     "weekly",
			kind:     "limit_reached_weekly",
			expected: "周限会在下周 UTC+8 周一 4 点重置",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
			settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")
			settings.set(SettingKeySubscriptionCreditPoolRepurchaseURL, "https://example.com/purchase")
			settings.set(SettingKeySubscriptionQuotaResetUTCOffsetMinutes, "480")
			settings.set(SettingKeySubscriptionQuotaResetHour, "4")

			payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
				UserID:         42,
				SubscriptionID: 123,
				Kind:           tc.kind,
			})
			if err := svc.Handle(context.Background(), payload); err != nil {
				t.Fatalf("Handle returned error: %v", err)
			}

			msgs := messenger.calls()
			if len(msgs) != 1 {
				t.Fatalf("site message expected 1 call, got %d", len(msgs))
			}
			if !strings.Contains(msgs[0].Content, tc.expected) {
				t.Fatalf("site message should describe configured reset schedule %q, got: %s", tc.expected, msgs[0].Content)
			}
			if strings.Contains(msgs[0].Content, "UTC 0 点") {
				t.Fatalf("site message should not hard-code UTC 0 reset, got: %s", msgs[0].Content)
			}

			emails := emailer.calls()
			if len(emails) != 1 {
				t.Fatalf("email expected 1 call, got %d", len(emails))
			}
			if !strings.Contains(emails[0].Body, tc.expected) {
				t.Fatalf("email body should describe configured reset schedule %q, got: %s", tc.expected, emails[0].Body)
			}
		})
	}
}

func TestRenderSubscriptionNotifyEmailUsesViewSubscriptionButtonCopy(t *testing.T) {
	body := renderSubscriptionNotifyEmail("订阅已到达周限额", "lead", "hint", "https://example.com/purchase")

	if !strings.Contains(body, "查看订阅 / Manage Subscription") {
		t.Fatalf("email button should say 查看订阅, got: %s", body)
	}
	if strings.Contains(body, "重新订阅 / Manage Subscription") {
		t.Fatalf("email button should not say 重新订阅, got: %s", body)
	}
}

func TestSubscriptionNotifyService_Handle_DefaultDoesNotSendEmail(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	settings.set(SettingKeySubscriptionCreditPoolRepurchaseURL, "https://example.com/buy")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 123,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if got := messenger.calls(); len(got) != 1 {
		t.Fatalf("site message expected 1 call, got %d", len(got))
	}
	if got := emailer.calls(); len(got) != 0 {
		t.Fatalf("email expected 0 calls when toggle disabled, got %d", len(got))
	}
}

func TestSubscriptionNotifyService_Handle_InvalidPayloadReturnsNil(t *testing.T) {
	svc, _, messenger, emailer, _ := newSubNotifyHarness(t)

	// Malformed JSON — must not bubble error up (would block worker watermark).
	err := svc.Handle(context.Background(), json.RawMessage("{not json"))
	if err != nil {
		t.Fatalf("invalid payload should return nil, got: %v", err)
	}
	if len(messenger.calls()) != 0 || len(emailer.calls()) != 0 {
		t.Errorf("no channel should be invoked on invalid payload")
	}
}

func TestSubscriptionNotifyService_Handle_EmptyPayloadReturnsNil(t *testing.T) {
	svc, _, messenger, emailer, _ := newSubNotifyHarness(t)

	if err := svc.Handle(context.Background(), nil); err != nil {
		t.Fatalf("nil payload should return nil, got: %v", err)
	}
	if err := svc.Handle(context.Background(), json.RawMessage{}); err != nil {
		t.Fatalf("empty payload should return nil, got: %v", err)
	}
	if len(messenger.calls()) != 0 || len(emailer.calls()) != 0 {
		t.Errorf("no channel should be invoked on empty payload")
	}
}

func TestSubscriptionNotifyService_Handle_MissingRequiredFieldsSkips(t *testing.T) {
	svc, _, messenger, emailer, _ := newSubNotifyHarness(t)

	cases := []SubscriptionNotifyPayload{
		{UserID: 0, SubscriptionID: 1, Kind: "limit_reached_total"},
		{UserID: 1, SubscriptionID: 0, Kind: "limit_reached_total"},
		{UserID: 1, SubscriptionID: 1, Kind: ""},
		{UserID: 1, SubscriptionID: 1, Kind: "   "},
	}
	for _, p := range cases {
		raw := mustMarshalPayload(t, p)
		if err := svc.Handle(context.Background(), raw); err != nil {
			t.Fatalf("Handle returned error: %v", err)
		}
	}
	if len(messenger.calls()) != 0 || len(emailer.calls()) != 0 {
		t.Errorf("no channel should be invoked when required fields are missing")
	}
}

func TestSubscriptionNotifyService_Handle_MessengerFailureDoesNotStopEmail(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	messenger.sendErr = errors.New("site db down")
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 999,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if got := emailer.calls(); len(got) != 1 {
		t.Errorf("email should still be sent when site message fails, got %d emails", len(got))
	}
}

func TestSubscriptionNotifyService_Handle_EmailFailureDoesNotStopMessage(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	emailer.sendErr = errors.New("smtp timeout")
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 999,
		Kind:           "expired",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if got := messenger.calls(); len(got) != 1 {
		t.Errorf("site message should still be sent when email fails, got %d messages", len(got))
	}
}

func TestSubscriptionNotifyService_Handle_BothChannelsFailingReturnsNil(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	messenger.sendErr = errors.New("site db down")
	emailer.sendErr = errors.New("smtp timeout")
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 1,
		Kind:           "limit_reached_daily",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v (should swallow per no-retry contract)", err)
	}
}

func TestSubscriptionNotifyService_Handle_UnknownKindStillSends(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 1,
		Kind:           "future_kind_we_dont_know_yet",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("unknown kind should not error: %v", err)
	}

	if len(messenger.calls()) != 1 {
		t.Errorf("unknown kind should fall back to generic message, got %d site messages", len(messenger.calls()))
	}
	if len(emailer.calls()) != 1 {
		t.Errorf("unknown kind should still email, got %d emails", len(emailer.calls()))
	}
}

func TestSubscriptionNotifyService_Handle_RepurchaseURLFallsBackToFrontend(t *testing.T) {
	svc, _, messenger, _, settings := newSubNotifyHarness(t)
	// 显式设置为空（覆盖默认）
	settings.set(SettingKeySubscriptionCreditPoolRepurchaseURL, "")
	settings.set(SettingKeyFrontendURL, "https://app.example.com/")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 1,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(messenger.calls()) != 1 {
		t.Fatalf("expected 1 site message, got %d", len(messenger.calls()))
	}
	body := messenger.calls()[0].Content
	if !strings.Contains(body, "https://app.example.com/purchase") {
		t.Errorf("body should fall back to frontend purchase page, got: %s", body)
	}
}

func TestSubscriptionNotifyService_Handle_RepurchaseURLFallsBackToConfiguredFrontendURL(t *testing.T) {
	users := newMockSubNotifyUsers()
	users.users[42] = &User{ID: 42, Email: "user42@example.com"}
	messenger := &mockSubscriptionNotifyMessenger{}
	emailer := &mockSubscriptionNotifyEmailer{}
	settings := newStubSubNotifySettings()
	resolver := &stubSubNotifyFrontendURLResolver{url: "https://cfg.example.com/base"}
	svc := NewSubscriptionNotifyService(users, messenger, emailer, settings, resolver.GetFrontendURL(context.Background()))

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 1,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(messenger.calls()) != 1 {
		t.Fatalf("expected 1 site message, got %d", len(messenger.calls()))
	}
	body := messenger.calls()[0].Content
	if !strings.Contains(body, "https://cfg.example.com/base/purchase") {
		t.Errorf("body should fall back to configured frontend url, got: %s", body)
	}
}

func TestSubscriptionNotifyService_Handle_RepurchaseURLDefaultsToPurchaseWhenAllEmpty(t *testing.T) {
	svc, _, messenger, _, _ := newSubNotifyHarness(t)

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         42,
		SubscriptionID: 1,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// 没有 repurchase URL，没有 frontend URL，文案仍能渲染（不崩溃）。
	if len(messenger.calls()) != 1 {
		t.Fatalf("expected 1 site message, got %d", len(messenger.calls()))
	}
	body := messenger.calls()[0].Content
	if !strings.Contains(body, "/purchase") {
		t.Errorf("body should include default purchase link, got: %s", body)
	}
}

func TestSubscriptionNotifyService_Handle_UserMissingSkipsEmailOnly(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID:         9999, // 不在 mock 里
		SubscriptionID: 1,
		Kind:           "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if got := messenger.calls(); len(got) != 1 {
		t.Errorf("site message should still send (it doesn't need email lookup), got %d", len(got))
	}
	if got := emailer.calls(); len(got) != 0 {
		t.Errorf("email should be skipped when user not found, got %d", len(got))
	}
}

func TestSubscriptionNotifyService_Handle_NilChannelsTolerated(t *testing.T) {
	// 任一 channel 为 nil 都应被跳过，不应 panic。
	users := newMockSubNotifyUsers()
	users.users[42] = &User{ID: 42, Email: "user42@example.com"}

	svc := NewSubscriptionNotifyService(users, nil, nil, newStubSubNotifySettings())
	payload := mustMarshalPayload(t, SubscriptionNotifyPayload{
		UserID: 42, SubscriptionID: 1, Kind: "limit_reached_total",
	})
	if err := svc.Handle(context.Background(), payload); err != nil {
		t.Fatalf("nil channels should not error: %v", err)
	}
}

// ─── Worker tests ───

type fakeSubscriptionNotifyOutboxRepo struct {
	mu        sync.Mutex
	events    []SchedulerOutboxEvent
	listErr   error
	maxIDErr  error
	maxIDCall int
}

func (f *fakeSubscriptionNotifyOutboxRepo) ListSubscriptionNotifyAfter(_ context.Context, afterID int64, limit int) ([]SchedulerOutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]SchedulerOutboxEvent, 0)
	for _, e := range f.events {
		if e.ID > afterID {
			out = append(out, e)
		}
		if len(out) >= limit && limit > 0 {
			break
		}
	}
	return out, nil
}

func (f *fakeSubscriptionNotifyOutboxRepo) MaxSubscriptionNotifyID(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.maxIDCall++
	if f.maxIDErr != nil {
		return 0, f.maxIDErr
	}
	var max int64
	for _, e := range f.events {
		if e.ID > max {
			max = e.ID
		}
	}
	return max, nil
}

func TestSubscriptionNotifyWorker_PollOnce_DispatchesEvents(t *testing.T) {
	svc, _, messenger, emailer, settings := newSubNotifyHarness(t)
	settings.set(SettingKeySubscriptionNotifyEmailEnabled, "true")

	repo := &fakeSubscriptionNotifyOutboxRepo{
		events: []SchedulerOutboxEvent{
			{
				ID:        10,
				EventType: SchedulerOutboxEventSubscriptionNotify,
				Payload: map[string]any{
					"user_id":         float64(42),
					"subscription_id": float64(100),
					"kind":            "limit_reached_total",
				},
			},
			{
				ID:        11,
				EventType: SchedulerOutboxEventSubscriptionNotify,
				Payload: map[string]any{
					"user_id":         float64(7),
					"subscription_id": float64(200),
					"kind":            "expired",
				},
			},
		},
	}

	worker := NewSubscriptionNotifyWorker(repo, svc, 0, 0)
	worker.PollOnceForTest()

	if msgs := messenger.calls(); len(msgs) != 2 {
		t.Errorf("expected 2 site messages, got %d", len(msgs))
	}
	if emails := emailer.calls(); len(emails) != 2 {
		t.Errorf("expected 2 emails, got %d", len(emails))
	}
	if got := worker.getWatermark(); got != 11 {
		t.Errorf("watermark = %d, want 11", got)
	}

	// Second poll: no new events, no extra dispatch.
	worker.PollOnceForTest()
	if msgs := messenger.calls(); len(msgs) != 2 {
		t.Errorf("after second poll expected still 2 site messages, got %d", len(msgs))
	}
}

func TestSubscriptionNotifyWorker_PollOnce_NoCrashOnRepoError(t *testing.T) {
	svc, _, _, _, _ := newSubNotifyHarness(t)
	repo := &fakeSubscriptionNotifyOutboxRepo{listErr: errors.New("db down")}
	worker := NewSubscriptionNotifyWorker(repo, svc, 0, 0)

	// pollOnce 不应 panic，应静默记录日志。
	worker.PollOnceForTest()
	if w := worker.getWatermark(); w != 0 {
		t.Errorf("watermark must not advance on poll error, got %d", w)
	}
}

func TestSubscriptionNotifyWorker_NewWithNilHandler_StartIsNoop(t *testing.T) {
	w := NewSubscriptionNotifyWorker(&fakeSubscriptionNotifyOutboxRepo{}, nil, 0, 0)
	w.Start() // 必须无 panic
	w.Stop()
}
