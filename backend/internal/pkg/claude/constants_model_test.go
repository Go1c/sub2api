package claude

import "testing"

func TestDefaultModelsContainsClaudeFable51(t *testing.T) {
	t.Parallel()

	for _, model := range DefaultModels {
		if model.ID == "claude-fable-5-1" {
			if model.DisplayName != "Claude Fable 5.1" {
				t.Fatalf("display name = %q", model.DisplayName)
			}
			if model.CreatedAt != "2026-09-01T00:00:00Z" {
				t.Fatalf("created at = %q", model.CreatedAt)
			}
			return
		}
	}
	t.Fatal("claude-fable-5-1 missing from DefaultModels")
}
