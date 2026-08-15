package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolateOpenAISessionID(t *testing.T) {
	t.Run("empty_raw_returns_empty", func(t *testing.T) {
		assert.Equal(t, "", isolateOpenAISessionID(1, "", nil))
		assert.Equal(t, "", isolateOpenAISessionID(1, "   ", nil))
	})

	t.Run("deterministic", func(t *testing.T) {
		a := isolateOpenAISessionID(42, "sess_abc123", nil)
		b := isolateOpenAISessionID(42, "sess_abc123", nil)
		assert.Equal(t, a, b)
	})

	t.Run("different_apiKeyID_different_result", func(t *testing.T) {
		a := isolateOpenAISessionID(1, "same_session", nil)
		b := isolateOpenAISessionID(2, "same_session", nil)
		require.NotEqual(t, a, b, "不同 API Key 使用相同 session_id 应产生不同隔离值")
	})

	t.Run("different_raw_different_result", func(t *testing.T) {
		a := isolateOpenAISessionID(1, "session_a", nil)
		b := isolateOpenAISessionID(1, "session_b", nil)
		require.NotEqual(t, a, b)
	})

	t.Run("format_is_16_hex_chars", func(t *testing.T) {
		result := isolateOpenAISessionID(99, "test_session", nil)
		assert.Len(t, result, 16, "应为 16 字符的 hex 字符串")
		for _, ch := range result {
			assert.True(t, (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
				"应仅包含 hex 字符: %c", ch)
		}
	})

	t.Run("zero_apiKeyID_still_works", func(t *testing.T) {
		result := isolateOpenAISessionID(0, "session", nil)
		assert.NotEmpty(t, result)
		// apiKeyID=0 与 apiKeyID=1 应产生不同结果
		other := isolateOpenAISessionID(1, "session", nil)
		assert.NotEqual(t, result, other)
	})
}
