//go:build unit

package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestApplyOpenAIAccountIQTestPayload_NoopWhenNotIQ(t *testing.T) {
	payload := createOpenAITestPayload("gpt-5.4", true)
	ApplyOpenAIAccountIQTestPayload(payload, "Generate an SVG of a pelican riding a bicycle", "", true)

	body := mustIQPayloadJSON(t, payload)
	require.Equal(t, "hi", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.False(t, gjson.GetBytes(body, "reasoning").Exists())
	require.False(t, gjson.GetBytes(body, "include").Exists())
}

func TestApplyOpenAIAccountIQTestPayload_ResponsesIQ(t *testing.T) {
	payload := createOpenAITestPayload("gpt-6-astra", true)
	ApplyOpenAIAccountIQTestPayload(payload, "Generate an SVG of a pelican riding a bicycle", "iq", true)

	body := mustIQPayloadJSON(t, payload)
	require.Equal(t, "Generate an SVG of a pelican riding a bicycle", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.Equal(t, "high", gjson.GetBytes(body, "reasoning.effort").String())
	require.Equal(t, "reasoning.encrypted_content", gjson.GetBytes(body, "include.0").String())
}

func TestApplyOpenAIAccountIQTestPayload_ChatCompletionsIQ(t *testing.T) {
	payload := createOpenAIChatCompletionsTestPayload("gpt-6-astra", "hello")
	ApplyOpenAIAccountIQTestPayload(payload, "hello", "IQ", false)

	body := mustIQPayloadJSON(t, payload)
	require.Equal(t, "hello", gjson.GetBytes(body, "messages.0.content").String())
	require.Equal(t, "high", gjson.GetBytes(body, "reasoning_effort").String())
	require.False(t, gjson.GetBytes(body, "reasoning").Exists())
	require.False(t, gjson.GetBytes(body, "include").Exists())
}

func TestApplyOpenAIAccountIQTestPayload_ChatCompletionsDefaultKeepsPromptOnly(t *testing.T) {
	payload := createOpenAIChatCompletionsTestPayload("gpt-5.4", "hello")
	ApplyOpenAIAccountIQTestPayload(payload, "hello", "default", false)

	body := mustIQPayloadJSON(t, payload)
	require.Equal(t, "hello", gjson.GetBytes(body, "messages.0.content").String())
	require.False(t, gjson.GetBytes(body, "reasoning_effort").Exists())
}

func TestRememberAccountIQTestModeKeepsFirstValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	rememberAccountIQTestMode(ctx, "iq")
	rememberAccountIQTestMode(ctx, "default")
	require.Equal(t, "iq", accountIQTestModeFromContext(ctx))
}

func TestAccountTestService_OpenAIResponsesIQModeOverridesPromptAndReasoning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          97,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)

	body, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "gpt-6-astra", gjson.GetBytes(body, "model").String())
	require.Equal(t, "Generate an SVG of a pelican riding a bicycle", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.Equal(t, "high", gjson.GetBytes(body, "reasoning.effort").String())
	require.Equal(t, "reasoning.encrypted_content", gjson.GetBytes(body, "include.0").String())
}

func TestAccountTestService_OpenAIResponsesIQModeSurvivesPublicEntryNormalize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	account := &Account{
		ID:          99,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{99: account},
		},
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}

	err := svc.TestAccountConnection(ctx, account.ID, "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)

	body, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "Generate an SVG of a pelican riding a bicycle", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.Equal(t, "high", gjson.GetBytes(body, "reasoning.effort").String())
}

func TestAccountTestService_OpenAIResponsesEmptyPromptStaysHi(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          96,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "   \n  ", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)

	body, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "hi", gjson.GetBytes(body, "input.0.content.0.text").String())
	require.False(t, gjson.GetBytes(body, "reasoning").Exists())
}

func TestAccountTestService_OpenAIChatCompletionsIQModeSetsReasoningEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	upstreamBody := strings.Join([]string{
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	account := &Account{
		ID:          98,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://compat-upstream.example/v1",
		},
		Extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: false},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Equal(t, "Generate an SVG of a pelican riding a bicycle", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
	require.Equal(t, "high", gjson.GetBytes(upstream.lastBody, "reasoning_effort").String())
}

func mustIQPayloadJSON(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return body
}
