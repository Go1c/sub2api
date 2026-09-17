//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCheck_IQSendsCandyQuizNotArithmetic(t *testing.T) {
	h := &openAICaptureHandler{
		rawResponse: `{"choices":[{"message":{"content":"最少要取 21 颗。"}}]}`,
	}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              11,
		Name:            "iq-main",
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        endpoint,
		APIKey:          "OLD:sk-openai",
		PrimaryModel:    "gpt-4o-mini",
		ExtraModels:     []string{"gpt-extra"},
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeIQ,
		IQFuzzyMatch:    true,
	}}
	svc := newQuotaModeService(repo)

	results, err := svc.RunCheck(context.Background(), 11)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, MonitorStatusOperational, results[0].Status)
	require.Equal(t, MonitorIQStatusOK, results[0].IqStatus)
	require.Equal(t, MonitorIQStatusOK, results[1].IqStatus)

	require.NotContains(t, fmtAny(h.lastBody["messages"]), "Calculate and respond")
	require.Contains(t, fmtAny(h.lastBody["messages"]), "苹果味")
	require.EqualValues(t, monitorIQMaxTokens, h.lastBody["max_tokens"])

	require.Len(t, repo.history, 2)
	require.Equal(t, MonitorIQStatusOK, repo.history[0].IqStatus)
}

func TestRunCheck_IQWrongAnswerStillOperational(t *testing.T) {
	h := &openAICaptureHandler{
		rawResponse: `{"choices":[{"message":{"content":"我猜 20 颗"}}]}`,
	}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              12,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        endpoint,
		APIKey:          "OLD:sk-openai",
		PrimaryModel:    "gpt-4o-mini",
		Enabled:         true,
		IntervalSeconds: 90,
		CheckMode:       MonitorCheckModeIQ,
		IQFuzzyMatch:    true,
	}}
	svc := newQuotaModeService(repo)

	results, err := svc.RunCheck(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, MonitorStatusOperational, results[0].Status)
	require.Equal(t, MonitorIQStatusDown, results[0].IqStatus)
	require.Equal(t, MonitorIQStatusDown, repo.history[0].IqStatus)
}

func TestRunCheck_IQHTTPErrorIsTestError(t *testing.T) {
	h := &openAICaptureHandler{status: 503, rawResponse: `{"error":"busy"}`}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              13,
		Provider:        MonitorProviderOpenAI,
		APIMode:         MonitorAPIModeChatCompletions,
		Endpoint:        endpoint,
		APIKey:          "OLD:sk-openai",
		PrimaryModel:    "gpt-4o-mini",
		Enabled:         true,
		IntervalSeconds: 60,
		CheckMode:       MonitorCheckModeIQ,
		IQFuzzyMatch:    true,
	}}
	svc := newQuotaModeService(repo)

	results, err := svc.RunCheck(context.Background(), 13)
	require.NoError(t, err)
	require.Equal(t, MonitorStatusFailed, results[0].Status)
	require.Equal(t, MonitorIQStatusTestErr, results[0].IqStatus)
}

func fmtAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
