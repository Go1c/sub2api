//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGinContextForEndpoint(t *testing.T, endpoint string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, endpoint, nil)
	return c, recorder
}

func parseResponsesFailedSSE(t *testing.T, body string) map[string]any {
	t.Helper()
	require.True(t, strings.HasPrefix(body, "event: response.failed\n"))
	lines := strings.SplitN(strings.TrimSuffix(body, "\n\n"), "\n", 2)
	require.Len(t, lines, 2)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &parsed))
	return parsed
}

func TestOpenAIHandleStreamingAwareError_ResponsesStreamingEmitsResponseFailed(t *testing.T) {
	c, recorder := newGinContextForEndpoint(t, "/v1/responses")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.RequestID, "fd277bc5-ff7e-45d1-8aa9-f54e1df318f1"))
	setOpsRequestContext(c, "gpt-5.6", true, nil)

	(&OpenAIGatewayHandler{}).handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "retry later", true)

	parsed := parseResponsesFailedSSE(t, recorder.Body.String())
	require.Equal(t, "response.failed", parsed["type"])
	response := parsed["response"].(map[string]any)
	require.Equal(t, "resp_fd277bc5ff7e45d18aa9f54e1df318f1", response["id"])
	require.Equal(t, "gpt-5.6", response["model"])
	require.Equal(t, []any{}, response["output"])
	createdAt, ok := response["created_at"].(float64)
	require.True(t, ok, "created_at must be a number, got %T", response["created_at"])
	require.Greater(t, int64(createdAt), int64(0))
	errorObject := response["error"].(map[string]any)
	require.Equal(t, "rate_limit_exceeded", errorObject["code"])
	require.Equal(t, "retry later", errorObject["message"])
}

func TestOpenAIHandleStreamingAwareError_ChatCompletionsKeepsLegacyEvent(t *testing.T) {
	c, recorder := newGinContextForEndpoint(t, "/v1/chat/completions")
	(&OpenAIGatewayHandler{}).handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "boom", true)
	require.True(t, strings.HasPrefix(recorder.Body.String(), "event: error\n"))
}

func TestInboundIsResponses_CoversKnownRoutes(t *testing.T) {
	tests := map[string]bool{
		"/v1/responses":                        true,
		"/v1/responses/compact":                true,
		"/responses":                           true,
		"/backend-api/codex/responses":         true,
		"/backend-api/codex/responses/compact": true,
		"/v1/chat/completions":                 false,
		"/responses-fake":                      false,
	}
	for route, want := range tests {
		t.Run(route, func(t *testing.T) {
			c, _ := newGinContextForEndpoint(t, route)
			require.Equal(t, want, inboundIsResponses(c))
		})
	}
}

func TestMapResponsesErrorCode(t *testing.T) {
	require.Equal(t, "permission_denied", mapResponsesErrorCode("permission_error"))
	require.Equal(t, "server_error", mapResponsesErrorCode("api_error"))
	require.Equal(t, "custom", mapResponsesErrorCode("custom"))
}
