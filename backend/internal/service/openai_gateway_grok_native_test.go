//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDoGrokNativeResponsesJSONPatchesEmptyModel(t *testing.T) {
	t.Parallel()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[]}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:       7,
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "xai-test",
			"base_url": "https://api.x.ai",
		},
	}

	body, err := svc.DoGrokNativeResponsesJSON(context.Background(), nil, account, []byte(`{"input":"hi","tools":[{"type":"x_search"}]}`))
	require.NoError(t, err)
	require.Equal(t, "resp_1", gjson.GetBytes(body, "id").String())
	require.Equal(t, grokDefaultResponsesModel, gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "Bearer xai-test", upstream.lastReq.Header.Get("Authorization"))
}

func TestDoGrokNativeResponsesJSONStripsRejectedSearchInclude(t *testing.T) {
	t.Parallel()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_2","output":[]}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:       9,
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "xai-test",
			"base_url": "https://api.x.ai",
		},
	}

	_, err := svc.DoGrokNativeResponsesJSON(context.Background(), nil, account, []byte(`{"model":"grok-4.6","input":"hi","tools":[{"type":"x_search"}],"include":["x_search_call.action.sources"]}`))
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(upstream.lastBody, "include").Exists())
}

func TestDoGrokNativeResponsesJSONFailoversOnUnauthorized(t *testing.T) {
	t.Parallel()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":"expired"}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:       8,
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "xai-test",
			"base_url": "https://api.x.ai",
		},
	}

	_, err := svc.DoGrokNativeResponsesJSON(context.Background(), nil, account, []byte(`{"model":"grok-4.5","input":"hi"}`))
	require.Error(t, err)
	var failover *UpstreamFailoverError
	require.ErrorAs(t, err, &failover)
	require.Equal(t, http.StatusUnauthorized, failover.StatusCode)
}
