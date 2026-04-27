package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterPublicRoutes_ChannelMonitorListDoesNotRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &publicChannelMonitorRepo{
		monitors: []*service.ChannelMonitor{
			{
				ID:           42,
				Name:         "Claude-Opus-Max",
				Provider:     service.MonitorProviderAnthropic,
				PrimaryModel: "claude-opus-4-7",
				GroupName:    "Anthropic",
				Enabled:      true,
			},
		},
		latest: map[int64][]*service.ChannelMonitorLatest{
			42: {
				{
					Model:         "claude-opus-4-7",
					Status:        service.MonitorStatusOperational,
					LatencyMs:     intPtr(1999),
					PingLatencyMs: intPtr(9),
					CheckedAt:     time.Now().UTC(),
				},
			},
		},
	}
	svc := service.NewChannelMonitorService(repo, nil)
	h := &handler.Handlers{
		ChannelMonitor: handler.NewChannelMonitorUserHandler(svc, nil),
	}

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterPublicRoutes(v1, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/channel-monitors", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				PrimaryStatus string `json:"primary_status"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, int64(42), body.Data.Items[0].ID)
	require.Equal(t, "Claude-Opus-Max", body.Data.Items[0].Name)
	require.Equal(t, service.MonitorStatusOperational, body.Data.Items[0].PrimaryStatus)
}

type publicChannelMonitorRepo struct {
	monitors []*service.ChannelMonitor
	latest   map[int64][]*service.ChannelMonitorLatest
}

func (r *publicChannelMonitorRepo) Create(context.Context, *service.ChannelMonitor) error {
	return nil
}

func (r *publicChannelMonitorRepo) GetByID(_ context.Context, id int64) (*service.ChannelMonitor, error) {
	for _, m := range r.monitors {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, service.ErrChannelMonitorNotFound
}

func (r *publicChannelMonitorRepo) Update(context.Context, *service.ChannelMonitor) error {
	return nil
}

func (r *publicChannelMonitorRepo) Delete(context.Context, int64) error {
	return nil
}

func (r *publicChannelMonitorRepo) List(context.Context, service.ChannelMonitorListParams) ([]*service.ChannelMonitor, int64, error) {
	return r.monitors, int64(len(r.monitors)), nil
}

func (r *publicChannelMonitorRepo) ListEnabled(context.Context) ([]*service.ChannelMonitor, error) {
	return r.monitors, nil
}

func (r *publicChannelMonitorRepo) MarkChecked(context.Context, int64, time.Time) error {
	return nil
}

func (r *publicChannelMonitorRepo) InsertHistoryBatch(context.Context, []*service.ChannelMonitorHistoryRow) error {
	return nil
}

func (r *publicChannelMonitorRepo) DeleteHistoryBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *publicChannelMonitorRepo) ListHistory(context.Context, int64, string, int) ([]*service.ChannelMonitorHistoryEntry, error) {
	return nil, nil
}

func (r *publicChannelMonitorRepo) ListLatestPerModel(_ context.Context, monitorID int64) ([]*service.ChannelMonitorLatest, error) {
	return r.latest[monitorID], nil
}

func (r *publicChannelMonitorRepo) ComputeAvailability(_ context.Context, monitorID int64, windowDays int) ([]*service.ChannelMonitorAvailability, error) {
	return []*service.ChannelMonitorAvailability{
		{
			Model:           "claude-opus-4-7",
			WindowDays:      windowDays,
			AvailabilityPct: 100,
		},
	}, nil
}

func (r *publicChannelMonitorRepo) ListLatestForMonitorIDs(_ context.Context, ids []int64) (map[int64][]*service.ChannelMonitorLatest, error) {
	out := make(map[int64][]*service.ChannelMonitorLatest, len(ids))
	for _, id := range ids {
		out[id] = r.latest[id]
	}
	return out, nil
}

func (r *publicChannelMonitorRepo) ComputeAvailabilityForMonitors(_ context.Context, ids []int64, windowDays int) (map[int64][]*service.ChannelMonitorAvailability, error) {
	out := make(map[int64][]*service.ChannelMonitorAvailability, len(ids))
	for _, id := range ids {
		out[id] = []*service.ChannelMonitorAvailability{
			{
				Model:           "claude-opus-4-7",
				WindowDays:      windowDays,
				AvailabilityPct: 100,
			},
		}
	}
	return out, nil
}

func (r *publicChannelMonitorRepo) ListRecentHistoryForMonitors(_ context.Context, ids []int64, primaryModels map[int64]string, _ int) (map[int64][]*service.ChannelMonitorHistoryEntry, error) {
	now := time.Now().UTC()
	out := make(map[int64][]*service.ChannelMonitorHistoryEntry, len(ids))
	for _, id := range ids {
		out[id] = []*service.ChannelMonitorHistoryEntry{
			{
				Model:         primaryModels[id],
				Status:        service.MonitorStatusOperational,
				LatencyMs:     intPtr(1999),
				PingLatencyMs: intPtr(9),
				CheckedAt:     now,
			},
		}
	}
	return out, nil
}

func (r *publicChannelMonitorRepo) UpsertDailyRollupsFor(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *publicChannelMonitorRepo) DeleteRollupsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *publicChannelMonitorRepo) LoadAggregationWatermark(context.Context) (*time.Time, error) {
	return nil, nil
}

func (r *publicChannelMonitorRepo) UpdateAggregationWatermark(context.Context, time.Time) error {
	return nil
}

func intPtr(v int) *int {
	return &v
}
