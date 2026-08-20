//go:build unit

package checkin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func authenticatedContext(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})
		c.Next()
	}
}

func decodeResponseData(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.NoError(t, json.Unmarshal(envelope.Data, target))
}

func TestHandlerUserStatusAndCheckInResponseContract(t *testing.T) {
	now := time.Date(2026, 8, 19, 2, 3, 4, 0, time.UTC)
	record := Record{
		ID: 8, UserID: 17, UserEmail: "u@example.com", Username: "user",
		BusinessDate: now, CheckedAt: now, Timezone: "Asia/Shanghai",
		StreakDays: 7, CycleDay: 7, MilestoneDay: 7,
		BaseReward: decimal.RequireFromString("0.1"), MilestoneBonus: decimal.RequireFromString("1"),
		ActualReward: decimal.RequireFromString("1.1"), Status: StatusAwarded,
		BalanceAfter: decimal.RequireFromString("9.25"),
	}
	repo := &repositoryStub{
		status: UserStatus{
			Enabled: true, CheckedInToday: true, TotalCheckIns: 7,
			TotalReward: decimal.RequireFromString("3.2"), CurrentStreak: 7, CycleDay: 7,
			Balance: decimal.RequireFromString("9.25"), TodayRecord: &record,
			RecentRecords: []Record{record},
		},
		result: CheckInResult{Record: record, AlreadyCheckedIn: true},
	}
	handler := newHandler(NewService(repo, nil, nil))
	handler.now = func() time.Time { return now }
	router := gin.New()
	router.Use(authenticatedContext(17))
	router.GET("/user/checkin", handler.GetUserStatus)
	router.POST("/user/checkin", handler.CheckIn)

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/user/checkin", nil))
	require.Equal(t, http.StatusOK, getRecorder.Code)
	var status map[string]any
	decodeResponseData(t, getRecorder, &status)
	require.Equal(t, "3.2000", status["total_reward"])
	require.Equal(t, "9.2500", status["balance"])
	require.Equal(t, float64(7), status["total_checkins"])

	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/user/checkin", nil))
	require.Equal(t, http.StatusOK, postRecorder.Code)
	var result map[string]any
	decodeResponseData(t, postRecorder, &result)
	require.Equal(t, "2026-08-19", result["business_date"])
	require.Equal(t, "1.1000", result["actual_reward"])
	require.Equal(t, true, result["already_checked_in"])
}

func TestHandlerMapsValidationAndDisabledErrors(t *testing.T) {
	repo := &repositoryStub{err: ErrDisabled}
	handler := newHandler(NewService(repo, nil, nil))
	router := gin.New()
	router.Use(authenticatedContext(17))
	router.POST("/user/checkin", handler.CheckIn)
	router.PUT("/admin/settings", handler.UpdateSettings)

	disabled := httptest.NewRecorder()
	router.ServeHTTP(disabled, httptest.NewRequest(http.MethodPost, "/user/checkin", nil))
	require.Equal(t, http.StatusForbidden, disabled.Code)
	require.Contains(t, disabled.Body.String(), "CHECKIN_DISABLED")

	invalid := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/settings", nil)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(invalid, request)
	require.Equal(t, http.StatusBadRequest, invalid.Code)
}

func TestModuleRegistersIndependentRouteNamespaces(t *testing.T) {
	module := NewModule(nil, nil, nil)
	router := gin.New()
	v1 := router.Group("/api/v1")
	module.RegisterUserRoutes(v1, authenticatedContext(17))
	module.RegisterAdminRoutes(v1, func(c *gin.Context) { c.Next() })

	routes := router.Routes()
	paths := make(map[string]bool, len(routes))
	for _, route := range routes {
		paths[route.Method+" "+route.Path] = true
	}
	require.True(t, paths[http.MethodGet+" /api/v1/user/checkin"])
	require.True(t, paths[http.MethodPost+" /api/v1/user/checkin"])
	require.True(t, paths[http.MethodGet+" /api/v1/admin/affiliates/checkins"])
	require.True(t, paths[http.MethodGet+" /api/v1/admin/affiliates/checkins/stats"])
	require.True(t, paths[http.MethodGet+" /api/v1/admin/affiliates/checkins/settings"])
	require.True(t, paths[http.MethodPut+" /api/v1/admin/affiliates/checkins/settings"])
}

func TestHandlerAdminStatsUsesPeriodAndReturnsPayoutDistribution(t *testing.T) {
	from := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	repo := &repositoryStub{
		settings: Settings{Timezone: "Asia/Shanghai"},
		stats: AdminStats{
			UniqueUsers:  3,
			CheckInCount: 4,
			TotalAmount:  decimal.RequireFromString("2.5"),
			AvgAmount:    decimal.RequireFromString("0.8333"),
			P50Amount:    decimal.RequireFromString("0.8"),
			P90Amount:    decimal.RequireFromString("1.2"),
			MaxAmount:    decimal.RequireFromString("1.5"),
		},
	}
	handler := newHandler(NewService(repo, nil, nil))
	handler.now = func() time.Time { return time.Date(2026, 8, 19, 16, 30, 0, 0, time.UTC) }
	router := gin.New()
	router.GET("/admin/affiliates/checkins/stats", handler.GetAdminStats)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/affiliates/checkins/stats?period=week&search=qq.com", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	var payload map[string]any
	decodeResponseData(t, recorder, &payload)
	require.Equal(t, "week", payload["period"])
	require.Equal(t, "Asia/Shanghai", payload["timezone"])
	require.Equal(t, formatBusinessDate(from), payload["from"])
	require.Equal(t, formatBusinessDate(to), payload["to"])
	require.Equal(t, float64(3), payload["unique_users"])
	require.Equal(t, float64(4), payload["checkin_count"])
	require.Equal(t, "2.5000", payload["total_amount"])
	require.Equal(t, "0.8333", payload["avg_amount"])
	require.Equal(t, "0.8000", payload["p50_amount"])
	require.Equal(t, "1.2000", payload["p90_amount"])
	require.Equal(t, "1.5000", payload["max_amount"])
	require.Equal(t, "qq.com", repo.lastStatsFilter.Search)
}

func TestHandlerAdminStatsRejectsUnknownPeriod(t *testing.T) {
	handler := newHandler(NewService(&repositoryStub{settings: Settings{Timezone: "UTC"}}, nil, nil))
	router := gin.New()
	router.GET("/stats", handler.GetAdminStats)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stats?period=quarter", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_CHECKIN_STATS_PERIOD")
}
