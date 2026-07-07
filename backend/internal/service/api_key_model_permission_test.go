package service

import "testing"

func TestAPIKeyAllowsModel(t *testing.T) {
	tests := []struct {
		name          string
		allowedModels []string
		model         string
		want          bool
	}{
		{
			name:          "empty allowlist permits any model",
			allowedModels: nil,
			model:         "claude-sonnet-4-5",
			want:          true,
		},
		{
			name:          "exact model match is allowed",
			allowedModels: []string{"claude-sonnet-4-5", "claude-opus-4-6"},
			model:         "claude-sonnet-4-5",
			want:          true,
		},
		{
			name:          "unlisted model is rejected",
			allowedModels: []string{"claude-sonnet-4-5"},
			model:         "claude-opus-4-6",
			want:          false,
		},
		{
			name:          "blank requested model is not allowed when allowlist is enabled",
			allowedModels: []string{"claude-sonnet-4-5"},
			model:         "",
			want:          false,
		},
		{
			name:          "allowlist ignores surrounding whitespace",
			allowedModels: []string{" claude-sonnet-4-5 "},
			model:         "claude-sonnet-4-5",
			want:          true,
		},
		{
			name:          "gemini models prefix in request matches unprefixed allowlist entry",
			allowedModels: []string{"gemini-2.5-pro"},
			model:         "models/gemini-2.5-pro",
			want:          true,
		},
		{
			name:          "gemini models prefix in allowlist matches unprefixed request",
			allowedModels: []string{"models/gemini-2.5-pro"},
			model:         "gemini-2.5-pro",
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{AllowedModels: tt.allowedModels}
			if got := key.AllowsModel(tt.model); got != tt.want {
				t.Fatalf("AllowsModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
