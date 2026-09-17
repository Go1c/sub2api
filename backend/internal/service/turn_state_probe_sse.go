package service

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

func BuildTurnStateProbePayload(model, question string) map[string]any {
	model = strings.TrimSpace(model)
	if model == "" {
		model = turnStateProbeDefaultModel
	}
	question = strings.TrimSpace(question)
	if question == "" {
		question = turnStateProbeDefaultQuestion
	}
	return map[string]any{
		"model":  model,
		"store":  false,
		"stream": true,
		"instructions": openai.DefaultInstructions,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": question},
				},
			},
		},
	}
}

type turnStateProbeSSEResult struct {
	Model string
	Text  string
}

// parseTurnStateProbeSSE reads a Codex /responses SSE body. Text is taken from
// the first output_text.done / response.output_text.done / completed event only,
// because concatenating deltas with the done payload duplicates the answer.
func parseTurnStateProbeSSE(body io.Reader) turnStateProbeSSEResult {
	var out turnStateProbeSSEResult
	if body == nil {
		return out
	}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		eventType := gjson.Get(payload, "type").String()
		if model := strings.TrimSpace(gjson.Get(payload, "response.model").String()); model != "" && out.Model == "" {
			out.Model = model
		}
		if model := strings.TrimSpace(gjson.Get(payload, "model").String()); model != "" && out.Model == "" {
			out.Model = model
		}
		if out.Text != "" {
			continue
		}
		switch eventType {
		case "response.output_text.done", "output_text.done", "response.output_text.completed":
			if text := strings.TrimSpace(gjson.Get(payload, "text").String()); text != "" {
				out.Text = text
			}
		case "response.completed":
			if text := firstCompletedOutputText(payload); text != "" {
				out.Text = text
			}
		}
	}
	return out
}

func firstCompletedOutputText(payload string) string {
	outputs := gjson.Get(payload, "response.output")
	if !outputs.IsArray() {
		return ""
	}
	var text string
	outputs.ForEach(func(_, item gjson.Result) bool {
		content := item.Get("content")
		if !content.IsArray() {
			return true
		}
		content.ForEach(func(_, part gjson.Result) bool {
			if part.Get("type").String() == "output_text" {
				if v := strings.TrimSpace(part.Get("text").String()); v != "" {
					text = v
					return false
				}
			}
			return true
		})
		return text == ""
	})
	return text
}

func marshalTurnStateProbePayload(model, question string) ([]byte, error) {
	return json.Marshal(BuildTurnStateProbePayload(model, question))
}
