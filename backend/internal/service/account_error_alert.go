package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	AccountErrorAlertExtraKey      = "error_alert"
	accountErrorAlertMaxKeywords   = 20
	accountErrorAlertMaxKeywordLen = 80
	accountErrorAlertMaxRules      = 10
	accountErrorAlertMaxSendsMax   = 60
)

type AccountErrorAlertSettings struct {
	Enabled  bool                    `json:"enabled"`
	Keywords []string                `json:"keywords,omitempty"`
	Rules    []AccountErrorAlertRule `json:"rules,omitempty"`
}

type AccountErrorAlertRule struct {
	Keyword       string `json:"keyword,omitempty"`
	WindowMinutes int    `json:"window_minutes,omitempty"`
	MinErrorCount int    `json:"min_error_count,omitempty"`
	MaxSends      int    `json:"max_sends,omitempty"`
}

func ParseAccountErrorAlertSettings(extra map[string]any) AccountErrorAlertSettings {
	out := AccountErrorAlertSettings{Enabled: true}
	if extra == nil {
		return out
	}
	raw, ok := extra[AccountErrorAlertExtraKey]
	if !ok || raw == nil {
		return out
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return out
	}
	var parsed struct {
		Enabled  *bool                   `json:"enabled"`
		Keywords []string                `json:"keywords"`
		Rules    []AccountErrorAlertRule `json:"rules"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return out
	}
	if parsed.Enabled != nil {
		out.Enabled = *parsed.Enabled
	}
	out.Keywords = parsed.Keywords
	out.Rules = parsed.Rules
	normalizeAccountErrorAlertSettings(&out)
	return out
}

func normalizeAccountErrorAlertSettings(cfg *AccountErrorAlertSettings) {
	if cfg == nil {
		return
	}
	cfg.Keywords = normalizeAccountErrorAlertKeywords(cfg.Keywords)
	if len(cfg.Rules) > accountErrorAlertMaxRules {
		cfg.Rules = cfg.Rules[:accountErrorAlertMaxRules]
	}
	out := make([]AccountErrorAlertRule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		rule.Keyword = strings.TrimSpace(rule.Keyword)
		if len(rule.Keyword) > accountErrorAlertMaxKeywordLen {
			rule.Keyword = rule.Keyword[:accountErrorAlertMaxKeywordLen]
		}
		if rule.WindowMinutes < 0 {
			rule.WindowMinutes = 0
		}
		if rule.WindowMinutes > 1440 {
			rule.WindowMinutes = 1440
		}
		if rule.MinErrorCount < 0 {
			rule.MinErrorCount = 0
		}
		if rule.MinErrorCount > 100000 {
			rule.MinErrorCount = 100000
		}
		if rule.MaxSends < 0 {
			rule.MaxSends = 0
		}
		if rule.MaxSends > accountErrorAlertMaxSendsMax {
			rule.MaxSends = accountErrorAlertMaxSendsMax
		}
		out = append(out, rule)
	}
	cfg.Rules = out
}

func normalizeAccountErrorAlertKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keywords))
	out := make([]string, 0, len(keywords))
	for _, raw := range keywords {
		kw := strings.TrimSpace(raw)
		if kw == "" {
			continue
		}
		if len(kw) > accountErrorAlertMaxKeywordLen {
			kw = kw[:accountErrorAlertMaxKeywordLen]
		}
		key := strings.ToLower(kw)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, kw)
		if len(out) >= accountErrorAlertMaxKeywords {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func AccountErrorAlertKeywordMatches(keyword, message string, statusCode int) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return true
	}
	if strconv.Itoa(statusCode) == keyword {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(message)), strings.ToLower(keyword))
}

func AccountErrorAlertMessageMatchesKeywords(message string, statusCode int, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	for _, keyword := range keywords {
		if AccountErrorAlertKeywordMatches(keyword, message, statusCode) {
			return true
		}
	}
	return false
}

func accountErrorAlertRuleCoversItem(rule AccountErrorAlertRule, item *OpsAccountErrorAlertCandidate) bool {
	if item == nil {
		return false
	}
	return AccountErrorAlertKeywordMatches(rule.Keyword, item.ErrorMessage, item.StatusCode)
}

func resolveAccountErrorAlertRuleWindow(cfg *OpsAccountErrorAlertConfig, rule AccountErrorAlertRule) int {
	if rule.WindowMinutes > 0 {
		return rule.WindowMinutes
	}
	if cfg != nil && cfg.WindowMinutes > 0 {
		return cfg.WindowMinutes
	}
	return 10
}

func resolveAccountErrorAlertRuleMinCount(cfg *OpsAccountErrorAlertConfig, rule AccountErrorAlertRule) int {
	if rule.MinErrorCount > 0 {
		return rule.MinErrorCount
	}
	if cfg != nil && cfg.MinErrorCount > 0 {
		return cfg.MinErrorCount
	}
	return 1
}

func resolveAccountErrorAlertRuleMaxSends(rule AccountErrorAlertRule) int {
	if rule.MaxSends > 0 {
		return rule.MaxSends
	}
	return 1
}
