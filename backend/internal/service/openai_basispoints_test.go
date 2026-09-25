package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBasisPointsSwitchDefaultsOffAndRoutesAstra(t *testing.T) {
	disabled := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	require.False(t, disabled.BasisPointsEnabled())

	enabled := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{"basispoints": map[string]any{"enabled": true}},
	}
	require.True(t, enabled.BasisPointsEnabled())
	require.False(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: enabled.Extra}).BasisPointsEnabled())

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	body := []byte(`{"model":"gpt-6-sol","input":"hi","service_tier":"priority"}`)
	rewritten, err := (&OpenAIGatewayService{}).rewriteBasisPointsRequest(c.Request.Context(), c, enabled, body, "token")
	require.NoError(t, err)
	require.Equal(t, string(body), string(rewritten))
	require.False(t, basisPointsRouted(c))

	astra := []byte(`{"model":"gpt-6-astra","stream":true,"instructions":"be brief","service_tier":"priority","include":["reasoning.encrypted_content"],"text":{"verbosity":"low"},"max_output_tokens":128,"reasoning":{"effort":"max"},"prompt_cache_key":"thread-1","metadata":{"client":"codex","task_id":"drop-me"},"context_management":[{"type":"compaction","compact_threshold":475000}],"tools":[{"type":"function","name":"exec_command","description":"run","parameters":{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}}],"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"pwd"}]},{"type":"compaction_trigger"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	rewritten, err = (&OpenAIGatewayService{}).rewriteBasisPointsRequest(c.Request.Context(), c, enabled, astra, "token")
	require.NoError(t, err)
	require.True(t, basisPointsRouted(c))
	require.Equal(t, "gpt-6-astra", gjson.GetBytes(rewritten, "model").String())
	require.Equal(t, "explicit", gjson.GetBytes(rewritten, "model_selection").String())
	require.False(t, gjson.GetBytes(rewritten, "store").Bool())
	require.False(t, gjson.GetBytes(rewritten, "service_tier").Exists())
	require.Equal(t, "run_officejs", gjson.GetBytes(rewritten, "tools.0.name").String())
	require.Equal(t, "thread-1", gjson.GetBytes(rewritten, "prompt_cache_key").String())
	require.Equal(t, "xhigh", gjson.GetBytes(rewritten, "reasoning_effort").String())
	require.Equal(t, "codex", gjson.GetBytes(rewritten, "metadata.client").String())
	require.NotEqual(t, "drop-me", gjson.GetBytes(rewritten, "metadata.task_id").String())
	require.Equal(t, "compaction", gjson.GetBytes(rewritten, "context_management.0.type").String())
	require.False(t, gjson.GetBytes(rewritten, "instructions").Exists())
	require.False(t, gjson.GetBytes(rewritten, "include").Exists())
	require.False(t, gjson.GetBytes(rewritten, "text").Exists())
	require.False(t, gjson.GetBytes(rewritten, "max_output_tokens").Exists())
	require.Equal(t, "1", gjson.GetBytes(rewritten, "metadata.agent_iteration").String())
	require.NotEmpty(t, gjson.GetBytes(rewritten, "metadata.turn_id").String())
	require.Equal(t, "developer", gjson.GetBytes(rewritten, "input.0.role").String())
	require.Contains(t, gjson.GetBytes(rewritten, "input.0.content.0.text").String(), "be brief")
	catalog := gjson.GetBytes(rewritten, "input.1.content.0.text").String()
	require.Contains(t, catalog, "exec_command")
	require.Contains(t, catalog, "run_officejs")
	require.Contains(t, catalog, "cmd (required)")
	require.Contains(t, catalog, `"properties"`)
	require.Equal(t, "compaction_trigger", gjson.GetBytes(rewritten, "input.@reverse.0.type").String())

	req := httptest.NewRequest(http.MethodPost, basisPointsUpstreamURL, strings.NewReader(string(rewritten)))
	enabled.Credentials = map[string]any{"chatgpt_account_id": "acct-1"}
	applyBasisPointsHeaders(req, enabled)
	require.Equal(t, basisPointsUpstreamHost, req.Host)
	require.Equal(t, "acct-1", req.Header.Get("ChatGPT-Account-ID"))
	require.Equal(t, "chatgpt", req.Header.Get("X-Basispoints-Auth-Mode"))
	require.Equal(t, "basispoints-excel-plugin", req.Header.Get("X-OpenAI-Internal-Basispoints-Client-Product"))
	require.Equal(t, "text/event-stream", req.Header.Get("Accept"))
	require.Empty(t, req.Header.Get("originator"))

	// 影子账号自己没有 chatgpt_account_id。resolve 先写入母账号 ID 后，apply 必须留下它。
	shadow := httptest.NewRequest(http.MethodPost, basisPointsUpstreamURL, strings.NewReader(`{"stream":false}`))
	shadow.Header.Set("ChatGPT-Account-ID", "parent-acct")
	applyBasisPointsHeaders(shadow, &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth})
	require.Equal(t, "parent-acct", shadow.Header.Get("ChatGPT-Account-ID"))
	require.Equal(t, "parent-acct", shadow.Header.Get("X-OpenAI-Account-ID"))
	require.Equal(t, "application/json", shadow.Header.Get("Accept"))
}

func TestBasisPointsToolRoundTripKeepsClientName(t *testing.T) {
	source := []byte(`{"model":"gpt-6-astra","tools":[{"type":"function","name":"exec_command","parameters":{"type":"object"}}],"input":[{"type":"function_call","call_id":"call_1","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}"}]}`)
	rewritten, err := prepareBasisPointsBody(source)
	require.NoError(t, err)
	call := gjson.GetBytes(rewritten, "input.1")
	require.Equal(t, "run_officejs", call.Get("name").String())
	code := gjson.Get(call.Get("arguments").String(), "code").String()
	require.Equal(t, "exec_command", gjson.Get(code, "tool").String())
	require.Equal(t, "pwd", gjson.Get(code, "args.cmd").String())

	upstream := []byte(`{"id":"resp_1","model":"gpt-6-astra","output":[{"type":"function_call","id":"fc_1","call_id":"call_2","name":"run_officejs","arguments":"{\"code\":\"{\\\"tool\\\":\\\"exec_command\\\",\\\"args\\\":{\\\"cmd\\\":\\\"pwd\\\"}}\"}"}],"usage":{"input_tokens":3,"output_tokens":4}}`)
	restored, err := unwrapBasisPointsResponseBody(upstream, source)
	require.NoError(t, err)
	require.Equal(t, "exec_command", gjson.GetBytes(restored, "output.0.name").String())
	require.Equal(t, "call_2", gjson.GetBytes(restored, "output.0.call_id").String())
	require.JSONEq(t, `{"cmd":"pwd"}`, gjson.GetBytes(restored, "output.0.arguments").String())

	unknown := []byte(`{"output":[{"type":"function_call","call_id":"call_9","name":"run_officejs","arguments":"{\"code\":\"{\\\"tool\\\":\\\"not_in_catalog\\\",\\\"args\\\":{}}\"}"}]}`)
	_, err = unwrapBasisPointsResponseBody(unknown, source)
	require.Error(t, err)
}

func TestBasisPointsTurnStaysStableAcrossToolOutput(t *testing.T) {
	first := []byte(`{"model":"gpt-6-astra","prompt_cache_key":"thread-9","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"pwd"}]}]}`)
	second := []byte(`{"model":"gpt-6-astra","prompt_cache_key":"thread-9","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"pwd"}]},{"type":"function_call_output","call_id":"call_1","output":"/tmp"}]}`)
	firstBody, err := prepareBasisPointsBody(first)
	require.NoError(t, err)
	secondBody, err := prepareBasisPointsBody(second)
	require.NoError(t, err)
	require.Equal(t, gjson.GetBytes(firstBody, "metadata.turn_id").String(), gjson.GetBytes(secondBody, "metadata.turn_id").String())
	require.Equal(t, "1", gjson.GetBytes(firstBody, "metadata.agent_iteration").String())
	require.Equal(t, "2", gjson.GetBytes(secondBody, "metadata.agent_iteration").String())
	require.Equal(t, "function_call_output", gjson.GetBytes(secondBody, "input.2.type").String())
}

func TestBasisPointsSSEReplaysTextAndRewritesTool(t *testing.T) {
	source := []byte(`{"tools":[{"type":"function","name":"exec_command","parameters":{"type":"object"}}]}`)
	raw := []byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\nevent: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"id\":\"fc_2\",\"call_id\":\"call_2\",\"name\":\"run_officejs\",\"arguments\":\"{\\\"code\\\":\\\"{\\\\\\\"tool\\\\\\\":\\\\\\\"exec_command\\\\\\\",\\\\\\\"args\\\\\\\":{\\\\\\\"cmd\\\\\\\":\\\\\\\"pwd\\\\\\\"}}\\\"}\",\"status\":\"completed\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"type\":\"response.function_call_arguments.delta\",\"name\":\"run_officejs\",\"delta\":\"{\\\"code\\\"\"}\n\nevent: response.function_call_arguments.done\ndata: {\"type\":\"response.function_call_arguments.done\",\"name\":\"run_officejs\",\"arguments\":\"{\\\"code\\\":\\\"{\\\\\\\"tool\\\\\\\":\\\\\\\"exec_command\\\\\\\",\\\\\\\"args\\\\\\\":{\\\\\\\"cmd\\\\\\\":\\\\\\\"pwd\\\\\\\"}}\\\"}\"}\n\nevent: response.output_item.done\ndata: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"function_call\",\"id\":\"fc_2\",\"call_id\":\"call_2\",\"name\":\"run_officejs\",\"arguments\":\"{\\\"code\\\":\\\"{\\\\\\\"tool\\\\\\\":\\\\\\\"exec_command\\\\\\\",\\\\\\\"args\\\\\\\":{\\\\\\\"cmd\\\\\\\":\\\\\\\"pwd\\\\\\\"}}\\\"}\"}}\n\ndata: [DONE]\n\n")
	replay, err := replayBasisPointsSSE(raw, source)
	require.NoError(t, err)
	text := string(replay)
	require.Contains(t, text, "response.output_text.delta")
	require.Contains(t, text, "\"delta\":\"hi\"")
	require.NotContains(t, text, "function_call_arguments.delta")
	require.NotContains(t, text, "run_officejs")
	require.Contains(t, text, "response.output_item.added")
	require.Contains(t, text, `"arguments":""`)
	require.Contains(t, text, "exec_command")
	require.Contains(t, text, `"name":"exec_command"`)
}

func TestBasisPointsReplaysRememberedNativeCall(t *testing.T) {
	native := `{"type":"function_call","id":"fc_native","call_id":"call_native","name":"run_officejs","arguments":"{\"summary\":\"original\",\"code\":\"{\\\"tool\\\":\\\"exec_command\\\",\\\"args\\\":{\\\"cmd\\\":\\\"pwd\\\"}}\",\"destructive\":false,\"references\":[]}","status":"completed"}`
	rememberBasisPointsCall(gjson.Parse(native))
	next := []byte(`{"model":"gpt-6-astra","tools":[{"type":"function","name":"exec_command"}],"input":[{"type":"message","role":"user","content":"pwd"},{"type":"function_call","call_id":"call_native","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}"},{"type":"function_call_output","call_id":"call_native","name":"exec_command","output":"/tmp"},{"type":"reasoning","encrypted_content":"cipher","summary":[{"type":"summary_text","text":"thinking"}]},{"type":"item_reference","id":"ref_1"}]}`)
	rewritten, err := prepareBasisPointsBody(next)
	require.NoError(t, err)
	call := gjson.GetBytes(rewritten, "input.2")
	require.Equal(t, "run_officejs", call.Get("name").String())
	require.Equal(t, "fc_native", call.Get("id").String())
	require.Contains(t, call.Get("arguments").String(), "original")
	output := gjson.GetBytes(rewritten, "input.3")
	require.Equal(t, "function_call_output", output.Get("type").String())
	require.Equal(t, "fc_call_native", output.Get("id").String())
	require.False(t, output.Get("name").Exists())
	require.Equal(t, "/tmp", output.Get("output").String())
	reasoning := gjson.GetBytes(rewritten, "input.4")
	require.Equal(t, "cipher", reasoning.Get("encrypted_content").String())
	require.False(t, reasoning.Get("summary.0.text").Exists())
	require.False(t, strings.Contains(string(rewritten), "item_reference"))
	require.False(t, gjson.GetBytes(rewritten, "include").Exists())
}

func TestBasisPointsImageDataURLBecomesFileID(t *testing.T) {
	media, data, err := decodeBasisPointsDataURL("data:image/png;base64,aGVsbG8=")
	require.NoError(t, err)
	require.Equal(t, "image/png", media)
	require.Equal(t, "hello", string(data))

	_, _, err = decodeBasisPointsDataURL("data:text/plain;base64,aGVsbG8=")
	require.Error(t, err)

	body := []byte(`{"input":[{"role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,aGVsbG8=","file_id":"already"}]}]}`)
	_, err = (&OpenAIGatewayService{}).uploadBasisPointsImages(t.Context(), nil, &Account{}, body, "token")
	require.Error(t, err)
}
