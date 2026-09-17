//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCheck_IQSendsArithmeticThenCandyQuiz(t *testing.T) {
	h := &openAICaptureHandler{}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              11,
		Name:            "iq-main",
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

	results, err := svc.RunCheck(context.Background(), 11)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, MonitorStatusOperational, results[0].Status)
	require.Equal(t, MonitorIQStatusOK, results[0].IqStatus)

	require.Len(t, h.bodies, 2)
	require.Contains(t, fmtAny(h.bodies[0]["messages"]), "Calculate and respond")
	require.NotContains(t, fmtAny(h.bodies[0]["messages"]), "苹果味")
	require.EqualValues(t, monitorChallengeMaxTokens, h.bodies[0]["max_tokens"])

	require.Contains(t, fmtAny(h.bodies[1]["messages"]), "苹果味")
	require.NotContains(t, fmtAny(h.bodies[1]["messages"]), "Calculate and respond")
	require.EqualValues(t, monitorIQMaxTokens, h.bodies[1]["max_tokens"])

	require.Len(t, repo.history, 1)
	require.Equal(t, MonitorIQStatusOK, repo.history[0].IqStatus)
}

func TestRunCheck_IQExtraModelsAlsoSendTwoRequests(t *testing.T) {
	h := &openAICaptureHandler{}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              14,
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

	results, err := svc.RunCheck(context.Background(), 14)
	require.NoError(t, err)
	require.Len(t, results, 2)

	probe, iq := countCapturedPromptKinds(h)
	require.Equal(t, 2, probe)
	require.Equal(t, 2, iq)
}

func TestRunCheck_IQWrongAnswerStillOperational(t *testing.T) {
	h := &openAICaptureHandler{
		iqRawResponse: `{"choices":[{"message":{"content":"我猜 20 颗"}}]}`,
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

func TestRunCheck_IQHTTPErrorDoesNotFailServerProbe(t *testing.T) {
	h := &openAICaptureHandler{iqStatus: 503, iqRawResponse: `{"error":"busy"}`}
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
	require.Equal(t, MonitorStatusOperational, results[0].Status)
	require.Equal(t, MonitorIQStatusTestErr, results[0].IqStatus)
}

func TestRunCheck_IQProbeHTTPErrorKeepsIQResult(t *testing.T) {
	h := &openAICaptureHandler{status: 400, rawResponse: `{"error":"bad probe"}`}
	endpoint := setupFakeOpenAI(t, h)
	repo := &quotaModeRepoStub{monitor: &ChannelMonitor{
		ID:              15,
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

	results, err := svc.RunCheck(context.Background(), 15)
	require.NoError(t, err)
	require.Equal(t, MonitorStatusError, results[0].Status)
	require.Equal(t, MonitorIQStatusOK, results[0].IqStatus)
}

func countCapturedPromptKinds(h *openAICaptureHandler) (probe, iq int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, body := range h.bodies {
		text := fmtAny(body["messages"])
		if strings.Contains(text, "Calculate and respond") {
			probe++
		}
		if strings.Contains(text, "苹果味") {
			iq++
		}
	}
	return probe, iq
}

func fmtAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
