package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWidgetConfigReturnsOnlyPublicConfigWithCors(t *testing.T) {
	handler := newServer(Config{
		DocsGPTAPIBaseURL:   "https://docsgpt.internal",
		DocsGPTAgentAPIKey:  "agent-secret",
		AllowedOrigins:      []string{"https://app.example.com"},
		SupportEmail:        "support@example.com",
		SupportURL:          "https://support.example.com",
		OfficialContactText: "Contact official support",
		WidgetTitle:         "LumioAPI Support",
		WelcomeMessage:      "How can we help?",
		RateLimitWindow:     time.Minute,
		RateLimitMax:        10,
	}, http.DefaultClient)

	req := httptest.NewRequest(http.MethodGet, "/widget-config", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected CORS origin to be echoed, got %q", got)
	}
	if strings.Contains(rec.Body.String(), "agent-secret") {
		t.Fatalf("widget config leaked the DocsGPT agent key: %s", rec.Body.String())
	}

	var body widgetConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode widget config: %v", err)
	}
	if body.SupportEmail != "support@example.com" || body.SupportURL != "https://support.example.com" {
		t.Fatalf("unexpected support config: %+v", body)
	}
}

func TestWidgetConfigLocalizesDefaultsByLocale(t *testing.T) {
	handler := newServer(Config{
		DocsGPTAPIBaseURL:  "https://docsgpt.internal",
		DocsGPTAgentAPIKey: "agent-secret",
		AllowedOrigins:     []string{"https://app.example.com"},
		RateLimitWindow:    time.Minute,
		RateLimitMax:       10,
	}, http.DefaultClient)

	req := httptest.NewRequest(http.MethodGet, "/widget-config?locale=zh-Hant", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body widgetConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode widget config: %v", err)
	}
	if body.Title != "LumioAPI 客服" || body.OfficialContactText != "聯絡官方支援" {
		t.Fatalf("expected Traditional Chinese defaults, got %+v", body)
	}
	if !strings.Contains(body.WelcomeMessage, "文件") {
		t.Fatalf("expected Traditional Chinese welcome message, got %q", body.WelcomeMessage)
	}
}

func TestChatStreamInjectsAgentKeyAndForwardsSSE(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/stream" {
			t.Fatalf("expected /stream, got %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("expected event-stream accept header, got %q", r.Header.Get("Accept"))
		}

		var payload docsGPTStreamRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}
		if payload.APIKey != "agent-secret" {
			t.Fatalf("expected gateway to inject api key, got %q", payload.APIKey)
		}
		if payload.Question != "How do I recharge?" || payload.ConversationID != "conv-1" {
			t.Fatalf("unexpected upstream payload: %+v", payload)
		}
		if payload.Passthrough["user_id"] != "u-1" || payload.Passthrough["user_email"] != "u@example.com" {
			t.Fatalf("expected user passthrough, got %+v", payload.Passthrough)
		}
		if payload.Passthrough["locale"] != "zh-Hant" || payload.Passthrough["language"] != "Traditional Chinese" {
			t.Fatalf("expected locale passthrough, got %+v", payload.Passthrough)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"answer\",\"answer\":\"Use the billing page.\"}\n\n" +
					"data: {\"type\":\"id\",\"conversation_id\":\"conv-2\"}\n\n" +
					"data: {\"type\":\"end\"}\n\n",
			)),
			Request: r,
		}, nil
	})}

	handler := newServer(Config{
		DocsGPTAPIBaseURL:  "https://docsgpt.internal",
		DocsGPTAgentAPIKey: "agent-secret",
		AllowedOrigins:     []string{"https://app.example.com"},
		RateLimitWindow:    time.Minute,
		RateLimitMax:       10,
	}, client)

	body := bytes.NewBufferString(`{"message":"How do I recharge?","conversationId":"conv-1","locale":"zh-Hant","user":{"id":"u-1","email":"u@example.com"}}`)
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", body)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"answer":"Use the billing page."`) {
		t.Fatalf("gateway did not forward upstream SSE: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "agent-secret") {
		t.Fatalf("chat stream leaked the DocsGPT agent key: %s", rec.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestChatStreamRateLimitsByClientIP(t *testing.T) {
	handler := newServer(Config{
		DocsGPTAPIBaseURL:  "https://docsgpt.internal",
		DocsGPTAgentAPIKey: "agent-secret",
		AllowedOrigins:     []string{"*"},
		RateLimitWindow:    time.Minute,
		RateLimitMax:       1,
	}, http.DefaultClient)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/chat/stream", bytes.NewBufferString(`{"message":"hello"}`))
		req.Header.Set("X-Forwarded-For", "203.0.113.10")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected second request to be rate limited, got %d", rec.Code)
		}
	}
}
