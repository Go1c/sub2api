//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAICapacityShedExhaustionReturnsClientRetryableServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:             http.StatusBadRequest,
		ResponseBody:           []byte(`{"error":{"message":"Our servers are currently overloaded. Please try again later."}}`),
		RetryableOnSameAccount: false,
		RequestScopedTransient: true,
		NextAccountAction:      service.NextAccountStop,
		Reason:                 service.OpenAICapacityShedReason,
		ClientStatusCode:       http.StatusServiceUnavailable,
		ClientMessage:          "Our servers are currently overloaded. Please try again later.",
	}, false)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "server_error", errObj["type"])
	require.Equal(t, service.OpenAICapacityShedRetryableClientCode, errObj["code"])
	require.Equal(t, "Our servers are currently overloaded. Please try again later.", errObj["message"])
}

func TestOpenAICapacityShedAnthropicExhaustionReturnsOverloadedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	(&OpenAIGatewayHandler{}).handleAnthropicFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:             http.StatusServiceUnavailable,
		RetryableOnSameAccount: false,
		RequestScopedTransient: true,
		NextAccountAction:      service.NextAccountStop,
		Reason:                 service.OpenAICapacityShedReason,
		ClientStatusCode:       http.StatusServiceUnavailable,
		ClientMessage:          "Our servers are currently overloaded. Please try again later.",
	}, false)

	require.Equal(t, 529, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "overloaded_error", errObj["type"])
	require.Equal(t, "Our servers are currently overloaded. Please try again later.", errObj["message"])
}
