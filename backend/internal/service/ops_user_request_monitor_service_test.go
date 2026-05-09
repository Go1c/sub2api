package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type opsUserRequestMonitorRepoStub struct {
	OpsRepository

	activeMonitors []*OpsUserRequestMonitor
	created        []*OpsCreateUserRequestMonitorRecord
	inserted       []*OpsInsertUserRequestCaptureInput

	createErr error
	insertErr error
}

func (r *opsUserRequestMonitorRepoStub) GetActiveUserRequestMonitors(ctx context.Context, userID int64, now time.Time) ([]*OpsUserRequestMonitor, error) {
	return r.activeMonitors, nil
}

func (r *opsUserRequestMonitorRepoStub) CreateUserRequestMonitor(ctx context.Context, input *OpsCreateUserRequestMonitorRecord) (*OpsUserRequestMonitor, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.created = append(r.created, input)
	return &OpsUserRequestMonitor{
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
	}, nil
}

func (r *opsUserRequestMonitorRepoStub) InsertUserRequestCapture(ctx context.Context, input *OpsInsertUserRequestCaptureInput) (int64, error) {
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	r.inserted = append(r.inserted, input)
	return int64(len(r.inserted)), nil
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
