package checkin

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Repository interface {
	GetSettings(context.Context) (Settings, error)
	UpdateSettings(context.Context, Settings) (Settings, error)
	GetUserStatus(context.Context, int64, time.Time) (UserStatus, error)
	CheckIn(context.Context, int64, time.Time, ClientInfo) (CheckInResult, error)
	ListAdminRecords(context.Context, AdminRecordFilter) ([]Record, int64, error)
}

type BalanceCacheInvalidator interface {
	InvalidateUserBalance(context.Context, int64) error
}

type AuthCacheInvalidator interface {
	InvalidateAuthCacheByUserIDStrict(context.Context, int64) error
}

type SettingsValidationError struct {
	Cause error
}

func (e *SettingsValidationError) Error() string {
	return fmt.Sprintf("invalid check-in settings: %v", e.Cause)
}

func (e *SettingsValidationError) Unwrap() error { return e.Cause }

type Service struct {
	repository Repository
	balance    BalanceCacheInvalidator
	auth       AuthCacheInvalidator
}

func NewService(repository Repository, balance BalanceCacheInvalidator, auth AuthCacheInvalidator) *Service {
	return &Service{repository: repository, balance: balance, auth: auth}
}

func (s *Service) GetSettings(ctx context.Context) (Settings, error) {
	return s.repository.GetSettings(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, request SettingsRequest) (Settings, error) {
	settings, err := normalizeSettings(request)
	if err != nil {
		return Settings{}, &SettingsValidationError{Cause: err}
	}
	return s.repository.UpdateSettings(ctx, settings)
}

func (s *Service) GetUserStatus(ctx context.Context, userID int64, now time.Time) (UserStatus, error) {
	return s.repository.GetUserStatus(ctx, userID, now)
}

func (s *Service) CheckIn(ctx context.Context, userID int64, now time.Time, client ClientInfo) (CheckInResult, error) {
	result, err := s.repository.CheckIn(ctx, userID, now, client)
	if err != nil || result.AlreadyCheckedIn {
		return result, err
	}
	if s.balance != nil {
		if err := s.balance.InvalidateUserBalance(ctx, userID); err != nil {
			slog.Warn("daily check-in balance cache invalidation failed", "user_id", userID, "error", err)
		}
	}
	if s.auth != nil {
		if err := s.auth.InvalidateAuthCacheByUserIDStrict(ctx, userID); err != nil {
			slog.Warn("daily check-in auth cache invalidation failed", "user_id", userID, "error", err)
		}
	}
	return result, nil
}

func (s *Service) ListAdminRecords(ctx context.Context, filter AdminRecordFilter) ([]Record, int64, error) {
	return s.repository.ListAdminRecords(ctx, filter)
}
