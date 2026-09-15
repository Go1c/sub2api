package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	accountIQTestMode            = "iq"
	accountIQTestModeKey         = "account_iq_test_mode"
	accountIQTestReasoningEffort = "low"
	accountIQTestMaxAttempts     = 3
	accountIQTestInstructions    = "You are taking a drawing test. Reply with one complete <svg xmlns=\"http://www.w3.org/2000/svg\">...</svg> in this message. Do not ask questions, greet, choose a filename, use tools, or write files."
	accountIQTestRetryExhausted  = "上游多次过载，请稍后重试"
)

var errAccountIQTestRetryable = errors.New("account iq test retryable")

func accountIQTestRetryStatusText(attempt, max int) string {
	return fmt.Sprintf("上游瞬时失败，正在重试 (%d/%d)", attempt, max)
}

func accountIQTestRetryableHTTP(status int, body []byte) bool {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	if status >= http.StatusInternalServerError {
		return true
	}
	return isOpenAIRequestScopedCapacityShed("", body)
}

func accountIQTestRetryableStreamFailure(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if strings.Contains(lower, "invalid_api_key") ||
		strings.Contains(lower, "incorrect api key") ||
		strings.Contains(lower, "unauthorized") {
		return false
	}
	return true
}

func (s *AccountTestService) runWithAccountIQRetries(c *gin.Context, run func() error) error {
	attempts := 1
	if isAccountIQTestMode(accountIQTestModeFromContext(c)) {
		attempts = accountIQTestMaxAttempts
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := c.Request.Context().Err(); err != nil {
			return s.sendErrorAndEnd(c, err.Error())
		}
		if attempt > 1 {
			s.sendEvent(c, TestEvent{Type: "status", Text: accountIQTestRetryStatusText(attempt, attempts)})
		}
		err := run()
		if err == nil {
			return nil
		}
		if !errors.Is(err, errAccountIQTestRetryable) {
			return err
		}
		if attempt == attempts {
			return s.sendErrorAndEnd(c, accountIQTestRetryExhausted)
		}
	}
	return s.sendErrorAndEnd(c, accountIQTestRetryExhausted)
}

func (s *AccountTestService) accountIQRetryOrEnd(c *gin.Context, retryable bool, errMsg string) error {
	if retryable && isAccountIQTestMode(accountIQTestModeFromContext(c)) {
		return errAccountIQTestRetryable
	}
	return s.sendErrorAndEnd(c, errMsg)
}

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
	applyAccountIQTestInstructions(payload)
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
				"role":    "system",
				"content": accountIQTestInstructions,
			},
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

func applyAccountIQTestInstructions(payload map[string]any) {
	if _, ok := payload["messages"]; ok {
		return
	}
	payload["instructions"] = accountIQTestInstructions
}

func applyAccountIQTestReasoning(payload map[string]any) {
	if _, ok := payload["messages"]; ok {
		payload["reasoning_effort"] = accountIQTestReasoningEffort
		return
	}
	payload["reasoning"] = map[string]any{"effort": accountIQTestReasoningEffort}
}

func extractCompleteSVG(text string) string {
	lower := strings.ToLower(text)
	end := strings.LastIndex(lower, "</svg>")
	if end < 0 {
		return ""
	}
	start := strings.LastIndex(lower[:end], "<svg")
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(text[start : end+len("</svg>")])
}

func openAIStreamUsageTokens(data map[string]any) int64 {
	response, _ := data["response"].(map[string]any)
	if response == nil {
		return 0
	}
	usage, _ := response["usage"].(map[string]any)
	if usage == nil {
		return 0
	}
	if total := jsonNumberToInt64(usage["total_tokens"]); total > 0 {
		return total
	}
	return jsonNumberToInt64(usage["input_tokens"]) + jsonNumberToInt64(usage["output_tokens"])
}

func jsonNumberToInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		parsed, _ := n.Int64()
		return parsed
	default:
		return 0
	}
}
