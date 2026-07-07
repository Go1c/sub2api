package handler

import "testing"

func TestModelAllowedBySet(t *testing.T) {
	tests := []struct {
		name    string
		allowed map[string]struct{}
		model   string
		want    bool
	}{
		{
			name:    "empty allowlist permits any model",
			allowed: nil,
			model:   "claude-sonnet-4-5",
			want:    true,
		},
		{
			name:    "exact model is allowed",
			allowed: map[string]struct{}{"claude-sonnet-4-5": {}},
			model:   "claude-sonnet-4-5",
			want:    true,
		},
		{
			name:    "unlisted model is rejected",
			allowed: map[string]struct{}{"claude-sonnet-4-5": {}},
			model:   "claude-opus-4-6",
			want:    false,
		},
		{
			name:    "prefixed gemini list model matches unprefixed allowlist entry",
			allowed: map[string]struct{}{"gemini-2.5-pro": {}},
			model:   "models/gemini-2.5-pro",
			want:    true,
		},
		{
			name:    "unprefixed gemini list model matches prefixed allowlist entry",
			allowed: map[string]struct{}{"models/gemini-2.5-pro": {}},
			model:   "gemini-2.5-pro",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modelAllowedBySet(tt.allowed, tt.model); got != tt.want {
				t.Fatalf("modelAllowedBySet(%v, %q) = %v, want %v", tt.allowed, tt.model, got, tt.want)
			}
		})
	}
}
