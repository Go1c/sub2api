package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type wasteStatsServiceStub struct {
	result service.WasteStatsResult
	query  service.WasteStatsQuery
	called bool
}

func (s *wasteStatsServiceStub) GetWasteStats(_ context.Context, query service.WasteStatsQuery) (service.WasteStatsResult, error) {
	s.called = true
	s.query = query
	return s.result, nil
}

func TestAdminWasteStatsDefaultsRangeToLast30Days(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	stub := &wasteStatsServiceStub{}
	handler := NewSubscriptionWasteStatsHandler(stub)
	handler.now = func() time.Time { return now }

	router := gin.New()
	router.GET("/admin/subscriptions/waste-stats", handler.Get)
	req := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/waste-stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !stub.called {
		t.Fatal("expected service to be called")
	}
	if !stub.query.StartTime.Equal(now.AddDate(0, 0, -30)) || !stub.query.EndTime.Equal(now) {
		t.Fatalf("expected last 30 days, got %s - %s", stub.query.StartTime, stub.query.EndTime)
	}
	if stub.query.Window != "all" {
		t.Fatalf("expected default window all, got %q", stub.query.Window)
	}
	if stub.query.Projection != service.WasteStatsProjectionFull {
		t.Fatalf("expected full projection, got %q", stub.query.Projection)
	}
}

func TestAdminWasteStatsParsesFiltersAndReturnsProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	stub := &wasteStatsServiceStub{result: service.WasteStatsResult{
		StartTime:        start,
		EndTime:          end,
		Window:           "weekly",
		TotalWastedUSD:   3.5,
		TotalWastedRatio: 0.25,
	}}
	handler := NewSubscriptionWasteStatsHandler(stub)

	router := gin.New()
	router.GET("/admin/subscriptions/waste-stats", handler.Get)
	req := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/waste-stats?start=2026-05-01T00:00:00Z&end=2026-05-28T00:00:00Z&plan_id=7&user_id=9&window=weekly", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if stub.query.PlanID == nil || *stub.query.PlanID != 7 {
		t.Fatalf("expected plan_id 7, got %v", stub.query.PlanID)
	}
	if stub.query.UserID == nil || *stub.query.UserID != 9 {
		t.Fatalf("expected user_id 9, got %v", stub.query.UserID)
	}
	var body struct {
		Data service.WasteStatsResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.TotalWastedUSD != 3.5 || body.Data.TotalWastedRatio != 0.25 {
		t.Fatalf("unexpected response data: %+v", body.Data)
	}
}

func TestAdminWasteStatsDateOnlyEndIsExclusiveNextDay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &wasteStatsServiceStub{}
	handler := NewSubscriptionWasteStatsHandler(stub)

	router := gin.New()
	router.GET("/admin/subscriptions/waste-stats", handler.Get)
	req := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/waste-stats?start=2026-05-01&end=2026-05-28", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	want := time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)
	if !stub.query.EndTime.Equal(want) {
		t.Fatalf("expected date-only end to become %s, got %s", want, stub.query.EndTime)
	}
}

func TestAdminWasteStatsProjectionEndpointsSetFocusedProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &wasteStatsServiceStub{result: service.WasteStatsResult{
		ByPlan: []service.PlanWasteBucket{{PlanName: "starter"}},
		TimeSeries: []service.WasteTimeBucket{{
			BucketStart: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		}},
	}}
	handler := NewSubscriptionWasteStatsHandler(stub)

	router := gin.New()
	router.GET("/admin/subscriptions/waste-stats/by-plan", handler.GetByPlan)
	router.GET("/admin/subscriptions/waste-stats/time-series", handler.GetTimeSeries)

	byPlanReq := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/waste-stats/by-plan", nil)
	byPlanResp := httptest.NewRecorder()
	router.ServeHTTP(byPlanResp, byPlanReq)
	if byPlanResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for by-plan, got %d body=%s", byPlanResp.Code, byPlanResp.Body.String())
	}
	if stub.query.Projection != service.WasteStatsProjectionByPlan {
		t.Fatalf("expected by-plan projection, got %q", stub.query.Projection)
	}

	timeReq := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/waste-stats/time-series", nil)
	timeResp := httptest.NewRecorder()
	router.ServeHTTP(timeResp, timeReq)
	if timeResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for time-series, got %d body=%s", timeResp.Code, timeResp.Body.String())
	}
	if stub.query.Projection != service.WasteStatsProjectionTimeSeries {
		t.Fatalf("expected time-series projection, got %q", stub.query.Projection)
	}
}
