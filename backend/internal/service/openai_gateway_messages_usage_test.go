//go:build unit

package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCopyOpenAIUsageFromResponsesUsageTrustsCanonicalCacheCreationValue(t *testing.T) {
	usage := &apicompat.ResponsesUsage{
		InputTokens:              20,
		OutputTokens:             2,
		CacheCreationInputTokens: 0,
		InputTokensDetails: &apicompat.ResponsesInputTokensDetails{
			CachedTokens:     3,
			CacheWriteTokens: 19,
		},
	}

	got := copyOpenAIUsageFromResponsesUsage(usage)

	require.Equal(t, 20, got.InputTokens)
	require.Equal(t, 3, got.CacheReadInputTokens)
	require.Zero(t, got.CacheCreationInputTokens)
}

func TestCopyOpenAIUsageFromResponsesUsageCopiesImageTokens(t *testing.T) {
	var usage apicompat.ResponsesUsage
	require.NoError(t, json.Unmarshal([]byte(`{
		"input_tokens":371,
		"output_tokens":439,
		"input_tokens_details":{"image_tokens":352},
		"output_tokens_details":{"image_tokens":439}
	}`), &usage))

	got := copyOpenAIUsageFromResponsesUsage(&usage)

	require.Equal(t, 352, got.ImageInputTokens)
	require.Equal(t, 439, got.ImageOutputTokens)
}

func TestReadOpenAICompatBufferedTerminalParsesWrappedUsage(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`event: response.completed
data: {"response":{"id":"resp_buffered","object":"response","model":"gpt-5.6-sol","status":"completed","output":[]},"data":{"response":{"usage":{"prompt_tokens":21,"completion_tokens":9}}}}

`)),
	}

	svc := &OpenAIGatewayService{}
	finalResponse, usage, _, err := svc.readOpenAICompatBufferedTerminal(resp, "test buffered", "rid_buffered")

	require.NoError(t, err)
	require.NotNil(t, finalResponse)
	require.Equal(t, "resp_buffered", finalResponse.ID)
	require.Equal(t, 21, usage.InputTokens)
	require.Equal(t, 9, usage.OutputTokens)
}

func TestHandleAnthropicStreamingResponseParsesWrappedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	resp := &http.Response{
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"x-request-id": []string{"rid_messages_usage"},
		},
		Body: io.NopCloser(strings.NewReader(`event: response.completed
data: {"response":{"id":"resp_messages","object":"response","model":"gpt-5.6-sol","status":"completed","output":[]},"data":{"usage":{"input_tokens":22,"output_tokens":10}}}

`)),
	}

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	result, err := svc.handleAnthropicStreamingResponse(
		resp,
		c,
		&Account{ID: 113, Name: "openai-compatible", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		"claude-compatible",
		"gpt-5.6-sol",
		"gpt-5.6-sol",
		time.Now(),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 22, result.Usage.InputTokens)
	require.Equal(t, 10, result.Usage.OutputTokens)
}
