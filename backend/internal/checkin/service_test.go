//go:build unit

package checkin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type repositoryStub struct {
	settings        Settings
	result          CheckInResult
	status          UserStatus
	items           []Record
	total           int64
	stats           AdminStats
	lastStatsFilter AdminRecordFilter
	err             error
	updateCalls     int
	checkInCalls    int
}

func (s *repositoryStub) GetSettings(context.Context) (Settings, error) { return s.settings, s.err }
func (s *repositoryStub) UpdateSettings(_ context.Context, settings Settings) (Settings, error) {
	s.updateCalls++
	s.settings = settings
	return settings, s.err
}
func (s *repositoryStub) GetUserStatus(context.Context, int64, time.Time) (UserStatus, error) {
	return s.status, s.err
}
func (s *repositoryStub) CheckIn(context.Context, int64, time.Time, ClientInfo) (CheckInResult, error) {
	s.checkInCalls++
	return s.result, s.err
}
func (s *repositoryStub) ListAdminRecords(context.Context, AdminRecordFilter) ([]Record, int64, error) {
	return s.items, s.total, s.err
}
func (s *repositoryStub) AdminStats(_ context.Context, filter AdminRecordFilter) (AdminStats, error) {
	s.lastStatsFilter = filter
	return s.stats, s.err
}

type invalidatorStub struct {
	calls int
	err   error
}

func (s *invalidatorStub) InvalidateUserBalance(context.Context, int64) error {
	s.calls++
	return s.err
}
func (s *invalidatorStub) InvalidateAuthCacheByUserIDStrict(context.Context, int64) error {
	s.calls++
	return s.err
}

func TestServiceUpdateSettingsValidatesBeforeRepository(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo, nil, nil)
	_, err := service.UpdateSettings(context.Background(), SettingsRequest{Enabled: true, MinReward: "1", MaxReward: "0", Timezone: "UTC", DailyCap: "0"})
	var validationError *SettingsValidationError
	require.ErrorAs(t, err, &validationError)
	require.Zero(t, repo.updateCalls)
}

func TestServiceCacheFailuresDoNotUndoCommittedCheckin(t *testing.T) {
	repo := &repositoryStub{result: CheckInResult{Record: Record{ID: 3, UserID: 17}}}
	balance := &invalidatorStub{err: errors.New("redis unavailable")}
	auth := &invalidatorStub{err: errors.New("redis unavailable")}
	service := NewService(repo, balance, auth)

	result, err := service.CheckIn(context.Background(), 17, time.Now(), ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, int64(3), result.Record.ID)
	require.Equal(t, 1, balance.calls)
	require.Equal(t, 1, auth.calls)
}

func TestServiceAdminStatsAppliesPeriodBoundsWithoutListDate(t *testing.T) {
	businessDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	repo := &repositoryStub{
		settings: Settings{Timezone: "Asia/Shanghai"},
		stats:    AdminStats{UniqueUsers: 3, CheckInCount: 4, TotalAmount: decimal.RequireFromString("2.5")},
	}
	service := NewService(repo, nil, nil)
	now := time.Date(2026, 8, 19, 16, 30, 0, 0, time.UTC)

	stats, err := service.AdminStats(context.Background(), PeriodMonth, AdminRecordFilter{Search: "qq.com", BusinessDate: &businessDate}, now)
	require.NoError(t, err)
	require.Equal(t, PeriodMonth, stats.Period)
	require.Equal(t, "Asia/Shanghai", stats.Timezone)
	require.Equal(t, "2026-08-01", formatBusinessDate(*stats.From))
	require.Equal(t, "2026-08-20", formatBusinessDate(*stats.To))
	require.Nil(t, repo.lastStatsFilter.BusinessDate)
	require.Equal(t, "2026-08-01", formatBusinessDate(*repo.lastStatsFilter.BusinessDateFrom))
	require.Equal(t, "2026-08-20", formatBusinessDate(*repo.lastStatsFilter.BusinessDateTo))
	require.Equal(t, "qq.com", repo.lastStatsFilter.Search)
	require.Equal(t, int64(3), stats.UniqueUsers)
	require.Equal(t, "2.5000", formatAmount(stats.TotalAmount))
}

func TestServiceAdminStatsRejectsUnknownPeriod(t *testing.T) {
	service := NewService(&repositoryStub{settings: Settings{Timezone: "UTC"}}, nil, nil)
	_, err := service.AdminStats(context.Background(), "quarter", AdminRecordFilter{}, time.Now())
	require.ErrorIs(t, err, ErrInvalidStatsPeriod)
}

func TestServiceDoesNotInvalidateCachesForReplay(t *testing.T) {
	repo := &repositoryStub{result: CheckInResult{Record: Record{ID: 3, UserID: 17}, AlreadyCheckedIn: true}}
	balance, auth := &invalidatorStub{}, &invalidatorStub{}
	service := NewService(repo, balance, auth)

	_, err := service.CheckIn(context.Background(), 17, time.Now(), ClientInfo{})
	require.NoError(t, err)
	require.Zero(t, balance.calls)
	require.Zero(t, auth.calls)
}
