package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userUsageRepoFullCapture struct {
	service.UsageLogRepository
	listParams  pagination.PaginationParams
	listFilters usagestats.UsageLogFilters
	statsStart  time.Time
	statsEnd    time.Time
	trendStart  time.Time
	trendEnd    time.Time
	modelStart  time.Time
	modelEnd    time.Time
}

func (s *userUsageRepoFullCapture) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listParams = params
	s.listFilters = filters
	return []service.UsageLog{}, &pagination.PaginationResult{
		Total: 0, Page: params.Page, PageSize: params.PageSize, Pages: 0,
	}, nil
}

func (s *userUsageRepoFullCapture) GetUserStatsAggregated(_ context.Context, _ int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	s.statsStart = startTime
	s.statsEnd = endTime
	return &usagestats.UsageStats{}, nil
}

func (s *userUsageRepoFullCapture) GetUserUsageTrendByUserID(_ context.Context, _ int64, startTime, endTime time.Time, _ string) ([]usagestats.TrendDataPoint, error) {
	s.trendStart = startTime
	s.trendEnd = endTime
	return nil, nil
}

func (s *userUsageRepoFullCapture) GetUserModelStats(_ context.Context, _ int64, startTime, endTime time.Time) ([]usagestats.ModelStat, error) {
	s.modelStart = startTime
	s.modelEnd = endTime
	return nil, nil
}

func newUserUsageAbuseTestRouter(repo *userUsageRepoFullCapture) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	h := NewUsageHandler(usageSvc, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage", h.List)
	router.GET("/usage/stats", h.Stats)
	router.GET("/usage/dashboard/trend", h.DashboardTrend)
	router.GET("/usage/dashboard/models", h.DashboardModels)
	return router
}

func TestUserUsageList_PageSizeCappedAt100(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?page_size=1000", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, userUsageMaxPageSize, repo.listParams.PageSize)
	require.Equal(t, userUsageMaxPageSize, repo.listParams.Limit())
}

func TestUserUsageList_RejectsRangeOver90Days(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?start_date=2024-01-01&end_date=2024-06-01", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
}

func TestUserUsageStats_RejectsRangeOver90Days(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/stats?start_date=2024-01-01&end_date=2024-06-01", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageDashboardTrend_RejectsRangeOver90Days(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/trend?start_date=2024-01-01&end_date=2024-06-01", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageDashboardModels_RejectsRangeOver90Days(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/models?start_date=2024-01-01&end_date=2024-06-01", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageList_AllowsRangeWithin90Days(t *testing.T) {
	repo := &userUsageRepoFullCapture{}
	router := newUserUsageAbuseTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/usage?start_date=2024-01-01&end_date=2024-03-30&page_size=50", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 50, repo.listParams.PageSize)
	require.NotNil(t, repo.listFilters.StartTime)
	require.NotNil(t, repo.listFilters.EndTime)
}
