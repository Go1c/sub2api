package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DocsGPTAPIBaseURL   string
	DocsGPTAgentAPIKey  string
	AllowedOrigins      []string
	SupportEmail        string
	SupportURL          string
	OfficialContactText string
	WidgetTitle         string
	WelcomeMessage      string
	RateLimitWindow     time.Duration
	RateLimitMax        int
}

func loadConfigFromEnv() (Config, error) {
	cfg := Config{
		DocsGPTAPIBaseURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("DOCSGPT_API_BASE_URL")), "/"),
		DocsGPTAgentAPIKey:  strings.TrimSpace(os.Getenv("DOCSGPT_AGENT_API_KEY")),
		AllowedOrigins:      splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		SupportEmail:        strings.TrimSpace(os.Getenv("SUPPORT_EMAIL")),
		SupportURL:          strings.TrimSpace(os.Getenv("SUPPORT_URL")),
		OfficialContactText: strings.TrimSpace(os.Getenv("OFFICIAL_CONTACT_TEXT")),
		WidgetTitle:         strings.TrimSpace(os.Getenv("WIDGET_TITLE")),
		WelcomeMessage:      strings.TrimSpace(os.Getenv("WELCOME_MESSAGE")),
		RateLimitWindow:     time.Duration(getenvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		RateLimitMax:        getenvInt("RATE_LIMIT_MAX_REQUESTS", 20),
	}

	if cfg.DocsGPTAPIBaseURL == "" {
		return Config{}, errors.New("DOCSGPT_API_BASE_URL is required")
	}
	if cfg.DocsGPTAgentAPIKey == "" {
		return Config{}, errors.New("DOCSGPT_AGENT_API_KEY is required")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, errors.New("ALLOWED_ORIGINS is required")
	}
	return cfg, nil
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, strings.TrimRight(part, "/"))
		}
	}
	return out
}
