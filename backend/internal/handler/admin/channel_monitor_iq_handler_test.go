//go:build unit

package admin

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorCreateRequestAcceptsIQFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{
		"name": "iq monitor",
		"provider": "openai",
		"interval_seconds": 60,
		"check_mode": "iq",
		"iq_question": "黑色袋子中有苹果味糖果",
		"iq_answer": "21",
		"iq_fuzzy_match": true
	}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/channel-monitors", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var req channelMonitorCreateRequest
	require.NoError(t, c.ShouldBindJSON(&req))
	require.Equal(t, service.MonitorCheckModeIQ, req.CheckMode)
	require.Equal(t, "21", req.IQAnswer)
	require.NotNil(t, req.IQFuzzyMatch)
	require.True(t, *req.IQFuzzyMatch)
	require.Contains(t, req.IQQuestion, "苹果味")
}

func TestChannelMonitorResponsesKeepIQStatus(t *testing.T) {
	checked := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	monitor := channelMonitorToResponse(&service.ChannelMonitor{
		Name:         "iq monitor",
		Provider:     service.MonitorProviderOpenAI,
		CheckMode:    service.MonitorCheckModeIQ,
		IQQuestion:   "糖果题",
		IQAnswer:     "21",
		IQFuzzyMatch: true,
		CreatedAt:    checked,
		UpdatedAt:    checked,
	})
	raw, err := json.Marshal(monitor)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, service.MonitorCheckModeIQ, body["check_mode"])
	require.Equal(t, "21", body["iq_answer"])
	require.Equal(t, true, body["iq_fuzzy_match"])

	result := checkResultToResponse(&service.CheckResult{
		Model:     "gpt-6.1-sol",
		Status:    service.MonitorStatusOperational,
		IqStatus:  service.MonitorIQStatusDown,
		CheckedAt: checked,
	})
	raw, err = json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, service.MonitorIQStatusDown, body["iq_status"])

	history := historyEntryToResponse(&service.ChannelMonitorHistoryEntry{
		ID:        9,
		Model:     "gpt-6.1-sol",
		Status:    service.MonitorStatusOperational,
		IqStatus:  service.MonitorIQStatusTimeout,
		CheckedAt: checked,
	})
	raw, err = json.Marshal(history)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, service.MonitorIQStatusTimeout, body["iq_status"])
}
