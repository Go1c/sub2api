package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	opsUserRequestMonitorMaxBodyBytes  = 256 * 1024
	opsUserRequestMonitorDefaultRetain = 7
	opsUserRequestMonitorMaxRetain     = 30
	opsUserRequestMonitorMaxDuration   = 24 * time.Hour
	opsUserRequestMonitorMaxPerMinute  = 120
)

var ErrOpsUserRequestMonitorAlreadyActive = errors.New("ops user request monitor already active")

type opsUserRequestMonitorUserLookup interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type opsUserRequestCaptureLimiter interface {
	Allow(ctx context.Context, monitorID int64, captureMinute time.Time, maxPerMinute int) (bool, error)
}

type opsUserRequestMonitorCacheEntry struct {
	monitors []*OpsUserRequestMonitor
	expires  time.Time
}

// OpsUserRequestMonitorService owns targeted raw request body capture for admins.
// Capture is best-effort: failures are logged and never block the gateway path.
type OpsUserRequestMonitorService struct {
	opsRepo    OpsRepository
	userLookup opsUserRequestMonitorUserLookup
	limiter    opsUserRequestCaptureLimiter

	sample func(percent int) bool
	now    func() time.Time

	cacheMu sync.Mutex
	cache   map[int64]opsUserRequestMonitorCacheEntry

	// createMonitorMu serializes admin create flows so two concurrent requests
	// cannot both pass the "no active monitor" pre-check in the same process.
	createMonitorMu sync.Mutex

	captureTimeout time.Duration
	cacheTTL       time.Duration
	cleanupEvery   time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
}

func NewOpsUserRequestMonitorService(opsRepo OpsRepository, userRepo UserRepository, redisClient *redis.Client) *OpsUserRequestMonitorService {
	var limiter opsUserRequestCaptureLimiter
	if redisClient != nil {
		limiter = &opsUserRequestRedisLimiter{client: redisClient}
	}
	return &OpsUserRequestMonitorService{
		opsRepo:        opsRepo,
		userLookup:     userRepo,
		limiter:        limiter,
		sample:         defaultOpsUserRequestMonitorSample,
		now:            time.Now,
		cache:          make(map[int64]opsUserRequestMonitorCacheEntry),
		captureTimeout: 2 * time.Second,
		cacheTTL:       2 * time.Second,
		cleanupEvery:   time.Minute,
		stopCh:         make(chan struct{}),
	}
}

func (s *OpsUserRequestMonitorService) Start() {
	if s == nil || s.opsRepo == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.cleanupLoop()
	})
}

func (s *OpsUserRequestMonitorService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *OpsUserRequestMonitorService) CreateMonitor(ctx context.Context, input *OpsCreateUserRequestMonitorInput) (*OpsUserRequestMonitor, error) {
	if s == nil || s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if input == nil {
		return nil, infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID", "monitor input is required")
	}
	retentionDays := input.RetentionDays
	if retentionDays == 0 {
		retentionDays = opsUserRequestMonitorDefaultRetain
	}
	if err := validateOpsUserRequestMonitorInput(input.UserID, input.DurationSeconds, input.MaxCapturesPerMinute, input.SampleRatePercent, retentionDays, input.CreatedBy); err != nil {
		return nil, err
	}
	s.createMonitorMu.Lock()
	defer s.createMonitorMu.Unlock()
	if s.userLookup == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_USER_LOOKUP_UNAVAILABLE", "user lookup unavailable")
	}
	user, err := s.userLookup.GetByID(ctx, input.UserID)
	if err != nil || user == nil {
		if errors.Is(err, sql.ErrNoRows) || err == nil {
			return nil, infraerrors.NotFound("OPS_USER_REQUEST_MONITOR_USER_NOT_FOUND", "target user not found")
		}
		return nil, infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_USER_LOAD_FAILED", "failed to load target user").WithCause(err)
	}

	now := s.currentTime().UTC()
	active, err := s.opsRepo.GetActiveUserRequestMonitors(ctx, input.UserID, now)
	if err != nil {
		return nil, infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_ACTIVE_CHECK_FAILED", "failed to check active monitors").WithCause(err)
	}
	if hasActiveOpsUserRequestMonitor(active, now) {
		return nil, infraerrors.Conflict("OPS_USER_REQUEST_MONITOR_ALREADY_ACTIVE", "target user already has an active request monitor")
	}

	record := &OpsCreateUserRequestMonitorRecord{
		UserID:               input.UserID,
		TargetEmail:          strings.TrimSpace(user.Email),
		DurationSeconds:      input.DurationSeconds,
		MaxCapturesPerMinute: input.MaxCapturesPerMinute,
		SampleRatePercent:    input.SampleRatePercent,
		RetentionDays:        retentionDays,
		CreatedBy:            input.CreatedBy,
		CreatedAt:            now,
		StartsAt:             now,
		EndsAt:               now.Add(time.Duration(input.DurationSeconds) * time.Second),
	}
	monitor, err := s.opsRepo.CreateUserRequestMonitor(ctx, record)
	if err != nil {
		if errors.Is(err, ErrOpsUserRequestMonitorAlreadyActive) {
			return nil, infraerrors.Conflict("OPS_USER_REQUEST_MONITOR_ALREADY_ACTIVE", "target user already has an active request monitor")
		}
		return nil, infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_CREATE_FAILED", "failed to create request monitor").WithCause(err)
	}
	s.invalidateUser(input.UserID)
	return monitor, nil
}

func (s *OpsUserRequestMonitorService) ListMonitors(ctx context.Context, filter *OpsUserRequestMonitorFilter) ([]*OpsUserRequestMonitor, int64, error) {
	if s == nil || s.opsRepo == nil {
		return []*OpsUserRequestMonitor{}, 0, nil
	}
	normalizeOpsUserRequestMonitorFilter(filter)
	items, total, err := s.opsRepo.ListUserRequestMonitors(ctx, filter)
	if err != nil {
		return nil, 0, infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_LIST_FAILED", "failed to list request monitors").WithCause(err)
	}
	now := s.currentTime()
	for _, item := range items {
		normalizeOpsUserRequestMonitorStatus(item, now)
	}
	return items, total, nil
}

func (s *OpsUserRequestMonitorService) StopMonitor(ctx context.Context, id int64) (*OpsUserRequestMonitor, error) {
	if s == nil || s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_ID", "invalid monitor id")
	}
	monitor, err := s.opsRepo.StopUserRequestMonitor(ctx, id, s.currentTime().UTC())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("OPS_USER_REQUEST_MONITOR_NOT_FOUND", "request monitor not found")
		}
		return nil, infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_STOP_FAILED", "failed to stop request monitor").WithCause(err)
	}
	if monitor != nil {
		s.invalidateUser(monitor.UserID)
	}
	return monitor, nil
}

func (s *OpsUserRequestMonitorService) DeleteMonitor(ctx context.Context, id int64) error {
	if s == nil || s.opsRepo == nil {
		return infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if id <= 0 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_ID", "invalid monitor id")
	}
	monitor, err := s.opsRepo.GetUserRequestMonitorByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.NotFound("OPS_USER_REQUEST_MONITOR_NOT_FOUND", "request monitor not found")
		}
		return infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_LOAD_FAILED", "failed to load request monitor").WithCause(err)
	}
	deleted, err := s.opsRepo.DeleteUserRequestMonitor(ctx, id)
	if err != nil {
		return infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_DELETE_FAILED", "failed to delete request monitor").WithCause(err)
	}
	if !deleted {
		return infraerrors.NotFound("OPS_USER_REQUEST_MONITOR_NOT_FOUND", "request monitor not found")
	}
	if monitor != nil {
		s.invalidateUser(monitor.UserID)
	}
	return nil
}

func (s *OpsUserRequestMonitorService) ListCaptures(ctx context.Context, filter *OpsUserRequestCaptureFilter) ([]*OpsUserRequestCapture, int64, error) {
	if s == nil || s.opsRepo == nil {
		return []*OpsUserRequestCapture{}, 0, nil
	}
	if filter == nil {
		filter = &OpsUserRequestCaptureFilter{}
	}
	if filter.MonitorID <= 0 {
		return nil, 0, infraerrors.BadRequest("OPS_USER_REQUEST_CAPTURE_INVALID_MONITOR_ID", "invalid monitor id")
	}
	normalizeOpsUserRequestCaptureFilter(filter)
	items, total, err := s.opsRepo.ListUserRequestCaptures(ctx, filter)
	if err != nil {
		return nil, 0, infraerrors.InternalServer("OPS_USER_REQUEST_CAPTURE_LIST_FAILED", "failed to list request captures").WithCause(err)
	}
	return items, total, nil
}

func (s *OpsUserRequestMonitorService) GetCapture(ctx context.Context, monitorID, captureID int64) (*OpsUserRequestCapture, error) {
	if s == nil || s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if monitorID <= 0 || captureID <= 0 {
		return nil, infraerrors.BadRequest("OPS_USER_REQUEST_CAPTURE_INVALID_ID", "invalid capture id")
	}
	capture, err := s.opsRepo.GetUserRequestCapture(ctx, monitorID, captureID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("OPS_USER_REQUEST_CAPTURE_NOT_FOUND", "request capture not found")
		}
		return nil, infraerrors.InternalServer("OPS_USER_REQUEST_CAPTURE_LOAD_FAILED", "failed to load request capture").WithCause(err)
	}
	return capture, nil
}

func (s *OpsUserRequestMonitorService) ExportCapturesJSONL(ctx context.Context, monitorID int64, w io.Writer) error {
	if s == nil || s.opsRepo == nil {
		return infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if monitorID <= 0 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_CAPTURE_INVALID_MONITOR_ID", "invalid monitor id")
	}
	if w == nil {
		return infraerrors.BadRequest("OPS_USER_REQUEST_CAPTURE_EXPORT_INVALID_WRITER", "export writer is required")
	}
	if _, err := s.opsRepo.GetUserRequestMonitorByID(ctx, monitorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.NotFound("OPS_USER_REQUEST_MONITOR_NOT_FOUND", "request monitor not found")
		}
		return infraerrors.InternalServer("OPS_USER_REQUEST_MONITOR_LOAD_FAILED", "failed to load request monitor").WithCause(err)
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := s.opsRepo.StreamUserRequestCaptures(ctx, monitorID, func(capture *OpsUserRequestCapture) error {
		return encoder.Encode(capture)
	}); err != nil {
		return infraerrors.InternalServer("OPS_USER_REQUEST_CAPTURE_EXPORT_FAILED", "failed to export request captures").WithCause(err)
	}
	return nil
}

func (s *OpsUserRequestMonitorService) DeleteCapture(ctx context.Context, monitorID, captureID int64) error {
	if s == nil || s.opsRepo == nil {
		return infraerrors.ServiceUnavailable("OPS_USER_REQUEST_MONITOR_UNAVAILABLE", "user request monitor service unavailable")
	}
	if monitorID <= 0 || captureID <= 0 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_CAPTURE_INVALID_ID", "invalid capture id")
	}
	deleted, err := s.opsRepo.DeleteUserRequestCapture(ctx, monitorID, captureID)
	if err != nil {
		return infraerrors.InternalServer("OPS_USER_REQUEST_CAPTURE_DELETE_FAILED", "failed to delete request capture").WithCause(err)
	}
	if !deleted {
		return infraerrors.NotFound("OPS_USER_REQUEST_CAPTURE_NOT_FOUND", "request capture not found")
	}
	return nil
}

func (s *OpsUserRequestMonitorService) CaptureClientRequestIfEnabled(ctx context.Context, input *OpsCaptureClientRequestInput) {
	if s == nil || input == nil {
		return
	}
	snapshot := snapshotOpsCaptureClientRequestInput(input)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[OpsUserRequestMonitor] capture panic recovered: %v", r)
			}
		}()
		captureCtx, cancel := context.WithTimeout(context.Background(), s.captureTimeout)
		defer cancel()
		_ = s.CaptureClientRequestSync(captureCtx, snapshot)
	}()
}

func (s *OpsUserRequestMonitorService) CaptureClientRequestSync(ctx context.Context, input *OpsCaptureClientRequestInput) error {
	if s == nil || s.opsRepo == nil || input == nil || input.UserID <= 0 {
		return nil
	}
	now := s.currentTime().UTC()
	monitors, err := s.activeMonitorsForUser(ctx, input.UserID, now)
	if err != nil {
		log.Printf("[OpsUserRequestMonitor] active lookup failed: %v", err)
		return nil
	}
	if len(monitors) == 0 {
		return nil
	}
	if s.limiter == nil {
		// Redis-backed limiter is required for distributed per-minute semantics.
		return nil
	}

	captureMinute := now.Truncate(time.Minute)
	for _, monitor := range monitors {
		if !isOpsUserRequestMonitorActive(monitor, now) {
			continue
		}
		allowed, err := s.limiter.Allow(ctx, monitor.ID, captureMinute, monitor.MaxCapturesPerMinute)
		if err != nil {
			log.Printf("[OpsUserRequestMonitor] limiter failed monitor_id=%d: %v", monitor.ID, err)
			continue
		}
		if !allowed {
			continue
		}
		if s.sample != nil && !s.sample(monitor.SampleRatePercent) {
			continue
		}

		body, bodyBytes, truncated := prepareOpsUserRequestMonitorBody(input.Body, input.BodyBytes)
		insert := &OpsInsertUserRequestCaptureInput{
			MonitorID:         monitor.ID,
			UserID:            input.UserID,
			APIKeyID:          cloneOpsUserRequestMonitorInt64Ptr(input.APIKeyID),
			AccountID:         cloneOpsUserRequestMonitorInt64Ptr(input.AccountID),
			GroupID:           cloneOpsUserRequestMonitorInt64Ptr(input.GroupID),
			RequestID:         truncateString(strings.TrimSpace(input.RequestID), 64),
			Model:             truncateString(strings.TrimSpace(input.Model), 100),
			InboundEndpoint:   truncateString(strings.TrimSpace(input.InboundEndpoint), 256),
			Method:            truncateString(strings.TrimSpace(input.Method), 16),
			ContentType:       truncateString(strings.TrimSpace(input.ContentType), 128),
			Body:              body,
			BodyBytes:         bodyBytes,
			BodyTruncated:     truncated,
			SampleRatePercent: monitor.SampleRatePercent,
			CaptureMinute:     captureMinute,
			CreatedAt:         now,
			ExpiresAt:         now.AddDate(0, 0, monitor.RetentionDays),
		}
		if _, err := s.opsRepo.InsertUserRequestCapture(ctx, insert); err != nil {
			log.Printf("[OpsUserRequestMonitor] insert capture failed monitor_id=%d user_id=%d: %v", monitor.ID, input.UserID, err)
			continue
		}
	}
	return nil
}

func (s *OpsUserRequestMonitorService) CleanupOnce(ctx context.Context) {
	if s == nil || s.opsRepo == nil {
		return
	}
	now := s.currentTime().UTC()
	if _, err := s.opsRepo.ExpireUserRequestMonitors(ctx, now); err != nil {
		log.Printf("[OpsUserRequestMonitor] expire monitors failed: %v", err)
	}
	if _, err := s.opsRepo.DeleteExpiredUserRequestCaptures(ctx, now); err != nil {
		log.Printf("[OpsUserRequestMonitor] delete expired captures failed: %v", err)
	}
}

func (s *OpsUserRequestMonitorService) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			s.CleanupOnce(ctx)
			cancel()
		case <-s.stopCh:
			return
		}
	}
}

func (s *OpsUserRequestMonitorService) activeMonitorsForUser(ctx context.Context, userID int64, now time.Time) ([]*OpsUserRequestMonitor, error) {
	if s == nil || s.opsRepo == nil || userID <= 0 {
		return nil, nil
	}
	if s.cacheTTL > 0 {
		s.cacheMu.Lock()
		if entry, ok := s.cache[userID]; ok && now.Before(entry.expires) {
			cached := cloneOpsUserRequestMonitors(entry.monitors)
			s.cacheMu.Unlock()
			return filterOpsUserRequestMonitorsActive(cached, now), nil
		}
		s.cacheMu.Unlock()
	}
	monitors, err := s.opsRepo.GetActiveUserRequestMonitors(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	monitors = filterOpsUserRequestMonitorsActive(monitors, now)
	if s.cacheTTL > 0 {
		s.cacheMu.Lock()
		s.cache[userID] = opsUserRequestMonitorCacheEntry{
			monitors: cloneOpsUserRequestMonitors(monitors),
			expires:  now.Add(s.cacheTTL),
		}
		s.cacheMu.Unlock()
	}
	return monitors, nil
}

func (s *OpsUserRequestMonitorService) invalidateUser(userID int64) {
	if s == nil || userID <= 0 {
		return
	}
	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()
}

func (s *OpsUserRequestMonitorService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func validateOpsUserRequestMonitorInput(userID int64, durationSeconds, maxCapturesPerMinute, sampleRatePercent, retentionDays int, createdBy int64) error {
	if userID <= 0 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_USER_ID", "invalid user id")
	}
	if createdBy <= 0 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_OPERATOR", "invalid operator")
	}
	if durationSeconds <= 0 || time.Duration(durationSeconds)*time.Second > opsUserRequestMonitorMaxDuration {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_DURATION", "duration must be between 1 second and 24 hours")
	}
	if maxCapturesPerMinute <= 0 || maxCapturesPerMinute > opsUserRequestMonitorMaxPerMinute {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_RATE_LIMIT", "max captures per minute must be between 1 and 120")
	}
	if sampleRatePercent <= 0 || sampleRatePercent > 100 {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_SAMPLE_RATE", "sample rate must be between 1 and 100")
	}
	if retentionDays <= 0 || retentionDays > opsUserRequestMonitorMaxRetain {
		return infraerrors.BadRequest("OPS_USER_REQUEST_MONITOR_INVALID_RETENTION", "retention days must be between 1 and 30")
	}
	return nil
}

func normalizeOpsUserRequestMonitorFilter(filter *OpsUserRequestMonitorFilter) {
	if filter == nil {
		return
	}
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.UserQuery = strings.TrimSpace(filter.UserQuery)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
}

func normalizeOpsUserRequestCaptureFilter(filter *OpsUserRequestCaptureFilter) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
}

func hasActiveOpsUserRequestMonitor(monitors []*OpsUserRequestMonitor, now time.Time) bool {
	for _, monitor := range monitors {
		if isOpsUserRequestMonitorActive(monitor, now) {
			return true
		}
	}
	return false
}

func isOpsUserRequestMonitorActive(monitor *OpsUserRequestMonitor, now time.Time) bool {
	if monitor == nil {
		return false
	}
	if monitor.Status != OpsUserRequestMonitorStatusActive {
		return false
	}
	if !monitor.StartsAt.IsZero() && now.Before(monitor.StartsAt) {
		return false
	}
	if !monitor.EndsAt.IsZero() && !now.Before(monitor.EndsAt) {
		return false
	}
	return true
}

func normalizeOpsUserRequestMonitorStatus(monitor *OpsUserRequestMonitor, now time.Time) {
	if monitor == nil {
		return
	}
	if monitor.Status == OpsUserRequestMonitorStatusActive && !monitor.EndsAt.IsZero() && !now.Before(monitor.EndsAt) {
		monitor.Status = OpsUserRequestMonitorStatusExpired
	}
}

func filterOpsUserRequestMonitorsActive(monitors []*OpsUserRequestMonitor, now time.Time) []*OpsUserRequestMonitor {
	out := make([]*OpsUserRequestMonitor, 0, len(monitors))
	for _, monitor := range monitors {
		if isOpsUserRequestMonitorActive(monitor, now) {
			out = append(out, monitor)
		}
	}
	return out
}

func cloneOpsUserRequestMonitors(in []*OpsUserRequestMonitor) []*OpsUserRequestMonitor {
	out := make([]*OpsUserRequestMonitor, 0, len(in))
	for _, item := range in {
		if item == nil {
			continue
		}
		copyItem := *item
		out = append(out, &copyItem)
	}
	return out
}

func prepareOpsUserRequestMonitorBody(raw []byte, originalBytes int) (body string, bodyBytes int, truncated bool) {
	bodyBytes = originalBytes
	if bodyBytes < len(raw) {
		bodyBytes = len(raw)
	}
	if bodyBytes == 0 {
		return "", 0, false
	}
	if bodyBytes <= opsUserRequestMonitorMaxBodyBytes {
		return string(raw), bodyBytes, false
	}
	if len(raw) > opsUserRequestMonitorMaxBodyBytes {
		return string(raw[:opsUserRequestMonitorMaxBodyBytes]), bodyBytes, true
	}
	return string(raw), bodyBytes, true
}

func snapshotOpsCaptureClientRequestInput(input *OpsCaptureClientRequestInput) *OpsCaptureClientRequestInput {
	if input == nil {
		return nil
	}
	out := *input
	out.APIKeyID = cloneOpsUserRequestMonitorInt64Ptr(input.APIKeyID)
	out.AccountID = cloneOpsUserRequestMonitorInt64Ptr(input.AccountID)
	out.GroupID = cloneOpsUserRequestMonitorInt64Ptr(input.GroupID)
	if out.BodyBytes < len(input.Body) {
		out.BodyBytes = len(input.Body)
	}
	if len(input.Body) > 0 {
		body := input.Body
		if len(body) > opsUserRequestMonitorMaxBodyBytes {
			body = body[:opsUserRequestMonitorMaxBodyBytes]
		}
		out.Body = append([]byte(nil), body...)
	}
	return &out
}

func cloneOpsUserRequestMonitorInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func defaultOpsUserRequestMonitorSample(percent int) bool {
	if percent >= 100 {
		return true
	}
	if percent <= 0 {
		return false
	}
	return rand.Intn(100)+1 <= percent
}

type opsUserRequestRedisLimiter struct {
	client *redis.Client
}

func (l *opsUserRequestRedisLimiter) Allow(ctx context.Context, monitorID int64, captureMinute time.Time, maxPerMinute int) (bool, error) {
	if l == nil || l.client == nil {
		return false, fmt.Errorf("redis client unavailable")
	}
	key := fmt.Sprintf("ops:user-request-monitor:%d:%s", monitorID, captureMinute.UTC().Format("200601021504"))
	n, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := l.client.Expire(ctx, key, 2*time.Minute).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(maxPerMinute), nil
}
