//go:build unit

package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserMonitorViewToItemExposesIQTimeline(t *testing.T) {
	checked := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	latency := 1539
	ping := 24
	item := userMonitorViewToItem(&service.UserMonitorView{
		ID:        7,
		Name:      "Claude-反重力反代",
		Provider:  service.MonitorProviderAnthropic,
		CheckMode: service.MonitorCheckModeIQ,
		Timeline: []service.UserMonitorTimelinePoint{{
			Status:        service.MonitorStatusOperational,
			IqStatus:      service.MonitorIQStatusOK,
			LatencyMs:     &latency,
			PingLatencyMs: &ping,
			CheckedAt:     checked,
		}},
		IQTimeline: []service.UserMonitorTimelinePoint{{
			Status:        service.MonitorIQStatusOK,
			IqStatus:      service.MonitorIQStatusOK,
			LatencyMs:     &latency,
			PingLatencyMs: &ping,
			CheckedAt:     checked,
		}},
	}, false)

	raw, err := json.Marshal(item)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, service.MonitorCheckModeIQ, body["check_mode"])

	iq, ok := body["iq_timeline"].([]any)
	require.True(t, ok, "iq monitors must return a second timeline, got %s", raw)
	require.Len(t, iq, 1)
	point := iq[0].(map[string]any)
	require.Equal(t, service.MonitorIQStatusOK, point["status"])
	require.Equal(t, service.MonitorIQStatusOK, point["iq_status"])
}
