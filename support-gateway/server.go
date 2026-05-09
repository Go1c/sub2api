package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type widgetConfigResponse struct {
	Title               string `json:"title"`
	WelcomeMessage      string `json:"welcomeMessage"`
	SupportEmail        string `json:"supportEmail,omitempty"`
	SupportURL          string `json:"supportUrl,omitempty"`
	OfficialContactText string `json:"officialContactText"`
}

type chatStreamRequest struct {
	Message        string    `json:"message"`
	ConversationID string    `json:"conversationId,omitempty"`
	Locale         string    `json:"locale,omitempty"`
	User           *chatUser `json:"user,omitempty"`
}

type chatUser struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type docsGPTStreamRequest struct {
	Question       string            `json:"question"`
	APIKey         string            `json:"api_key"`
	ConversationID string            `json:"conversation_id,omitempty"`
	Passthrough    map[string]string `json:"passthrough,omitempty"`
}

type gatewayServer struct {
	cfg     Config
	client  *http.Client
	limiter *rateLimiter
	mux     *http.ServeMux
}

func newServer(cfg Config, client *http.Client) http.Handler {
	normalizeConfig(&cfg)
	if client == nil {
		client = http.DefaultClient
	}

	server := &gatewayServer{
		cfg:     cfg,
		client:  client,
		limiter: newRateLimiter(cfg.RateLimitWindow, cfg.RateLimitMax),
		mux:     http.NewServeMux(),
	}
	server.mux.HandleFunc("/healthz", server.handleHealthz)
	server.mux.HandleFunc("/widget-config", server.handleWidgetConfig)
	server.mux.HandleFunc("/chat/stream", server.handleChatStream)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.applyCORS(w, r) {
			return
		}
		server.mux.ServeHTTP(w, r)
	})
}

func normalizeConfig(cfg *Config) {
	cfg.DocsGPTAPIBaseURL = strings.TrimRight(strings.TrimSpace(cfg.DocsGPTAPIBaseURL), "/")
	if cfg.RateLimitWindow <= 0 {
		cfg.RateLimitWindow = time.Minute
	}
	if cfg.RateLimitMax <= 0 {
		cfg.RateLimitMax = 20
	}
}

func (s *gatewayServer) applyCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	if origin == "" {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	}

	if !originAllowed(origin, s.cfg.AllowedOrigins) {
		http.Error(w, "origin is not allowed", http.StatusForbidden)
		return true
	}

	w.Header().Set("Vary", "Origin")
	if allowsWildcard(s.cfg.AllowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func (s *gatewayServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *gatewayServer) handleWidgetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.widgetConfig(r.URL.Query().Get("locale")))
}

func (s *gatewayServer) widgetConfig(locale string) widgetConfigResponse {
	config := localizedWidgetDefaults(locale)
	if s.cfg.WidgetTitle != "" {
		config.Title = s.cfg.WidgetTitle
	}
	if s.cfg.WelcomeMessage != "" {
		config.WelcomeMessage = s.cfg.WelcomeMessage
	}
	if s.cfg.OfficialContactText != "" {
		config.OfficialContactText = s.cfg.OfficialContactText
	}
	config.SupportEmail = s.cfg.SupportEmail
	config.SupportURL = s.cfg.SupportURL
	return config
}

func localizedWidgetDefaults(locale string) widgetConfigResponse {
	switch responseLanguage(locale) {
	case "Traditional Chinese":
		return widgetConfigResponse{
			Title:               "LumioAPI 客服",
			WelcomeMessage:      "你好，我會優先基於 LumioAPI 文件回答。涉及帳戶、付款或無法確認的問題，請聯絡官方支援。",
			OfficialContactText: "聯絡官方支援",
		}
	case "Simplified Chinese":
		return widgetConfigResponse{
			Title:               "LumioAPI 客服",
			WelcomeMessage:      "你好，我会优先基于 LumioAPI 文档回答。涉及账户、支付或无法确认的问题，请联系官方支持。",
			OfficialContactText: "联系官方支持",
		}
	default:
		return widgetConfigResponse{
			Title:               "LumioAPI Support",
			WelcomeMessage:      "Ask a question and the AI support assistant will answer from the LumioAPI docs.",
			OfficialContactText: "Contact official support",
		}
	}
}

func (s *gatewayServer) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	clientIP := requestIP(r)
	if !s.limiter.allow(clientIP) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	var input chatStreamRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	if s.cfg.DocsGPTAPIBaseURL == "" || s.cfg.DocsGPTAgentAPIKey == "" {
		writeSSEError(w, http.StatusServiceUnavailable, "support chat is not configured")
		return
	}

	payload := docsGPTStreamRequest{
		Question:       input.Message,
		APIKey:         s.cfg.DocsGPTAgentAPIKey,
		ConversationID: strings.TrimSpace(input.ConversationID),
		Passthrough:    requestPassthrough(input.User, input.Locale),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		writeSSEError(w, http.StatusInternalServerError, "failed to prepare upstream request")
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.cfg.DocsGPTAPIBaseURL+"/stream", bytes.NewReader(body))
	if err != nil {
		writeSSEError(w, http.StatusBadGateway, "invalid DocsGPT upstream URL")
		return
	}
	upstreamReq.Header.Set("Accept", "text/event-stream")
	upstreamReq.Header.Set("Content-Type", "application/json")

	upstreamRes, err := s.client.Do(upstreamReq)
	if err != nil {
		writeSSEError(w, http.StatusBadGateway, "DocsGPT upstream is unavailable")
		return
	}
	defer upstreamRes.Body.Close()

	if upstreamRes.StatusCode < 200 || upstreamRes.StatusCode >= 300 {
		writeSSEError(w, http.StatusBadGateway, "DocsGPT upstream returned "+upstreamRes.Status)
		return
	}

	startSSE(w, http.StatusOK)
	_, _ = io.Copy(flushWriter{ResponseWriter: w}, upstreamRes.Body)
}

func requestPassthrough(user *chatUser, locale string) map[string]string {
	passthrough := map[string]string{}
	if user != nil {
		if user.ID != "" {
			passthrough["user_id"] = user.ID
		}
		if user.Email != "" {
			passthrough["user_email"] = user.Email
		}
	}
	if locale != "" {
		passthrough["locale"] = locale
		passthrough["language"] = responseLanguage(locale)
		passthrough["language_instruction"] = "Answer in " + responseLanguage(locale) + "."
	}
	if len(passthrough) == 0 {
		return nil
	}
	return passthrough
}

func responseLanguage(locale string) string {
	switch strings.ToLower(strings.ReplaceAll(locale, "_", "-")) {
	case "zh-hant", "zh-tw", "zh-hk", "zh-mo":
		return "Traditional Chinese"
	case "zh-cn", "zh-hans":
		return "Simplified Chinese"
	case "en", "en-us", "en-gb":
		return "English"
	default:
		return locale
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeSSEError(w http.ResponseWriter, status int, message string) {
	startSSE(w, status)
	escaped, _ := json.Marshal(message)
	_, _ = fmt.Fprintf(w, "data: {\"type\":\"error\",\"error\":%s}\n\n", escaped)
	_, _ = io.WriteString(w, "data: {\"type\":\"end\"}\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func startSSE(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(status)
}

type flushWriter struct {
	http.ResponseWriter
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}

func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	for _, candidate := range allowed {
		candidate = strings.TrimRight(strings.TrimSpace(candidate), "/")
		if candidate == "*" || candidate == origin {
			return true
		}
	}
	return false
}

func allowsWildcard(allowed []string) bool {
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == "*" {
			return true
		}
	}
	return false
}

func requestIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if first := strings.TrimSpace(parts[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	hits   map[string]rateLimitEntry
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(window time.Duration, max int) *rateLimiter {
	return &rateLimiter{
		window: window,
		max:    max,
		hits:   map[string]rateLimitEntry{},
	}
}

func (l *rateLimiter) allow(key string) bool {
	if l.max <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.hits[key]
	if entry.resetAt.IsZero() || now.After(entry.resetAt) {
		entry = rateLimitEntry{resetAt: now.Add(l.window)}
	}
	if entry.count >= l.max {
		l.hits[key] = entry
		return false
	}
	entry.count++
	l.hits[key] = entry
	return true
}
