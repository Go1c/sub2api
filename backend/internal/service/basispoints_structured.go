package service

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const basisPointsStructuredSchemaURL = "https://basispoints.invalid/structured-output.json"

var basisPointsStructuredFormatName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// BPS rejects native text.format. The requested format becomes developer
// instructions, and the final answer is validated locally before any of its
// text is shown to the client.
type basisPointsStructuredOutput struct {
	format map[string]any
	schema *jsonschema.Schema
}

type basisPointsLocalSchemaLoader struct{}

func (basisPointsLocalSchemaLoader) Load(string) (any, error) {
	return nil, fmt.Errorf("external structured output schema references are not supported")
}

func prepareBasisPointsStructuredOutput(body []byte) (*basisPointsStructuredOutput, error) {
	textNode := gjson.GetBytes(body, "text")
	if !textNode.Exists() || textNode.Type == gjson.Null {
		return nil, nil
	}
	if !textNode.IsObject() {
		return nil, fmt.Errorf("basispoints text must be an object")
	}
	formatNode := textNode.Get("format")
	if !formatNode.Exists() || formatNode.Type == gjson.Null {
		return nil, nil
	}
	if !formatNode.IsObject() {
		return nil, fmt.Errorf("basispoints text.format must be an object")
	}
	var format map[string]any
	if err := json.Unmarshal([]byte(formatNode.Raw), &format); err != nil || format == nil {
		return nil, fmt.Errorf("basispoints text.format must be an object")
	}
	kind := basisPointsText(format["type"])
	if kind == "text" {
		return nil, nil
	}
	if kind != "json_object" && kind != "json_schema" {
		return nil, fmt.Errorf("basispoints text.format requires text, json_object or json_schema")
	}
	for key := range format {
		if key == "type" || (kind == "json_schema" && (key == "name" || key == "schema" || key == "strict" || key == "description")) {
			continue
		}
		return nil, fmt.Errorf("basispoints text.format contains an unsupported field")
	}
	result := &basisPointsStructuredOutput{format: format}
	if kind == "json_object" {
		return result, nil
	}
	if !basisPointsStructuredFormatName.MatchString(basisPointsText(format["name"])) {
		return nil, fmt.Errorf("basispoints json_schema requires a name of 1-64 letters, digits, underscores or hyphens")
	}
	if value, exists := format["strict"]; exists && value != nil {
		if _, ok := value.(bool); !ok {
			return nil, fmt.Errorf("basispoints json_schema strict must be a boolean")
		}
	}
	if value, exists := format["description"]; exists && value != nil {
		if _, ok := value.(string); !ok {
			return nil, fmt.Errorf("basispoints json_schema description must be a string")
		}
	}
	schema, ok := format["schema"].(map[string]any)
	if !ok || schema == nil {
		return nil, fmt.Errorf("basispoints json_schema requires a schema object")
	}
	encoded, err := json.Marshal(schema)
	if err != nil || len(encoded) > 1<<20 {
		return nil, fmt.Errorf("basispoints structured output schema exceeds 1 MiB")
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(basisPointsLocalSchemaLoader{})
	if err := compiler.AddResource(basisPointsStructuredSchemaURL, schema); err != nil {
		return nil, fmt.Errorf("basispoints structured output schema is invalid")
	}
	result.schema, err = compiler.Compile(basisPointsStructuredSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("basispoints structured output schema is invalid or references an external resource")
	}
	return result, nil
}

func (s *basisPointsStructuredOutput) instructions() string {
	if s == nil {
		return ""
	}
	prompt := "The client requires a structured final answer. Your final assistant answer must be exactly one JSON value, with no Markdown fences or surrounding prose. " +
		"Tool calls and refusals remain separate protocol items; use the client tool transport as needed before the final answer. " +
		"The gateway validates the final JSON before returning it to the client."
	if s.schema != nil {
		encoded, _ := json.Marshal(s.format)
		prompt += " The final answer must satisfy the schema in this output format: \n" + string(encoded)
	}
	return prompt
}

func (s *basisPointsStructuredOutput) validate(response map[string]any) error {
	if s == nil {
		return nil
	}
	output, _ := response["output"].([]any)
	var answer strings.Builder
	hasTool, hasRefusal := false, false
	for _, raw := range output {
		item, _ := raw.(map[string]any)
		itemType := basisPointsText(item["type"])
		hasTool = hasTool || itemType == "function_call" || itemType == "custom_tool_call"
		if itemType != "message" {
			continue
		}
		content, _ := item["content"].([]any)
		for _, rawPart := range content {
			part, _ := rawPart.(map[string]any)
			switch basisPointsText(part["type"]) {
			case "output_text":
				value, ok := part["text"].(string)
				if !ok || answer.Len()+len(value) > 16<<20 {
					return fmt.Errorf("basispoints structured output text is invalid or exceeds 16 MiB")
				}
				_, _ = answer.WriteString(value)
			case "refusal":
				hasRefusal = true
			default:
				return fmt.Errorf("basispoints structured output contains unsupported message content")
			}
		}
	}
	if hasTool || (hasRefusal && answer.Len() == 0) {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(answer.String()))
	decoder.UseNumber()
	var instance any
	if err := decoder.Decode(&instance); err != nil {
		return fmt.Errorf("basispoints structured output is not one valid JSON value")
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil && err != io.EOF {
		return fmt.Errorf("basispoints structured output is not one valid JSON value")
	}
	if s.schema != nil && s.schema.Validate(instance) != nil {
		return fmt.Errorf("basispoints structured output does not satisfy the requested JSON schema")
	}
	return nil
}

func basisPointsStructuredMessageEvent(kind string, item gjson.Result) bool {
	if strings.HasPrefix(kind, "response.output_text.") || strings.HasPrefix(kind, "response.refusal.") || strings.HasPrefix(kind, "response.content_part.") {
		return true
	}
	return strings.HasPrefix(kind, "response.output_item.") && item.Get("type").String() == "message"
}

func applyBasisPointsStructuredResponse(body []byte, format *basisPointsStructuredOutput) ([]byte, error) {
	if format == nil || len(body) == 0 {
		return body, nil
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("basispoints structured output is missing its terminal response")
	}
	if err := format.validate(response); err != nil {
		return nil, err
	}
	return basisPointsApplyStructuredFormat(body, format.format)
}

func writeBasisPointsStructuredMessages(builder *strings.Builder, response []byte) error {
	output := gjson.GetBytes(response, "output")
	if !output.IsArray() {
		return nil
	}
	for index, item := range output.Array() {
		if item.Get("type").String() != "message" {
			continue
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(item.Raw), &decoded); err != nil {
			return err
		}
		if err := emitBasisPointsStructuredMessage(builder, decoded, index); err != nil {
			return err
		}
	}
	return nil
}

func emitBasisPointsStructuredMessage(builder *strings.Builder, item map[string]any, index int) error {
	id := basisPointsText(item["id"])
	added := make(map[string]any, len(item))
	for key, value := range item {
		added[key] = value
	}
	added["content"] = []any{}
	added["status"] = "in_progress"
	if err := writeBasisPointsStructuredEvent(builder, "response.output_item.added", map[string]any{"output_index": index, "item": added}); err != nil {
		return err
	}
	content, _ := item["content"].([]any)
	for i, raw := range content {
		part, _ := raw.(map[string]any)
		field, prefix := "text", "response.output_text"
		if basisPointsText(part["type"]) == "refusal" {
			field, prefix = "refusal", "response.refusal"
		}
		empty := make(map[string]any, len(part))
		for key, value := range part {
			empty[key] = value
		}
		empty[field] = ""
		events := []struct {
			kind    string
			payload map[string]any
		}{
			{"response.content_part.added", map[string]any{"part": empty}},
			{prefix + ".delta", map[string]any{"delta": part[field]}},
			{prefix + ".done", map[string]any{field: part[field]}},
			{"response.content_part.done", map[string]any{"part": part}},
		}
		for _, event := range events {
			event.payload["output_index"] = index
			event.payload["item_id"] = id
			event.payload["content_index"] = i
			if err := writeBasisPointsStructuredEvent(builder, event.kind, event.payload); err != nil {
				return err
			}
		}
	}
	return writeBasisPointsStructuredEvent(builder, "response.output_item.done", map[string]any{"output_index": index, "item": item})
}

func writeBasisPointsStructuredEvent(builder *strings.Builder, event string, payload map[string]any) error {
	payload["type"] = event
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	writeBasisPointsSSE(builder, event, encoded)
	return nil
}

func basisPointsApplyStructuredFormat(response []byte, format map[string]any) ([]byte, error) {
	if format == nil {
		return response, nil
	}
	encoded, err := json.Marshal(format)
	if err != nil {
		return nil, err
	}
	if !gjson.GetBytes(response, "text").Exists() {
		response, err = sjson.SetRawBytes(response, "text", []byte(`{}`))
		if err != nil {
			return nil, err
		}
	}
	return sjson.SetRawBytes(response, "text.format", encoded)
}
