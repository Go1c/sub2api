//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestGrokProviderUsesOpenAIV1ChatCompletions 确认 Grok 渠道监控走 OpenAI 兼容
// /v1/chat/completions + Bearer auth，并复用 OpenAI 响应解析。
func TestGrokProviderUsesOpenAIV1ChatCompletions(t *testing.T) {
	if !isSupportedProvider(MonitorProviderGrok) {
		t.Fatal("grok should be a supported channel monitor provider")
	}

	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	endpoint := setupFakeMonitorProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"42"}}]}`))
	}))

	text, _, status, err := callProvider(
		context.Background(),
		MonitorProviderGrok,
		endpoint,
		"sk-test",
		"grok-3",
		"What is 6*7?",
		nil,
	)
	if err != nil {
		t.Fatalf("callProvider: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d", status)
	}
	if gotPath != providerOpenAIPath {
		t.Fatalf("path=%q want %q", gotPath, providerOpenAIPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotBody["model"] != "grok-3" {
		t.Fatalf("model=%v", gotBody["model"])
	}
	if text != "42" {
		t.Fatalf("text=%q", text)
	}
}

func TestValidateProviderAcceptsGrok(t *testing.T) {
	if err := validateProvider(MonitorProviderGrok); err != nil {
		t.Fatalf("validateProvider(grok): %v", err)
	}
}
