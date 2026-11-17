package service

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSanitizeGrokResponsesModelInputAgentMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		text    string
	}{
		{"text only", `[{"type":"input_text","text":"What is 19 plus 23?"}]`, "What is 19 plus 23?"},
		{"envelope and plaintext payload", `[{"type":"input_text","text":"Message Type: NEW_TASK\nPayload:\n"},{"type":"encrypted_content","encrypted_content":"Review the code"},{"type":"text","text":"\nthen report"},{"type":"encrypted_content","encrypted_content":" back. "}]`, "Message Type: NEW_TASK\nPayload:\nReview the code\nthen report back. "},
		{"string content", `"  task\n"`, "  task\n"},
		{"opaque content is not decrypted", `[{"type":"encrypted_content","encrypted_content":"gAAAAABopaque=="}]`, "gAAAAABopaque=="},
		{"empty content", `[]`, ""},
		{"null content", `null`, ""},
		{"unsupported content", `[{"type":"input_image","image_url":"https://example.com/image.png"},{"type":"output_text","text":"ignored"}]`, ""},
		{"malformed content", `{"text":"ignored"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"input":[{"type":"agent_message","id":"amsg_1","author":"/root","recipient":"/root/child","role":"assistant","content":` + tt.content + `}]}`)
			patched, err := sanitizeGrokResponsesModelInput(body)
			require.NoError(t, err)
			if tt.text == "" {
				require.Empty(t, gjson.GetBytes(patched, "input").Array())
				return
			}
			expected, err := json.Marshal([]map[string]any{{"type": "message", "role": "user", "content": tt.text}})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), gjson.GetBytes(patched, "input").Raw)
		})
	}
}

func TestSanitizeGrokResponsesModelInputAgentMessageMixedHistory(t *testing.T) {
	body := []byte(`{"input":[
		{"type":"message","role":"user","content":"start"},
		{"type":"agent_message","author":"/root","recipient":"/root/child","content":[{"type":"input_text","text":"Payload:\n"},{"type":"encrypted_content","encrypted_content":"task"}]},
		{"type":"message","role":"assistant","content":"working"},
		{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{}"},
		{"type":"function_call_output","call_id":"call_1","output":"found"},
		{"type":"agent_message","content":[]},
		{"type":"agent_message","author":"/root/child","recipient":"/root","content":[{"type":"input_text","text":"result"}]},
		{"type":"message","role":"user","content":"continue"}
	]}`)
	patched, err := sanitizeGrokResponsesModelInput(body)
	require.NoError(t, err)
	require.JSONEq(t, `[
		{"type":"message","role":"user","content":"start"},
		{"type":"message","role":"user","content":"Payload:\ntask"},
		{"type":"message","role":"assistant","content":"working"},
		{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{}"},
		{"type":"function_call_output","call_id":"call_1","output":"found"},
		{"type":"message","role":"user","content":"result"},
		{"type":"message","role":"user","content":"continue"}
	]`, gjson.GetBytes(patched, "input").Raw)
}

func TestSanitizeGrokResponsesModelInputNormalizesReplayHistory(t *testing.T) {
	body := []byte(`{
		"input":[
			{"role":"system","content":"rules"},
			{"type":"message","role":"assistant","id":"msg_partial","content":[{"type":"output_text","text":"done"}]},
			{"type":"reasoning","id":"rs_1","status":"completed","content":null,"summary":[{"type":"summary_text","text":"keep"}]},
			{"type":"message","role":"assistant","id":"msg_complete","status":"completed","content":[{"type":"output_text","text":"kept"}]}
		]
	}`)

	patched, err := sanitizeGrokResponsesModelInput(body)
	require.NoError(t, err)
	require.Equal(t, "message", gjson.GetBytes(patched, "input.0.type").String())
	require.Equal(t, "done", gjson.GetBytes(patched, "input.1.content").String())
	require.False(t, gjson.GetBytes(patched, "input.1.id").Exists())
	require.False(t, gjson.GetBytes(patched, "input.2.status").Exists())
	require.False(t, gjson.GetBytes(patched, "input.2.content").Exists())
	require.Equal(t, "keep", gjson.GetBytes(patched, "input.2.summary.0.text").String())
	require.Equal(t, "msg_complete", gjson.GetBytes(patched, "input.3.id").String())
	require.Equal(t, "completed", gjson.GetBytes(patched, "input.3.status").String())
}

func TestSanitizeGrokResponsesModelInputStripsOnlyNonPairCallIDs(t *testing.T) {
	body := []byte(`{"input":[
		{"type":"message","role":"user","call_id":"remove_message","content":"continue"},
		{"type":"reasoning","call_id":"remove_reasoning","summary":[]},
		{"type":"image_generation_call","call_id":"remove_image","id":"ig_1","status":"completed"},
		{"type":"function_call","call_id":"keep_function","name":"lookup","arguments":"{}"},
		{"type":"function_call_output","call_id":"keep_function","output":"ok"}
	]}`)

	patched, err := sanitizeGrokResponsesModelInput(body)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		require.False(t, gjson.GetBytes(patched, "input."+strconv.Itoa(i)+".call_id").Exists())
	}
	require.Equal(t, "keep_function", gjson.GetBytes(patched, "input.3.call_id").String())
	require.Equal(t, "keep_function", gjson.GetBytes(patched, "input.4.call_id").String())
}

func TestSanitizeGrokResponsesModelInputPairsCallsInTwoPasses(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"function_call_output","output":{"before":true,"large_id":9007199254740993}},
			{"type":"function_call","function":{"name":"lookup","arguments":{"query":"x","large_id":9007199254740993}}},
			{"type":"custom_tool_call","id":"custom_1","name":"apply_patch","input":"patch"},
			{"type":"custom_tool_call_output","tool_call_id":"custom_1","output":["ok"]},
			{"type":"tool_search_call","id":"search_1","arguments":{"query":"repo"}},
			{"type":"tool_search_output","call_id":"search_1","results":{"groups":["repo"]}},
			{"role":"tool","tool_call_id":"orphan","content":{"value":1}}
		]
	}`)

	patched, err := sanitizeGrokResponsesModelInput(body)
	require.NoError(t, err)
	generated := gjson.GetBytes(patched, "input.1.call_id").String()
	require.NotEmpty(t, generated)
	require.Equal(t, generated, gjson.GetBytes(patched, "input.0.call_id").String())
	require.JSONEq(t, `{"before":true,"large_id":9007199254740993}`, gjson.GetBytes(patched, "input.0.output").String())
	require.Equal(t, "9007199254740993", gjson.Get(gjson.GetBytes(patched, "input.0.output").String(), "large_id").Raw)
	require.Equal(t, "lookup", gjson.GetBytes(patched, "input.1.name").String())
	require.JSONEq(t, `{"query":"x","large_id":9007199254740993}`, gjson.GetBytes(patched, "input.1.arguments").String())
	require.Equal(t, "9007199254740993", gjson.Get(gjson.GetBytes(patched, "input.1.arguments").String(), "large_id").Raw)
	customCallID := gjson.GetBytes(patched, "input.2.call_id").String()
	require.NotEmpty(t, customCallID)
	require.NotEqual(t, "custom_1", customCallID)
	require.JSONEq(t, `{"input":"patch"}`, gjson.GetBytes(patched, "input.2.arguments").String())
	require.Equal(t, customCallID, gjson.GetBytes(patched, "input.3.call_id").String())
	require.JSONEq(t, `["ok"]`, gjson.GetBytes(patched, "input.3.output").String())
	require.Equal(t, "tool_search", gjson.GetBytes(patched, "input.4.name").String())
	searchCallID := gjson.GetBytes(patched, "input.4.call_id").String()
	require.NotEmpty(t, searchCallID)
	require.NotEqual(t, "search_1", searchCallID)
	require.Equal(t, searchCallID, gjson.GetBytes(patched, "input.5.call_id").String())
	require.JSONEq(t, `{"groups":["repo"]}`, gjson.GetBytes(patched, "input.5.output").String())
	require.Equal(t, "function_call_output", gjson.GetBytes(patched, "input.6.type").String())
	require.Equal(t, "orphan", gjson.GetBytes(patched, "input.6.call_id").String())
}

func TestSanitizeGrokResponsesModelInputMapsAllItemAliasesAndRejectsConflicts(t *testing.T) {
	body := []byte(`{"input":[
		{"type":"function_call","id":"fc_alias","call_id":"call_canonical","name":"one","arguments":"{}"},
		{"type":"function_call_output","id":"fc_alias","output":"ok"},
		{"type":"function_call","id":"duplicate_item","name":"two","arguments":"{}"},
		{"type":"function_call","id":"duplicate_item","name":"three","arguments":"{}"},
		{"type":"function_call_output","id":"duplicate_item","output":"ambiguous"}
	]}`)

	patched, err := sanitizeGrokResponsesModelInput(body)
	require.NoError(t, err)
	require.Equal(t, "call_canonical", gjson.GetBytes(patched, "input.0.call_id").String())
	require.Equal(t, "call_canonical", gjson.GetBytes(patched, "input.1.call_id").String())
	require.False(t, gjson.GetBytes(patched, "input.0.id").Exists())
	secondCallID := gjson.GetBytes(patched, "input.2.call_id").String()
	thirdCallID := gjson.GetBytes(patched, "input.3.call_id").String()
	ambiguousOutputID := gjson.GetBytes(patched, "input.4.call_id").String()
	require.NotEmpty(t, secondCallID)
	require.NotEmpty(t, thirdCallID)
	require.NotEmpty(t, ambiguousOutputID)
	require.NotEqual(t, "duplicate_item", secondCallID)
	require.NotEqual(t, secondCallID, thirdCallID)
	require.NotEqual(t, secondCallID, ambiguousOutputID)
	require.NotEqual(t, thirdCallID, ambiguousOutputID)
}
