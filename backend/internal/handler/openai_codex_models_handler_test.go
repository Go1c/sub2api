package handler

import (
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestShouldFailoverCodexModelsAccount(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "missing token",
			err:  infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_MODELS_TOKEN_MISSING", "missing token"),
			want: true,
		},
		{
			name: "upstream not configured",
			err:  infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_MODELS_UPSTREAM_NOT_CONFIGURED", "missing upstream"),
			want: true,
		},
		{
			name: "upstream request failed",
			err:  infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_MODELS_UPSTREAM_FAILED", "request failed"),
			want: false,
		},
		{
			name: "nil",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldFailoverCodexModelsAccount(tt.err); got != tt.want {
				t.Fatalf("shouldFailoverCodexModelsAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}
