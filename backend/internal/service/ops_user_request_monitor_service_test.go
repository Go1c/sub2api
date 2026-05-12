package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type opsUserRequestMonitorRepoStub struct {
	OpsRepository

	mu             sync.Mutex
	activeMonitors []*OpsUserRequestMonitor
	created        []*OpsCreateUserRequestMonitorRecord
	inserted       []*OpsInsertUserRequestCaptureInput

	createErr          error
	insertErr          error
	deleteMonitorErr   error
	streamErr          error
	createCalls        int
	deleteMonitorID    int64
	deleteMonitorOK    bool
	streamMonitorID    int64
	firstCreateStarted chan struct{}
	releaseFirstCreate chan struct{}
	monitorByID        *OpsUserRequestMonitor
	streamCaptures     []*OpsUserRequestCapture
}

func (r *opsUserRequestMonitorRepoStub) GetActiveUserRequestMonitors(ctx context.Context, userID int64, now time.Time) ([]*OpsUserRequestMonitor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneOpsUserRequestMonitors(r.activeMonitors), nil
}

func (r *opsUserRequestMonitorRepoStub) CreateUserRequestMonitor(ctx context.Context, input *OpsCreateUserRequestMonitorRecord) (*OpsUserRequestMonitor, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.mu.Lock()
	r.createCalls++
	callNum := r.createCalls
	r.mu.Unlock()
	if callNum == 1 && r.firstCreateStarted != nil {
		close(r.firstCreateStarted)
		if r.releaseFirstCreate != nil {
			<-r.releaseFirstCreate
		}
	}
	monitor := &OpsUserRequestMonitor{
		ID:                   11,
		UserID:               input.UserID,
		TargetEmail:          input.TargetEmail,
		Status:               OpsUserRequestMonitorStatusActive,
		DurationSeconds:      input.DurationSeconds,
		MaxCapturesPerMinute: input.MaxCapturesPerMinute,
		SampleRatePercent:    input.SampleRatePercent,
		RetentionDays:        input.RetentionDays,
		CreatedBy:            input.CreatedBy,
		CreatedAt:            input.CreatedAt,
		StartsAt:             input.StartsAt,
		EndsAt:               input.EndsAt,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, input)
	r.activeMonitors = append(r.activeMonitors, monitor)
	return monitor, nil
}

func (r *opsUserRequestMonitorRepoStub) InsertUserRequestCapture(ctx context.Context, input *OpsInsertUserRequestCaptureInput) (int64, error) {
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	r.inserted = append(r.inserted, input)
	return int64(len(r.inserted)), nil
}

func (r *opsUserRequestMonitorRepoStub) GetUserRequestMonitorByID(ctx context.Context, id int64) (*OpsUserRequestMonitor, error) {
	if r.monitorByID != nil {
		return r.monitorByID, nil
	}
	return &OpsUserRequestMonitor{ID: id, UserID: 42}, nil
}

func (r *opsUserRequestMonitorRepoStub) DeleteUserRequestMonitor(ctx context.Context, id int64) (bool, error) {
	r.deleteMonitorID = id
	if r.deleteMonitorErr != nil {
		return false, r.deleteMonitorErr
	}
	return r.deleteMonitorOK, nil
}

func (r *opsUserRequestMonitorRepoStub) StreamUserRequestCaptures(ctx context.Context, monitorID int64, handle func(*OpsUserRequestCapture) error) error {
	r.streamMonitorID = monitorID
	if r.streamErr != nil {
		return r.streamErr
	}
	for _, capture := range r.streamCaptures {
		if err := handle(capture); err != nil {
			return err
		}
	}
	return nil
}

type opsUserLookupStub struct {
	user *User
	err  error
}

func (s opsUserLookupStub) GetByID(ctx context.Context, id int64) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

type opsCaptureLimiterStub struct {
	allow bool
	err   error
	calls int
	order *[]string
}

func (l *opsCaptureLimiterStub) Allow(ctx context.Context, monitorID int64, captureMinute time.Time, maxPerMinute int) (bool, error) {
	l.calls++
	if l.order != nil {
		*l.order = append(*l.order, "limiter")
	}
	return l.allow, l.err
}

func newMonitorServiceForTest(repo *opsUserRequestMonitorRepoStub) *OpsUserRequestMonitorService {
	svc := NewOpsUserRequestMonitorService(repo, nil, nil)
	svc.userLookup = opsUserLookupStub{user: &User{ID: 42, Email: "target@example.com"}}
	svc.limiter = &opsCaptureLimiterStub{allow: true}
	svc.sample = func(percent int) bool { return true }
	svc.now = func() time.Time { return time.Date(2026, 5, 9, 10, 11, 12, 0, time.UTC) }
	return svc
}

func TestOpsUserRequestMonitorService_CreateDefaultsRetentionToSevenDays(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{}
	svc := newMonitorServiceForTest(repo)

	monitor, err := svc.CreateMonitor(context.Background(), &OpsCreateUserRequestMonitorInput{
		UserID:               42,
		DurationSeconds:      300,
		MaxCapturesPerMinute: 10,
		SampleRatePercent:    100,
		CreatedBy:            1,
	})
	if err != nil {
		t.Fatalf("CreateMonitor returned error: %v", err)
	}
	if monitor.RetentionDays != 7 {
		t.Fatalf("retention default = %d, want 7", monitor.RetentionDays)
	}
	if len(repo.created) != 1 {
		t.Fatalf("created records = %d, want 1", len(repo.created))
	}
	if repo.created[0].TargetEmail != "target@example.com" {
		t.Fatalf("target email = %q, want target@example.com", repo.created[0].TargetEmail)
	}
}

func TestOpsUserRequestMonitorService_CreateValidatesLimits(t *testing.T) {
	tests := []struct {
		name  string
		input OpsCreateUserRequestMonitorInput
	}{
		{
			name: "duration must be positive",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      0,
				MaxCapturesPerMinute: 10,
				SampleRatePercent:    100,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "duration capped at 24 hours",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      24*60*60 + 1,
				MaxCapturesPerMinute: 10,
				SampleRatePercent:    100,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "max captures must be positive",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      300,
				MaxCapturesPerMinute: 0,
				SampleRatePercent:    100,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "max captures capped",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      300,
				MaxCapturesPerMinute: 121,
				SampleRatePercent:    100,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "sample rate must be at least one",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      300,
				MaxCapturesPerMinute: 10,
				SampleRatePercent:    0,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "sample rate capped at one hundred",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      300,
				MaxCapturesPerMinute: 10,
				SampleRatePercent:    101,
				RetentionDays:        7,
				CreatedBy:            1,
			},
		},
		{
			name: "retention capped at thirty days",
			input: OpsCreateUserRequestMonitorInput{
				UserID:               42,
				DurationSeconds:      300,
				MaxCapturesPerMinute: 10,
				SampleRatePercent:    100,
				RetentionDays:        31,
				CreatedBy:            1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &opsUserRequestMonitorRepoStub{}
			svc := newMonitorServiceForTest(repo)

			if _, err := svc.CreateMonitor(context.Background(), &tt.input); err == nil {
				t.Fatalf("CreateMonitor returned nil error")
			}
			if len(repo.created) != 0 {
				t.Fatalf("created records = %d, want 0", len(repo.created))
			}
		})
	}
}

func TestOpsUserRequestMonitorService_CreateRejectsSecondActiveMonitor(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		activeMonitors: []*OpsUserRequestMonitor{{ID: 99, UserID: 42, Status: OpsUserRequestMonitorStatusActive}},
	}
	svc := newMonitorServiceForTest(repo)

	_, err := svc.CreateMonitor(context.Background(), &OpsCreateUserRequestMonitorInput{
		UserID:               42,
		DurationSeconds:      300,
		MaxCapturesPerMinute: 10,
		SampleRatePercent:    100,
		RetentionDays:        7,
		CreatedBy:            1,
	})
	if err == nil {
		t.Fatalf("CreateMonitor returned nil error")
	}
	if len(repo.created) != 0 {
		t.Fatalf("created records = %d, want 0", len(repo.created))
	}
}

func TestOpsUserRequestMonitorService_CreateMonitorSerializesConcurrentRequestsForSameUser(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		firstCreateStarted: make(chan struct{}),
		releaseFirstCreate: make(chan struct{}),
	}
	svc := newMonitorServiceForTest(repo)
	input := &OpsCreateUserRequestMonitorInput{
		UserID:               42,
		DurationSeconds:      300,
		MaxCapturesPerMinute: 10,
		SampleRatePercent:    100,
		RetentionDays:        7,
		CreatedBy:            1,
	}

	errCh := make(chan error, 2)
	go func() {
		_, err := svc.CreateMonitor(context.Background(), input)
		errCh <- err
	}()
	<-repo.firstCreateStarted

	secondReturned := make(chan struct{})
	go func() {
		_, err := svc.CreateMonitor(context.Background(), input)
		errCh <- err
		close(secondReturned)
	}()

	select {
	case <-secondReturned:
		t.Fatalf("second CreateMonitor returned before first create was released")
	case <-time.After(50 * time.Millisecond):
	}

	close(repo.releaseFirstCreate)

	var successCount int
	var conflictCount int
	for range 2 {
		err := <-errCh
		switch {
		case err == nil:
			successCount++
		case strings.Contains(err.Error(), "already has an active request monitor"):
			conflictCount++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("success=%d conflict=%d, want 1/1", successCount, conflictCount)
	}
	if repo.createCalls != 1 {
		t.Fatalf("repo create calls = %d, want 1", repo.createCalls)
	}
}

func TestOpsUserRequestMonitorService_CaptureTruncatesRawBody(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		activeMonitors: []*OpsUserRequestMonitor{{
			ID:                   7,
			UserID:               42,
			Status:               OpsUserRequestMonitorStatusActive,
			MaxCapturesPerMinute: 10,
			SampleRatePercent:    100,
			RetentionDays:        7,
			StartsAt:             time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			EndsAt:               time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC),
		}},
	}
	svc := newMonitorServiceForTest(repo)
	body := []byte(strings.Repeat("a", opsUserRequestMonitorMaxBodyBytes+1))

	err := svc.CaptureClientRequestSync(context.Background(), &OpsCaptureClientRequestInput{
		UserID:          42,
		RequestID:       "req_1",
		Model:           "claude-3-5-sonnet",
		InboundEndpoint: "/v1/messages",
		Method:          "POST",
		ContentType:     "application/json",
		Body:            body,
	})
	if err != nil {
		t.Fatalf("CaptureClientRequestSync returned error: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("inserted captures = %d, want 1", len(repo.inserted))
	}
	got := repo.inserted[0]
	if len(got.Body) != opsUserRequestMonitorMaxBodyBytes {
		t.Fatalf("stored body len = %d, want %d", len(got.Body), opsUserRequestMonitorMaxBodyBytes)
	}
	if got.BodyBytes != opsUserRequestMonitorMaxBodyBytes+1 {
		t.Fatalf("body_bytes = %d, want %d", got.BodyBytes, opsUserRequestMonitorMaxBodyBytes+1)
	}
	if !got.BodyTruncated {
		t.Fatalf("body_truncated = false, want true")
	}
}

func TestOpsUserRequestMonitorService_CaptureUsesProvidedOriginalBodyBytes(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		activeMonitors: []*OpsUserRequestMonitor{{
			ID:                   7,
			UserID:               42,
			Status:               OpsUserRequestMonitorStatusActive,
			MaxCapturesPerMinute: 10,
			SampleRatePercent:    100,
			RetentionDays:        7,
			StartsAt:             time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			EndsAt:               time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC),
		}},
	}
	svc := newMonitorServiceForTest(repo)

	err := svc.CaptureClientRequestSync(context.Background(), &OpsCaptureClientRequestInput{
		UserID:    42,
		Body:      []byte(`{"multipart_summary":true}`),
		BodyBytes: 2048,
	})
	if err != nil {
		t.Fatalf("CaptureClientRequestSync returned error: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("inserted captures = %d, want 1", len(repo.inserted))
	}
	if repo.inserted[0].BodyBytes != 2048 {
		t.Fatalf("body_bytes = %d, want 2048", repo.inserted[0].BodyBytes)
	}
}

func TestOpsUserRequestMonitorService_CaptureAppliesRateBeforeSampling(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		activeMonitors: []*OpsUserRequestMonitor{{
			ID:                   7,
			UserID:               42,
			Status:               OpsUserRequestMonitorStatusActive,
			MaxCapturesPerMinute: 1,
			SampleRatePercent:    50,
			RetentionDays:        7,
			StartsAt:             time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			EndsAt:               time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC),
		}},
	}
	svc := newMonitorServiceForTest(repo)
	order := []string{}
	svc.limiter = &opsCaptureLimiterStub{allow: true, order: &order}
	svc.sample = func(percent int) bool {
		order = append(order, "sample")
		return true
	}

	if err := svc.CaptureClientRequestSync(context.Background(), &OpsCaptureClientRequestInput{
		UserID: 42,
		Body:   []byte(`{"model":"x"}`),
	}); err != nil {
		t.Fatalf("CaptureClientRequestSync returned error: %v", err)
	}
	if strings.Join(order, ",") != "limiter,sample" {
		t.Fatalf("call order = %q, want limiter,sample", strings.Join(order, ","))
	}

	repo.inserted = nil
	order = []string{}
	svc.limiter = &opsCaptureLimiterStub{allow: false, order: &order}
	svc.sample = func(percent int) bool {
		order = append(order, "sample")
		return true
	}

	if err := svc.CaptureClientRequestSync(context.Background(), &OpsCaptureClientRequestInput{
		UserID: 42,
		Body:   []byte(`{"model":"x"}`),
	}); err != nil {
		t.Fatalf("CaptureClientRequestSync returned error: %v", err)
	}
	if strings.Join(order, ",") != "limiter" {
		t.Fatalf("call order = %q, want limiter", strings.Join(order, ","))
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("inserted captures = %d, want 0", len(repo.inserted))
	}
}

func TestOpsUserRequestMonitorService_CaptureSwallowsInsertErrors(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{
		activeMonitors: []*OpsUserRequestMonitor{{
			ID:                   7,
			UserID:               42,
			Status:               OpsUserRequestMonitorStatusActive,
			MaxCapturesPerMinute: 10,
			SampleRatePercent:    100,
			RetentionDays:        7,
			StartsAt:             time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			EndsAt:               time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC),
		}},
		insertErr: errors.New("database unavailable"),
	}
	svc := newMonitorServiceForTest(repo)

	if err := svc.CaptureClientRequestSync(context.Background(), &OpsCaptureClientRequestInput{
		UserID: 42,
		Body:   []byte(`{"model":"x"}`),
	}); err != nil {
		t.Fatalf("CaptureClientRequestSync returned error: %v", err)
	}
}

func TestOpsUserRequestMonitorService_DeleteMonitorDeletesWholeMonitor(t *testing.T) {
	repo := &opsUserRequestMonitorRepoStub{deleteMonitorOK: true}
	svc := newMonitorServiceForTest(repo)

	if err := svc.DeleteMonitor(context.Background(), 77); err != nil {
		t.Fatalf("DeleteMonitor returned error: %v", err)
	}

	if repo.deleteMonitorID != 77 {
		t.Fatalf("delete monitor id = %d, want 77", repo.deleteMonitorID)
	}
}

func TestOpsUserRequestMonitorService_ExportCapturesJSONLIncludesRawBodies(t *testing.T) {
	createdAt := time.Date(2026, 5, 12, 2, 3, 4, 0, time.UTC)
	repo := &opsUserRequestMonitorRepoStub{
		monitorByID: &OpsUserRequestMonitor{ID: 77, UserID: 42},
		streamCaptures: []*OpsUserRequestCapture{{
			ID:              9,
			MonitorID:       77,
			UserID:          42,
			RequestID:       "req-1",
			Model:           "gpt-test",
			InboundEndpoint: "/v1/chat/completions",
			ContentType:     "application/json",
			Body:            `{"messages":[{"role":"user","content":"hello"}]}`,
			BodyBytes:       48,
			CreatedAt:       createdAt,
			ExpiresAt:       createdAt.Add(24 * time.Hour),
		}},
	}
	svc := newMonitorServiceForTest(repo)

	var buf bytes.Buffer
	if err := svc.ExportCapturesJSONL(context.Background(), 77, &buf); err != nil {
		t.Fatalf("ExportCapturesJSONL returned error: %v", err)
	}

	if repo.streamMonitorID != 77 {
		t.Fatalf("stream monitor id = %d, want 77", repo.streamMonitorID)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("export line count = %d, want 1; body=%q", len(lines), buf.String())
	}
	var got OpsUserRequestCapture
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("export line is not JSON: %v", err)
	}
	if got.ID != 9 || got.MonitorID != 77 || got.Body == "" {
		t.Fatalf("exported capture = %+v, want id 9, monitor 77, non-empty raw body", got)
	}
}

func TestSnapshotOpsCaptureClientRequestInputCapsClonedBodyAndPreservesOriginalBytes(t *testing.T) {
	raw := []byte(strings.Repeat("a", opsUserRequestMonitorMaxBodyBytes+32))
	snapshot := snapshotOpsCaptureClientRequestInput(&OpsCaptureClientRequestInput{
		UserID:    42,
		Body:      raw,
		BodyBytes: len(raw),
	})
	if snapshot == nil {
		t.Fatalf("snapshot = nil")
	}
	if len(snapshot.Body) != opsUserRequestMonitorMaxBodyBytes {
		t.Fatalf("snapshot body len = %d, want %d", len(snapshot.Body), opsUserRequestMonitorMaxBodyBytes)
	}
	if snapshot.BodyBytes != len(raw) {
		t.Fatalf("snapshot body bytes = %d, want %d", snapshot.BodyBytes, len(raw))
	}
	raw[0] = 'z'
	if snapshot.Body[0] != 'a' {
		t.Fatalf("snapshot body mutated with source slice")
	}
}
