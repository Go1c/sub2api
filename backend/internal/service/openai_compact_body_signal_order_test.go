package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeCompactionTriggerInputOrder_MovesAndCollapsesTriggers(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":[{"type":"compaction_trigger"},{"type":"message","role":"user","content":"tail"},{"type":"compaction_trigger"}]}`)
	normalized, changed, err := NormalizeCompactionTriggerInputOrder(body)
	require.NoError(t, err)
	require.True(t, changed)
	items := gjson.GetBytes(normalized, "input").Array()
	require.Len(t, items, 2)
	require.Equal(t, "message", items[0].Get("type").String())
	require.Equal(t, "compaction_trigger", items[1].Get("type").String())
}

func TestNormalizeCompactionTriggerInputOrder_AlreadyFinalPreservesBytes(t *testing.T) {
	body := []byte(`{"input":[{"type":"message"},{"type":"compaction_trigger"}]}`)
	normalized, changed, err := NormalizeCompactionTriggerInputOrder(body)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, string(body), string(normalized))
}

func TestNormalizeCompactionTriggerInputOrder_PreservesHistoryAndLargeNumbers(t *testing.T) {
	body := []byte(`{"input":[{"type":"compaction_trigger"},{"type":"message","id":"msg_1","content":"visible"},{"type":"function_call_output","call_id":"call_1","output":"result","sequence":9007199254740993}]}`)

	normalized, changed, err := NormalizeCompactionTriggerInputOrder(body)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(normalized, "input.0.type").String())
	require.Equal(t, "function_call_output", gjson.GetBytes(normalized, "input.1.type").String())
	require.Equal(t, "9007199254740993", gjson.GetBytes(normalized, "input.1.sequence").Raw)
	require.Equal(t, "compaction_trigger", gjson.GetBytes(normalized, "input.2.type").String())

	second, changedAgain, err := NormalizeCompactionTriggerInputOrder(normalized)
	require.NoError(t, err)
	require.False(t, changedAgain)
	require.Equal(t, string(normalized), string(second))
}
