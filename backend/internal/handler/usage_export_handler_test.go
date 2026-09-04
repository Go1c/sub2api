package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageExportRepoStub struct {
	service.UsageLogRepository
	userID  int64
	afterID int64
	since   *time.Time
	limit   int
	rows    []service.UsageExportRow
}

func (s *usageExportRepoStub) ListExport(_ context.Context, userID, afterID int64, since *time.Time, limit int) ([]service.UsageExportRow, error) {
	s.userID = userID
	s.afterID = afterID
	s.since = since
	s.limit = limit
	return s.rows, nil
}

func newUsageExportTestRouter(repo *usageExportRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	h := NewUsageHandler(usageSvc, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/export", h.Export)
	return router
}

func TestUsageExport_RequiresAfterIDOrSince(t *testing.T) {
	router := newUsageExportTestRouter(&usageExportRepoStub{})
	req := httptest.NewRequest(http.MethodGet, "/usage/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsageExport_RejectsZeroAfterID(t *testing.T) {
	router := newUsageExportTestRouter(&usageExportRepoStub{})
	req := httptest.NewRequest(http.MethodGet, "/usage/export?after_id=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsageExport_RejectsSinceOlderThan24h(t *testing.T) {
	router := newUsageExportTestRouter(&usageExportRepoStub{})
	since := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/usage/export?since="+since, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsageExport_CapsLimitAndPages(t *testing.T) {
	corr := "downstream-uuid"
	rows := make([]service.UsageExportRow, 501)
	for i := range rows {
		rows[i] = service.UsageExportRow{
			ID:            int64(101 + i),
			CorrelationID: &corr,
			RequestID:     "client:downstream-uuid",
			ActualCost:    0.01,
		}
	}
	repo := &usageExportRepoStub{rows: rows}
	router := newUsageExportTestRouter(repo)

	since := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/usage/export?since="+since+"&limit=1000", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.userID)
	require.Equal(t, int64(0), repo.afterID)
	require.NotNil(t, repo.since)
	require.Equal(t, 501, repo.limit)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(0), body["code"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 500)
	require.Equal(t, true, data["has_more"])
	require.Equal(t, float64(600), data["next_after_id"])
}

func TestUsageExport_AfterIDWinsOverSince(t *testing.T) {
	repo := &usageExportRepoStub{rows: []service.UsageExportRow{{ID: 200}}}
	router := newUsageExportTestRouter(repo)
	since := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/usage/export?after_id=150&since="+since, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(150), repo.afterID)
	require.Nil(t, repo.since)
	require.Equal(t, 101, repo.limit)
}
