//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestExtractResponsesReasoningEffortFromBody(t *testing.T) {
	t.Parallel()

	got := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"claude-sonnet-4.5","reasoning":{"effort":"HIGH"}}`))
	require.NotNil(t, got)
	require.Equal(t, "high", *got)

	require.Nil(t, ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"claude-sonnet-4.5"}`)))

	maxGot := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"deepseek-v4-pro","reasoning":{"effort":"max"}}`))
	require.NotNil(t, maxGot)
	require.Equal(t, "max", *maxGot)

	mappedMax := ExtractResponsesReasoningEffortFromBody(
		[]byte(`{"model":"public-alias","reasoning":{"effort":"max"}}`),
		"provider/glm-5.2",
		"public-alias",
	)
	require.NotNil(t, mappedMax)
	require.Equal(t, "max", *mappedMax)

	legacyMax := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"gpt-5.5","reasoning":{"effort":"max"}}`))
	require.NotNil(t, legacyMax)
	require.Equal(t, "xhigh", *legacyMax)
}

func TestHandleResponsesBufferedStreamingResponse_PreservesMessageStartCacheUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_buffered"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":12,"cache_read_input_tokens":9,"cache_creation_input_tokens":3}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hello"}}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleResponsesBufferedStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 9, result.Usage.CacheReadInputTokens)
	require.Equal(t, 3, result.Usage.CacheCreationInputTokens)
	require.Contains(t, rec.Body.String(), `"cached_tokens":9`)
}

func TestHandleResponsesStreamingResponse_PreservesMessageStartCacheUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":20,"cache_read_input_tokens":11,"cache_creation_input_tokens":4}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hello"}}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":8}}`,
			``,
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleResponsesStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 20, result.Usage.InputTokens)
	require.Equal(t, 8, result.Usage.OutputTokens)
	require.Equal(t, 11, result.Usage.CacheReadInputTokens)
	require.Equal(t, 4, result.Usage.CacheCreationInputTokens)
	require.Contains(t, rec.Body.String(), `response.completed`)
}

func TestOpus55ResponsesSignedThinkingBufferedAndStreamed(t *testing.T) {
	payload := strings.Join([]string{
		"event: message_start\n" + `data: {"type":"message_start","message":{"id":"msg_1","model":"claude-opus-5-5","content":[],"usage":{"input_tokens":10}}}`,
		"event: content_block_start\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		"event: content_block_delta\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"signed"}}`,
		"event: content_block_stop\n" + `data: {"type":"content_block_stop","index":0}`,
		"event: content_block_start\n" + `data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		"event: content_block_delta\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"ok"}}`,
		"event: content_block_stop\n" + `data: {"type":"content_block_stop","index":1}`,
		"event: message_delta\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
		"event: message_stop\n" + `data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"
	for _, stream := range []bool{false, true} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}
		svc := &GatewayService{}
		var err error
		if stream {
			_, err = svc.handleResponsesStreamingResponse(resp, c, "public-opus", "claude-opus-5-5", nil, time.Now())
		} else {
			_, err = svc.handleResponsesBufferedStreamingResponse(resp, c, "public-opus", "claude-opus-5-5", nil, time.Now())
		}
		require.NoError(t, err)
		require.Contains(t, rec.Body.String(), "anthropic-thinking-v1:")
		require.Contains(t, rec.Body.String(), "public-opus")
		require.Contains(t, rec.Body.String(), `"text":"ok"`)
	}
}

func TestOpus55BridgeUsesMappedModelBeforeThinkingConversion(t *testing.T) {
	for _, chat := range []bool{false, true} {
		for _, forced := range []bool{false, true} {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			body := `{"model":"public-opus","input":"hello","reasoning":{"effort":"xhigh"}}`
			if chat {
				body = `{"model":"public-opus","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"xhigh"}`
			}
			if forced {
				body = body[:len(body)-1] + `,"tool_choice":"required","tools":[{"type":"function","name":"lookup","function":{"name":"lookup"}}]}`
			}
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
			upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`event: message_stop` + "\n\n"))}}
			svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, tlsFPProfileService: &TLSFingerprintProfileService{}}
			account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "fixture-key", "model_mapping": map[string]any{"public-opus": "claude-opus-5-5"}}}
			var err error
			if chat {
				_, err = svc.ForwardAsChatCompletions(context.Background(), c, account, []byte(body), nil)
			} else {
				_, err = svc.ForwardAsResponses(context.Background(), c, account, []byte(body), nil)
			}
			if forced {
				require.Error(t, err)
				require.Equal(t, 400, rec.Code)
				require.Nil(t, upstream.lastReq)
				continue
			}
			require.NoError(t, err)
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, "claude-opus-5-5", gjson.GetBytes(upstream.lastBody, "model").String())
			require.Equal(t, "adaptive", gjson.GetBytes(upstream.lastBody, "thinking.type").String())
			require.Equal(t, "xhigh", gjson.GetBytes(upstream.lastBody, "output_config.effort").String())
			require.False(t, gjson.GetBytes(upstream.lastBody, "thinking.budget_tokens").Exists())
		}
	}
}
