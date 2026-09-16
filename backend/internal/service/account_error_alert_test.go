package service

import "testing"

func TestParseAccountErrorAlertSettings_DefaultsEnabled(t *testing.T) {
	got := ParseAccountErrorAlertSettings(nil)
	if !got.Enabled {
		t.Fatal("missing extra should enable monitoring")
	}
	got = ParseAccountErrorAlertSettings(map[string]any{})
	if !got.Enabled {
		t.Fatal("empty extra should enable monitoring")
	}
}

func TestParseAccountErrorAlertSettings_ExplicitOff(t *testing.T) {
	got := ParseAccountErrorAlertSettings(map[string]any{
		AccountErrorAlertExtraKey: map[string]any{"enabled": false},
	})
	if got.Enabled {
		t.Fatal("enabled=false should disable monitoring")
	}
}

func TestParseAccountErrorAlertSettings_KeywordsAndRules(t *testing.T) {
	got := ParseAccountErrorAlertSettings(map[string]any{
		AccountErrorAlertExtraKey: map[string]any{
			"enabled":  true,
			"keywords": []any{"overloaded", " 503 ", "overloaded", ""},
			"rules": []any{
				map[string]any{
					"keyword":         "overloaded",
					"window_minutes":  10,
					"min_error_count": 3,
					"max_sends":       1,
				},
			},
		},
	})
	if !got.Enabled {
		t.Fatal("expected enabled")
	}
	if len(got.Keywords) != 2 || got.Keywords[0] != "overloaded" || got.Keywords[1] != "503" {
		t.Fatalf("keywords = %#v", got.Keywords)
	}
	if len(got.Rules) != 1 || got.Rules[0].Keyword != "overloaded" || got.Rules[0].MaxSends != 1 {
		t.Fatalf("rules = %#v", got.Rules)
	}
}

func TestAccountErrorAlertKeywordMatches(t *testing.T) {
	msg := "Our servers are currently overloaded. Please try again later."
	if !AccountErrorAlertKeywordMatches("overloaded", msg, 503) {
		t.Fatal("expected substring match")
	}
	if !AccountErrorAlertKeywordMatches("503", "upstream failed", 503) {
		t.Fatal("expected status code match")
	}
	if AccountErrorAlertKeywordMatches("429", msg, 503) {
		t.Fatal("did not expect unrelated keyword")
	}
	if !AccountErrorAlertMessageMatchesKeywords(msg, 503, nil) {
		t.Fatal("empty keywords should match all")
	}
	if AccountErrorAlertMessageMatchesKeywords(msg, 503, []string{"rate limit"}) {
		t.Fatal("unrelated keywords should not match")
	}
}

func TestAccountErrorAlertRuleCoversItem_EmptyKeywordMatchesAll(t *testing.T) {
	item := &OpsAccountErrorAlertCandidate{StatusCode: 503, ErrorMessage: "overloaded"}
	if !accountErrorAlertRuleCoversItem(AccountErrorAlertRule{}, item) {
		t.Fatal("empty rule keyword should match any error")
	}
	if accountErrorAlertRuleCoversItem(AccountErrorAlertRule{Keyword: "429"}, item) {
		t.Fatal("unrelated rule keyword should not match")
	}
}
