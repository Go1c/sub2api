package service

// fork 侧 Grok 转发辅助符号（自 dev 的 openai_gateway_grok.go 抽出，供
// openai_gateway_grok_native.go 使用）。

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	grokUpstreamUserAgent     = "sub2api-grok/1.0"
	grokRejectedSearchInclude = "x_search_call.action.sources"
)

func stripRejectedGrokSearchInclude(body []byte) ([]byte, error) {
	include := gjson.GetBytes(body, "include")
	if !include.Exists() {
		return body, nil
	}
	if !include.IsArray() {
		if strings.TrimSpace(include.String()) == grokRejectedSearchInclude {
			return sjson.DeleteBytes(body, "include")
		}
		return body, nil
	}

	kept := make([]json.RawMessage, 0, len(include.Array()))
	changed := false
	for _, item := range include.Array() {
		if strings.TrimSpace(item.String()) == grokRejectedSearchInclude {
			changed = true
			continue
		}
		kept = append(kept, json.RawMessage(item.Raw))
	}
	if !changed {
		return body, nil
	}
	if len(kept) == 0 {
		return sjson.DeleteBytes(body, "include")
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(body, "include", encoded)
}
