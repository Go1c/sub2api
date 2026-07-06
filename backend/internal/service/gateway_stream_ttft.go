package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/tidwall/gjson"
)

func openAIResponsesStreamEventsHaveOutputDelta(events []apicompat.ResponsesStreamEvent) bool {
	for _, event := range events {
		if openAIResponsesStreamEventHasOutputDelta(event) {
			return true
		}
	}
	return false
}

func openAIResponsesStreamEventHasOutputDelta(event apicompat.ResponsesStreamEvent) bool {
	switch strings.TrimSpace(event.Type) {
	case "response.output_text.delta", "response.reasoning_summary_text.delta", "response.function_call_arguments.delta":
		return event.Delta != ""
	default:
		return false
	}
}

func openAIResponsesStreamDataHasOutputDelta(data, eventType string) bool {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || trimmed == "[DONE]" {
		return false
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = strings.TrimSpace(gjson.Get(trimmed, "type").String())
	}
	switch eventType {
	case "response.output_text.delta", "response.reasoning_summary_text.delta", "response.function_call_arguments.delta":
		return gjson.Get(trimmed, "delta").String() != ""
	default:
		return false
	}
}

func openAIResponsesStreamDataShouldStartClientStream(data, eventType string) bool {
	if openAIResponsesStreamDataHasOutputDelta(data, eventType) {
		return true
	}
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return false
	}
	if trimmed == "[DONE]" {
		return true
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "response.failed" {
		return false
	}
	return openAIStreamEventIsTerminal(trimmed)
}

func anthropicStreamDataHasOutputDelta(data string) bool {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || trimmed == "[DONE]" {
		return false
	}
	var event apicompat.AnthropicStreamEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		return false
	}
	return anthropicStreamEventHasOutputDelta(&event)
}

func anthropicStreamEventHasOutputDelta(event *apicompat.AnthropicStreamEvent) bool {
	if event == nil || event.Delta == nil {
		return false
	}
	if event.Type != "" && event.Type != "content_block_delta" {
		return false
	}
	switch event.Delta.Type {
	case "text_delta":
		return event.Delta.Text != ""
	case "thinking_delta":
		return event.Delta.Thinking != ""
	case "input_json_delta":
		return event.Delta.PartialJSON != ""
	case "":
		return event.Delta.Text != "" || event.Delta.Thinking != "" || event.Delta.PartialJSON != ""
	default:
		return false
	}
}
