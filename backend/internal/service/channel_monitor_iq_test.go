//go:build unit

package service

import (
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorIQAnswerMatches_FuzzyContains21(t *testing.T) {
	require.True(t, monitorIQAnswerMatches("答案是 21。", MonitorIQDefaultAnswer, true))
	require.True(t, monitorIQAnswerMatches("21", "21", true))
	require.True(t, monitorIQAnswerMatches("最少 21 颗", "21", true))
	require.False(t, monitorIQAnswerMatches("20", MonitorIQDefaultAnswer, true))
	require.False(t, monitorIQAnswerMatches("", "21", true))
}

func TestMonitorIQAnswerMatches_ExactIgnoresPunctuation(t *testing.T) {
	require.True(t, monitorIQAnswerMatches("21。", "21", false))
	require.False(t, monitorIQAnswerMatches("答案是21", "21", false))
}

func TestIsMonitorNetworkError_DNS(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "missing.example", IsNotFound: true}
	require.True(t, isMonitorNetworkError(dnsErr))
	require.True(t, isMonitorNetworkError(&url.Error{Op: "Post", URL: "https://missing.example", Err: dnsErr}))
	require.True(t, isMonitorNetworkError(errors.New("do request: Post \"https://x\": no such host")))
	require.False(t, isMonitorNetworkError(errors.New("context deadline exceeded")))
	require.False(t, isMonitorNetworkError(nil))
}

func TestClassifyIQOutcome_FourStates(t *testing.T) {
	iq, msg := classifyIQOutcome(iqClassifyInput{
		statusCode: 200,
		respText:   "最少取 21 颗",
		expected:   "21",
		fuzzy:      true,
	})
	require.Equal(t, MonitorIQStatusOK, iq)
	require.Empty(t, msg)

	iq, _ = classifyIQOutcome(iqClassifyInput{
		statusCode: 200,
		respText:   "我猜是 20",
		expected:   "21",
		fuzzy:      true,
	})
	require.Equal(t, MonitorIQStatusDown, iq)

	iq, _ = classifyIQOutcome(iqClassifyInput{
		statusCode: 500,
		rawBody:    `{"error":"boom"}`,
		expected:   "21",
		fuzzy:      true,
	})
	require.Equal(t, MonitorIQStatusTestErr, iq)

	iq, _ = classifyIQOutcome(iqClassifyInput{
		statusCode: 200,
		respText:   "   ",
		expected:   "21",
		fuzzy:      true,
	})
	require.Equal(t, MonitorIQStatusTestErr, iq)

	iq, msg = classifyIQOutcome(iqClassifyInput{
		err:      &net.DNSError{Err: "no such host", Name: "gone.invalid", IsNotFound: true},
		expected: "21",
		fuzzy:    true,
	})
	require.Equal(t, MonitorIQStatusNetwork, iq)
	require.Equal(t, "monitor-side DNS lookup failed", msg)

	iq, _ = classifyIQOutcome(iqClassifyInput{
		err:      errors.New("context deadline exceeded"),
		expected: "21",
		fuzzy:    true,
	})
	require.Equal(t, MonitorIQStatusTestErr, iq)
}

func TestValidateCreateParams_IQKeepsOriginalInterval(t *testing.T) {
	err := validateCreateParams(ChannelMonitorCreateParams{
		Name:            "iq",
		Provider:        MonitorProviderAnthropic,
		CheckMode:       MonitorCheckModeIQ,
		Endpoint:        "https://api.anthropic.com",
		APIKey:          "sk-ant-test",
		PrimaryModel:    "claude-sonnet-4-5",
		IntervalSeconds: 15,
	})
	require.NoError(t, err)

	err = validateCreateParams(ChannelMonitorCreateParams{
		Name:            "iq",
		Provider:        MonitorProviderAnthropic,
		CheckMode:       MonitorCheckModeIQ,
		Endpoint:        "https://api.anthropic.com",
		APIKey:          "sk-ant-test",
		PrimaryModel:    "claude-sonnet-4-5",
		IntervalSeconds: 14,
	})
	require.ErrorIs(t, err, ErrChannelMonitorInvalidInterval)
}

func TestValidateCreateParams_IQDoesNotRequireAccount(t *testing.T) {
	err := validateCreateParams(ChannelMonitorCreateParams{
		Provider:        MonitorProviderOpenAI,
		CheckMode:       MonitorCheckModeIQ,
		Endpoint:        "https://api.openai.com",
		APIKey:          "sk-test",
		PrimaryModel:    "gpt-4o-mini",
		IntervalSeconds: 60,
	})
	require.NoError(t, err)
}

func TestApplyIQDefaults_FillsCandyQuiz(t *testing.T) {
	m := &ChannelMonitor{CheckMode: MonitorCheckModeIQ}
	applyIQDefaults(m)
	require.Equal(t, MonitorIQDefaultQuestion, m.IQQuestion)
	require.Equal(t, MonitorIQDefaultAnswer, m.IQAnswer)

	probe := &ChannelMonitor{CheckMode: MonitorCheckModeProbe}
	applyIQDefaults(probe)
	require.Empty(t, probe.IQQuestion)
}
