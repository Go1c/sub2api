package apicompat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	toolOutputMediaMarker      = "[Tool output media moved to the following user message]"
	toolOutputMediaAttribution = "[Tool output media for call %s]"
)

type responsesToolOutputMedia struct {
	callID   string
	imageURL string
}

// LiftResponsesToolOutputMedia moves image parts out of Responses tool outputs
// and into a following user message. Native Responses endpoints such as
// DeepSeek accept function_call_output.output as a string, but Codex view_image
// returns an array containing input_image. Keeping the image inside the tool
// output makes the upstream report "No tool output found for tool call ...".
func LiftResponsesToolOutputMedia(input any) (any, bool) {
	items, ok := input.([]any)
	if !ok {
		return input, false
	}

	rewritten := make([]any, 0, len(items)+1)
	changed := false

	// DeepSeek validates parallel tool outputs as one contiguous run. Codex can
	// inject developer notices between image outputs, so keep those notices
	// pending until the full batch and its lifted media have been emitted.
	for index := 0; index < len(items); {
		item, ok := items[index].(map[string]any)
		if !ok || !isResponsesToolOutputItem(item) {
			rewritten = append(rewritten, items[index])
			index++
			continue
		}

		batchStart := index
		outputs := make([]any, 0)
		trailing := make([]any, 0)
		pending := make([]responsesToolOutputMedia, 0)
		batchChanged := false

		for index < len(items) {
			rawItem := items[index]
			item, ok := rawItem.(map[string]any)
			if !ok {
				break
			}
			if isResponsesToolOutputItem(item) {
				rewrittenItem, media, didRewrite := liftResponsesToolOutputMediaItem(item)
				if didRewrite {
					batchChanged = true
					pending = append(pending, media...)
				}
				outputs = append(outputs, rewrittenItem)
				index++
				continue
			}
			if isResponsesToolBatchInstruction(item) {
				trailing = append(trailing, rawItem)
				index++
				continue
			}
			break
		}

		if !batchChanged {
			rewritten = append(rewritten, items[batchStart:index]...)
			continue
		}

		rewritten = append(rewritten, outputs...)
		if len(pending) > 0 {
			rewritten = append(rewritten, buildResponsesToolOutputMediaMessage(pending))
		}
		rewritten = append(rewritten, trailing...)
		changed = true
	}

	if !changed {
		return input, false
	}
	return rewritten, true
}

func liftResponsesToolOutputMediaItem(item map[string]any) (any, []responsesToolOutputMedia, bool) {
	output, exists := item["output"]
	if !exists {
		return item, nil, false
	}
	outputRaw, err := json.Marshal(output)
	if err != nil {
		return item, nil, false
	}

	outputText, media, didRewrite := extractToolOutputMedia(outputRaw)
	if !didRewrite {
		return item, nil, false
	}

	item["output"] = outputText
	callID := strings.TrimSpace(stringValue(item["call_id"]))
	lifted := make([]responsesToolOutputMedia, 0, len(media))
	for _, part := range media {
		if part.ImageURL == nil {
			continue
		}
		imageURL := strings.TrimSpace(part.ImageURL.URL)
		if imageURL == "" {
			continue
		}
		lifted = append(lifted, responsesToolOutputMedia{
			callID:   callID,
			imageURL: imageURL,
		})
	}
	return item, lifted, true
}

func buildResponsesToolOutputMediaMessage(pending []responsesToolOutputMedia) map[string]any {
	content := make([]map[string]any, 0, len(pending)*2)
	lastCallID := ""
	for _, media := range pending {
		if media.callID != lastCallID {
			text := "Tool output media"
			if media.callID != "" {
				text = fmt.Sprintf(toolOutputMediaAttribution, media.callID)
			}
			content = append(content, map[string]any{
				"type": "input_text",
				"text": text,
			})
			lastCallID = media.callID
		}
		content = append(content, map[string]any{
			"type":      "input_image",
			"image_url": media.imageURL,
		})
	}

	return map[string]any{
		"type":    "message",
		"role":    "user",
		"content": content,
	}
}

func isResponsesToolBatchInstruction(item map[string]any) bool {
	itemType := strings.TrimSpace(stringValue(item["type"]))
	if itemType != "" && itemType != "message" {
		return false
	}
	role := strings.TrimSpace(stringValue(item["role"]))
	return role == "developer" || role == "system"
}

func isResponsesToolOutputItem(item map[string]any) bool {
	switch strings.TrimSpace(stringValue(item["type"])) {
	case "function_call_output", "custom_tool_call_output",
		"tool_search_output", "tool_search_call_output", "mcp_tool_call_output":
		return true
	default:
		return false
	}
}

// extractToolOutputMedia rewrites only recognized image nodes. Media-free
// outputs return rewritten=false so the caller can preserve their original
// bytes and prompt-cache prefix.
func extractToolOutputMedia(outputRaw json.RawMessage) (string, []ChatContentPart, bool) {
	outputRaw = bytesTrimSpace(outputRaw)
	if len(outputRaw) == 0 || string(outputRaw) == "null" {
		return "", nil, false
	}

	var outputString string
	if err := json.Unmarshal(outputRaw, &outputString); err == nil {
		if isToolOutputImageDataURL(outputString) {
			return toolOutputMediaMarker, []ChatContentPart{toolOutputImagePart(outputString)}, true
		}

		nested, ok := decodeToolOutputJSON([]byte(outputString))
		if !ok {
			return "", nil, false
		}
		rewritten, media, changed := rewriteToolOutputMediaValue(nested)
		if !changed {
			return "", nil, false
		}
		encoded, err := json.Marshal(rewritten)
		if err != nil {
			return "", nil, false
		}
		return string(encoded), media, true
	}

	value, ok := decodeToolOutputJSON(outputRaw)
	if !ok {
		return "", nil, false
	}
	rewritten, media, changed := rewriteToolOutputMediaValue(value)
	if !changed {
		return "", nil, false
	}
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		return "", nil, false
	}
	return string(encoded), media, true
}

func decodeToolOutputJSON(raw []byte) (any, bool) {
	if !json.Valid(raw) {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	return value, true
}

func rewriteToolOutputMediaValue(value any) (any, []ChatContentPart, bool) {
	switch typed := value.(type) {
	case []any:
		var media []ChatContentPart
		changed := false
		for i, item := range typed {
			rewritten, itemMedia, itemChanged := rewriteToolOutputMediaValue(item)
			if !itemChanged {
				continue
			}
			typed[i] = rewritten
			media = append(media, itemMedia...)
			changed = true
		}
		return typed, media, changed
	case map[string]any:
		if imageURL, ok := recognizedToolOutputImageURL(typed); ok {
			return map[string]any{
				"type": "input_text",
				"text": toolOutputMediaMarker,
			}, []ChatContentPart{toolOutputImagePart(imageURL)}, true
		}

		content, ok := typed["content"]
		if !ok {
			return typed, nil, false
		}
		rewritten, media, changed := rewriteToolOutputMediaValue(content)
		if !changed {
			return typed, nil, false
		}
		typed["content"] = rewritten
		return typed, media, true
	default:
		return value, nil, false
	}
}

func recognizedToolOutputImageURL(value map[string]any) (string, bool) {
	partType, _ := value["type"].(string)
	if partType != "input_image" && partType != "image_url" {
		return "", false
	}

	switch imageURL := value["image_url"].(type) {
	case string:
		return imageURL, strings.TrimSpace(imageURL) != ""
	case map[string]any:
		url, _ := imageURL["url"].(string)
		return url, strings.TrimSpace(url) != ""
	default:
		return "", false
	}
}

func isToolOutputImageDataURL(value string) bool {
	const prefix = "data:image/"
	const separator = ";base64,"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	separatorIndex := strings.Index(value[len(prefix):], separator)
	if separatorIndex <= 0 {
		return false
	}
	payloadIndex := len(prefix) + separatorIndex + len(separator)
	return payloadIndex < len(value)
}

func toolOutputImagePart(imageURL string) ChatContentPart {
	return ChatContentPart{
		Type:     "image_url",
		ImageURL: &ChatImageURL{URL: imageURL},
	}
}
