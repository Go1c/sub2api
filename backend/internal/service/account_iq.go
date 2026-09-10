package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	accountIQTestMode    = "iq"
	accountIQTestModeKey = "account_iq_test_mode"
)

func rememberAccountIQTestMode(c *gin.Context, mode string) {
	if c == nil {
		return
	}
	if _, exists := c.Get(accountIQTestModeKey); exists {
		return
	}
	c.Set(accountIQTestModeKey, mode)
}

func accountIQTestModeFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(accountIQTestModeKey)
	if !ok {
		return ""
	}
	mode, _ := value.(string)
	return mode
}

// ApplyOpenAIAccountIQTestPayload 是智商检测改测试 payload 的唯一入口。
// 非 iq 模式不做任何改动，普通测试连接仍走原来的 "hi"。
func ApplyOpenAIAccountIQTestPayload(payload map[string]any, prompt, mode string, isOAuth bool) {
	if payload == nil || !isAccountIQTestMode(mode) {
		return
	}
	applyAccountIQTestPrompt(payload, prompt)
	applyAccountIQTestReasoning(payload)
	if isOAuth {
		ensureCodexReasoningInclude(payload)
	}
}

func isAccountIQTestMode(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), accountIQTestMode)
}

func applyAccountIQTestPrompt(payload map[string]any, prompt string) {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return
	}
	if _, ok := payload["messages"]; ok {
		payload["messages"] = []map[string]any{
			{
				"role":    "user",
				"content": text,
			},
		}
		return
	}
	payload["input"] = []map[string]any{
		{
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": text,
				},
			},
		},
	}
}

func applyAccountIQTestReasoning(payload map[string]any) {
	if _, ok := payload["messages"]; ok {
		payload["reasoning_effort"] = "high"
		return
	}
	payload["reasoning"] = map[string]any{"effort": "high"}
}
