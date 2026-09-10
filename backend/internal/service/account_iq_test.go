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
	require.Equal(t, "low", gjson.GetBytes(body, "reasoning.effort").String())
	require.Contains(t, gjson.GetBytes(body, "instructions").String(), "complete <svg")
	require.Contains(t, gjson.GetBytes(body, "instructions").String(), "filename")
	require.Equal(t, "reasoning.encrypted_content", gjson.GetBytes(body, "include.0").String())
}

func TestApplyOpenAIAccountIQTestPayload_ChatCompletionsIQ(t *testing.T) {
	payload := createOpenAIChatCompletionsTestPayload("gpt-6-astra", "hello")
	ApplyOpenAIAccountIQTestPayload(payload, "hello", "IQ", false)

	body := mustIQPayloadJSON(t, payload)
	require.Equal(t, "system", gjson.GetBytes(body, "messages.0.role").String())
	require.Contains(t, gjson.GetBytes(body, "messages.0.content").String(), "complete <svg")
	require.Equal(t, "hello", gjson.GetBytes(body, "messages.1.content").String())
	require.Equal(t, "low", gjson.GetBytes(body, "reasoning_effort").String())
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
	require.Equal(t, "low", gjson.GetBytes(body, "reasoning.effort").String())
	require.Contains(t, gjson.GetBytes(body, "instructions").String(), "complete <svg")
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
	require.Equal(t, "low", gjson.GetBytes(body, "reasoning.effort").String())
	require.Contains(t, gjson.GetBytes(body, "instructions").String(), "complete <svg")
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
	require.Equal(t, "Generate an SVG of a pelican riding a bicycle", gjson.GetBytes(upstream.lastBody, "messages.1.content").String())
	require.Equal(t, "system", gjson.GetBytes(upstream.lastBody, "messages.0.role").String())
	require.Equal(t, "low", gjson.GetBytes(upstream.lastBody, "reasoning_effort").String())
}

func TestAccountTestService_OpenAIResponsesIQRetriesCapacityShedThenSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	shed := newOpenAISSEResponse(
		`{"type":"response.output_text.delta","delta":"I'll create a standalone SVG illustration"}`,
		`{"type":"response.failed","response":{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}}`,
	)
	ok := newOpenAISSEResponse(
		`{"type":"response.output_text.delta","delta":"<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"}`,
		`{"type":"response.completed"}`,
	)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{shed, ok}}
	svc := &AccountTestService{httpUpstream: upstream}
	account := newOpenAIIQOAuthAccount(101)

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)

	sse := recorder.Body.String()
	require.Contains(t, sse, "正在重试")
	require.Contains(t, sse, "svg xmlns")
	require.Contains(t, sse, `"success":true`)
	require.NotContains(t, sse, "I'll create a standalone SVG illustration")
	require.NotContains(t, sse, "Our servers are currently overloaded")
}

func TestAccountTestService_OpenAIResponsesIQRetriesIncompleteStreamThenSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	cut := newOpenAISSEResponse(`{"type":"response.output_text.delta","delta":"thinking about the pelican"}`)
	ok := newOpenAISSEResponse(
		`{"type":"response.output_text.delta","delta":"<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"}`,
		`{"type":"response.completed"}`,
	)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{cut, ok}}
	svc := &AccountTestService{httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, newOpenAIIQOAuthAccount(102), "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)
	require.Contains(t, recorder.Body.String(), "svg xmlns")
	require.NotContains(t, recorder.Body.String(), "thinking about the pelican")
}

func TestAccountTestService_OpenAIResponsesIQRetriesHTTP503ThenSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	shed := newJSONResponse(http.StatusServiceUnavailable, `{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}`)
	ok := newOpenAISSEResponse(
		`{"type":"response.output_text.delta","delta":"<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"}`,
		`{"type":"response.completed"}`,
	)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{shed, ok}}
	svc := &AccountTestService{httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, newOpenAIIQOAuthAccount(103), "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.NotContains(t, recorder.Body.String(), "Our servers are currently overloaded")
}

func TestAccountTestService_OpenAIResponsesIQExhaustedRetriesStops(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	shed := func() *http.Response {
		return newOpenAISSEResponse(
			`{"type":"response.failed","response":{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}}`,
		)
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{shed(), shed(), shed()}}
	svc := &AccountTestService{httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, newOpenAIIQOAuthAccount(104), "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.Error(t, err)
	require.Len(t, upstream.requests, accountIQTestMaxAttempts)
	require.Contains(t, recorder.Body.String(), "上游多次过载")
	require.NotContains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestService_OpenAIResponsesIQDoesNotRetryUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	unauthorized := newJSONResponse(http.StatusUnauthorized, `{"error":{"message":"invalid_api_key"}}`)
	ok := newOpenAISSEResponse(`{"type":"response.completed"}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{unauthorized, ok}}
	svc := &AccountTestService{httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, newOpenAIIQOAuthAccount(105), "gpt-6-astra", "Generate an SVG of a pelican riding a bicycle", "iq")
	require.Error(t, err)
	require.Len(t, upstream.requests, 1)
	require.Contains(t, recorder.Body.String(), "401")
	require.NotContains(t, recorder.Body.String(), "正在重试")
}

func TestAccountTestService_OpenAIResponsesDefaultModeDoesNotRetryCapacityShed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	shed := newOpenAISSEResponse(
		`{"type":"response.failed","response":{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}}`,
	)
	ok := newOpenAISSEResponse(`{"type":"response.completed"}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{shed, ok}}
	svc := &AccountTestService{httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, newOpenAIIQOAuthAccount(106), "gpt-5.4", "hi", "")
	require.Error(t, err)
	require.Len(t, upstream.requests, 1)
	require.Contains(t, recorder.Body.String(), "overloaded")
	require.NotContains(t, recorder.Body.String(), "正在重试")
}

func TestAccountTestService_OpenAIChatCompletionsIQRetriesCapacityShedThenSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	shed := newJSONResponse(http.StatusServiceUnavailable, `{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}`)
	okBody := strings.Join([]string{
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	ok := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(okBody)),
	}
	upstream := &httpUpstreamRecorder{responses: []*http.Response{shed, ok}}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	account := &Account{
		ID:          107,
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
	require.Len(t, upstream.requests, 2)
	require.Contains(t, recorder.Body.String(), "svg xmlns")
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.NotContains(t, recorder.Body.String(), "Our servers are currently overloaded")
}

func TestAccountIQTestRetryableHTTP(t *testing.T) {
	require.True(t, accountIQTestRetryableHTTP(http.StatusServiceUnavailable, nil))
	require.True(t, accountIQTestRetryableHTTP(http.StatusTooManyRequests, nil))
	require.True(t, accountIQTestRetryableHTTP(http.StatusOK, []byte(`{"error":{"message":"Our servers are currently overloaded."}}`)))
	require.False(t, accountIQTestRetryableHTTP(http.StatusUnauthorized, []byte(`{"error":{"message":"invalid_api_key"}}`)))
	require.False(t, accountIQTestRetryableHTTP(http.StatusBadRequest, []byte(`{"error":{"message":"invalid_request"}}`)))
}

func newOpenAIIQOAuthAccount(id int64) *Account {
	return &Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
}

func newOpenAISSEResponse(events ...string) *http.Response {
	var b strings.Builder
	for _, event := range events {
		b.WriteString("data: ")
		b.WriteString(event)
		b.WriteString("\n\n")
	}
	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(b.String()))
	return resp
}

func mustIQPayloadJSON(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return body
}
