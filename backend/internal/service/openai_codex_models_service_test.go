package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func newCodexModelsTestAccount() *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "test-access-token",
			"chatgpt_account_id": "acc-123",
		},
	}
}

func newCodexModelsAPIKeyTestAccount(baseURL string) *Account {
	return &Account{
		ID:       2,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": baseURL,
		},
	}
}

func newCodexModelsAPIKeyTestService() *OpenAIGatewayService {
	return &OpenAIGatewayService{cfg: &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           false,
				AllowInsecureHTTP: true,
			},
		},
	}}
}

func TestFetchCodexModelsManifestAPIKeyCustomBaseURL(t *testing.T) {
	manifestBody := `{"models":[{"slug":"gpt-5.5-codex"}]}`

	var gotPath, gotAuth, gotClientVersion string
	cpaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotClientVersion = r.URL.Query().Get("client_version")
		w.Header().Set("ETag", `W/"cpa-models"`)
		_, _ = w.Write([]byte(manifestBody))
	}))
	defer cpaServer.Close()

	chatgptRequests := 0
	chatgptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatgptRequests++
		http.Error(w, "unexpected ChatGPT request", http.StatusInternalServerError)
	}))
	defer chatgptServer.Close()
	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = chatgptServer.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := newCodexModelsAPIKeyTestService()
	account := newCodexModelsAPIKeyTestAccount(cpaServer.URL + "/v1")
	manifest, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", "")
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}

	if gotPath != "/v1/models" {
		t.Errorf("request path: got %q, want %q", gotPath, "/v1/models")
	}
	if gotAuth != "Bearer test-api-key" {
		t.Errorf("authorization header: got %q", gotAuth)
	}
	if gotClientVersion != "0.137.0" {
		t.Errorf("client_version query: got %q", gotClientVersion)
	}
	if string(manifest.Body) != manifestBody {
		t.Errorf("body not passed through verbatim: got %q", manifest.Body)
	}
	if manifest.ETag != `W/"cpa-models"` {
		t.Errorf("etag not passed through: got %q", manifest.ETag)
	}
	if chatgptRequests != 0 {
		t.Errorf("ChatGPT requests: got %d, want 0", chatgptRequests)
	}
}

func TestFetchCodexModelsManifestAPIKeyNotModified(t *testing.T) {
	var gotIfNoneMatch string
	cpaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		w.Header().Set("ETag", `W/"cpa-models"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer cpaServer.Close()

	s := newCodexModelsAPIKeyTestService()
	account := newCodexModelsAPIKeyTestAccount(cpaServer.URL + "/v1")
	manifest, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", `W/"cpa-models"`)
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}

	if !manifest.NotModified {
		t.Error("expected NotModified to be true")
	}
	if manifest.ETag != `W/"cpa-models"` {
		t.Errorf("etag not passed through: got %q", manifest.ETag)
	}
	if gotIfNoneMatch != `W/"cpa-models"` {
		t.Errorf("if-none-match header: got %q", gotIfNoneMatch)
	}
}

func TestFetchCodexModelsManifestPassthrough(t *testing.T) {
	manifestBody := `{"models":[{"slug":"gpt-5.5","display_name":"GPT-5.5"}]}`

	var gotPath, gotAuth, gotAccountID, gotOriginator, gotClientVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAccountID = r.Header.Get("chatgpt-account-id")
		gotOriginator = r.Header.Get("Originator")
		gotClientVersion = r.URL.Query().Get("client_version")
		w.Header().Set("ETag", `W/"abc123"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(manifestBody))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL + "/backend-api/codex/models"
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	manifest, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", "")
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}

	if string(manifest.Body) != manifestBody {
		t.Errorf("body not passed through verbatim: got %q", manifest.Body)
	}
	if manifest.ETag != `W/"abc123"` {
		t.Errorf("etag not passed through: got %q", manifest.ETag)
	}
	if gotPath != "/backend-api/codex/models" {
		t.Errorf("request path: got %q", gotPath)
	}
	if gotAuth != "Bearer test-access-token" {
		t.Errorf("authorization header: got %q", gotAuth)
	}
	if gotAccountID != "acc-123" {
		t.Errorf("chatgpt-account-id header: got %q", gotAccountID)
	}
	if gotOriginator != "codex_cli_rs" {
		t.Errorf("originator header: got %q", gotOriginator)
	}
	if gotClientVersion != "0.137.0" {
		t.Errorf("client_version query: got %q", gotClientVersion)
	}
}

func TestFetchCodexModelsManifestDefaultClientVersion(t *testing.T) {
	var gotClientVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClientVersion = r.URL.Query().Get("client_version")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "", ""); err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}
	if gotClientVersion != openAICodexProbeVersion {
		t.Errorf("default client_version: got %q, want %q", gotClientVersion, openAICodexProbeVersion)
	}
}

func TestFetchCodexModelsManifestNotModified(t *testing.T) {
	var gotIfNoneMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		w.Header().Set("ETag", `W/"abc123"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	manifest, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", `W/"abc123"`)
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}
	if !manifest.NotModified {
		t.Error("expected NotModified to be true")
	}
	if gotIfNoneMatch != `W/"abc123"` {
		t.Errorf("if-none-match header: got %q", gotIfNoneMatch)
	}
}

func TestFetchCodexModelsManifestUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"boom"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", ""); err == nil {
		t.Fatal("expected error for upstream 500, got nil")
	}
}

func TestFetchCodexModelsManifestMissingToken(t *testing.T) {
	account := newCodexModelsTestAccount()
	delete(account.Credentials, "access_token")

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", ""); err == nil {
		t.Fatal("expected error for missing access token, got nil")
	} else if got := infraerrors.Reason(err); got != "OPENAI_CODEX_MODELS_TOKEN_MISSING" {
		t.Fatalf("error reason: got %q, want OPENAI_CODEX_MODELS_TOKEN_MISSING", got)
	} else if got := infraerrors.Message(err); got != "account has no Codex backend access token" {
		t.Fatalf("error message: got %q", got)
	}
}

func TestFetchCodexModelsManifestAPIKeyMissingCustomBaseURL(t *testing.T) {
	account := newCodexModelsAPIKeyTestAccount("")

	s := newCodexModelsAPIKeyTestService()
	if _, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", ""); err == nil {
		t.Fatal("expected error for missing custom base_url, got nil")
	} else if got := infraerrors.Reason(err); got != "OPENAI_CODEX_MODELS_UPSTREAM_NOT_CONFIGURED" {
		t.Fatalf("error reason: got %q, want OPENAI_CODEX_MODELS_UPSTREAM_NOT_CONFIGURED", got)
	} else if got := infraerrors.Message(err); got != "account has no custom OpenAI-compatible base_url for Codex models" {
		t.Fatalf("error message: got %q", got)
	}
}

func TestFetchCodexModelsManifestAPIKeyMissingAPIKey(t *testing.T) {
	account := newCodexModelsAPIKeyTestAccount("https://cpa.example.com/v1")
	delete(account.Credentials, "api_key")

	s := newCodexModelsAPIKeyTestService()
	if _, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", ""); err == nil {
		t.Fatal("expected error for missing api_key, got nil")
	} else if got := infraerrors.Reason(err); got != "OPENAI_CODEX_MODELS_TOKEN_MISSING" {
		t.Fatalf("error reason: got %q, want OPENAI_CODEX_MODELS_TOKEN_MISSING", got)
	} else if got := infraerrors.Message(err); got != "account has no OpenAI API key for Codex models upstream" {
		t.Fatalf("error message: got %q", got)
	}
}
