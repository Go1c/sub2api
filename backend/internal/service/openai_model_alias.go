package service

import (
	"context"
	"errors"
	"strings"
)

// openaiHiddenIngressFallbackModel is the public stand-in for models that
// clients may request but this gateway must not serve as themselves by default.
// Codex Auto-review hardcodes gpt-5.6-luna / codex-auto-review and has no
// client-side override. Groups without an explicit Luna account mapping fall
// back here; accounts that opt in via model_mapping may serve real Luna.
const openaiHiddenIngressFallbackModel = "gpt-5.6-terra"
const openaiHiddenLunaModel = "gpt-5.6-luna"

// RewriteOpenAIHiddenIngressModel rewrites models that must not be served
// (Luna and Codex Auto-review) onto the public fallback. Other names are
// returned unchanged so later alias / channel mapping still apply.
func RewriteOpenAIHiddenIngressModel(model string) string {
	if !isOpenAIHiddenIngressModel(model) {
		return strings.TrimSpace(model)
	}
	return openaiHiddenIngressFallbackModel
}

// ResolveOpenAIHiddenIngressModel keeps Luna when an account in the group has
// opted in via an explicit Luna mapping. Otherwise it falls back to Terra.
// Codex Auto-review is canonicalized to gpt-5.6-luna on the opt-in path.
func ResolveOpenAIHiddenIngressModel(model string, allowRealLuna bool) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" || !isOpenAIHiddenIngressModel(trimmed) {
		return trimmed
	}
	if allowRealLuna {
		return canonicalizeOpenAIHiddenIngressModel(trimmed)
	}
	return openaiHiddenIngressFallbackModel
}

func isOpenAIHiddenIngressModel(model string) bool {
	normalized := openAIHiddenIngressNormalizedName(model)
	if normalized == "" {
		return false
	}
	return isOpenAIHiddenAutoReviewModel(normalized) || strings.Contains(normalized, "gpt-5.6-luna")
}

func canonicalizeOpenAIHiddenIngressModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return trimmed
	}
	normalized := openAIHiddenIngressNormalizedName(trimmed)
	if isOpenAIHiddenAutoReviewModel(normalized) {
		return openaiHiddenLunaModel
	}
	return trimmed
}

func openAIHiddenIngressNormalizedName(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return ""
	}
	normalized := canonicalizeOpenAIModelAliasSpelling(trimmed)
	if normalized == "" {
		normalized = strings.ToLower(lastOpenAIModelSegment(trimmed))
	}
	return normalized
}

func isOpenAIHiddenAutoReviewModel(normalized string) bool {
	return normalized == "codex-auto-review" || strings.HasPrefix(normalized, "codex-auto-review-")
}

func mappingKeyServesOpenAIHiddenLuna(key string) bool {
	normalized := openAIHiddenIngressNormalizedName(key)
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(key))
	}
	return strings.Contains(normalized, "gpt-5.6-luna")
}

func mappingKeyIsOpenAIHiddenAutoReview(key string) bool {
	normalized := openAIHiddenIngressNormalizedName(key)
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(key))
	}
	return isOpenAIHiddenAutoReviewModel(normalized)
}

func accountExplicitlyServesOpenAIHiddenLuna(account *Account) bool {
	if account == nil {
		return false
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return false
	}
	for key, value := range mapping {
		if mappingKeyServesOpenAIHiddenLuna(key) {
			return true
		}
		if mappingKeyIsOpenAIHiddenAutoReview(key) && mappingKeyServesOpenAIHiddenLuna(value) {
			return true
		}
	}
	return false
}

func anyAccountExplicitlyServesOpenAIHiddenLuna(accounts []Account) bool {
	for i := range accounts {
		if accountExplicitlyServesOpenAIHiddenLuna(&accounts[i]) {
			return true
		}
	}
	return false
}

func accountSupportsOpenAISchedulingModel(account *Account, requestedModel string) bool {
	if requestedModel == "" {
		return true
	}
	if isOpenAIHiddenIngressModel(requestedModel) {
		return accountExplicitlyServesOpenAIHiddenLuna(account)
	}
	return account.IsModelSupported(requestedModel)
}

func shouldFallbackFromHiddenOpenAIIngress(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, ErrNoAvailableAccounts) || errors.Is(err, ErrNoAvailableCompactAccounts)
}

func (s *OpenAIGatewayService) ResolveOpenAIHiddenIngressModel(ctx context.Context, groupID *int64, model string) string {
	if s == nil || !isOpenAIHiddenIngressModel(model) {
		return strings.TrimSpace(model)
	}
	return ResolveOpenAIHiddenIngressModel(model, s.groupHasExplicitOpenAIHiddenLunaAccount(ctx, groupID))
}

func (s *OpenAIGatewayService) groupHasExplicitOpenAIHiddenLunaAccount(ctx context.Context, groupID *int64) bool {
	if s == nil {
		return false
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID, PlatformOpenAI)
	if err != nil {
		return false
	}
	return anyAccountExplicitlyServesOpenAIHiddenLuna(accounts)
}

func lastOpenAIModelSegment(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if strings.Contains(model, "/") {
		parts := strings.Split(model, "/")
		model = parts[len(parts)-1]
	}
	return strings.TrimSpace(model)
}

func canonicalizeOpenAIModelAliasSpelling(model string) string {
	model = strings.ToLower(lastOpenAIModelSegment(model))
	if model == "" {
		return ""
	}

	normalized := strings.ReplaceAll(model, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}

	if strings.HasPrefix(normalized, "gpt5") {
		normalized = "gpt-5" + strings.TrimPrefix(normalized, "gpt5")
	}
	if strings.HasPrefix(normalized, "gpt6") {
		normalized = "gpt-6" + strings.TrimPrefix(normalized, "gpt6")
	}
	if !strings.HasPrefix(normalized, "gpt-") && !strings.Contains(normalized, "codex") {
		return ""
	}

	replacements := []struct {
		from string
		to   string
	}{
		{"gpt-5.4mini", "gpt-5.4-mini"},
		{"gpt-5.4nano", "gpt-5.4-nano"},
		{"gpt-5.3-codexspark", "gpt-5.3-codex-spark"},
		{"gpt-5.3codexspark", "gpt-5.3-codex-spark"},
		{"gpt-5.3codex", "gpt-5.3-codex"},
		{"gpt-6astra", "gpt-6-astra"},
	}
	for _, replacement := range replacements {
		normalized = strings.ReplaceAll(normalized, replacement.from, replacement.to)
	}
	return normalized
}

func normalizeKnownOpenAICodexModel(model string) string {
	normalized := canonicalizeOpenAIModelAliasSpelling(model)
	if normalized == "" {
		return ""
	}

	if mapped := getNormalizedCodexModel(normalized); mapped != "" {
		return mapped
	}
	if strings.HasSuffix(normalized, "-openai-compact") {
		if mapped := getNormalizedCodexModel(strings.TrimSuffix(normalized, "-openai-compact")); mapped != "" {
			return mapped
		}
	}

	switch {
	case isOpenAIGPT6AstraModel(normalized):
		return "gpt-6-astra"
	case strings.Contains(normalized, "gpt-5.6-sol"):
		return "gpt-5.6-sol"
	case strings.Contains(normalized, "gpt-5.6-terra"):
		return "gpt-5.6-terra"
	case strings.Contains(normalized, "gpt-5.6-luna"):
		return openaiHiddenLunaModel
	case isOpenAIHiddenAutoReviewModel(normalized):
		return openaiHiddenLunaModel
	case normalized == "gpt-5.6":
		return "gpt-5.6-sol"
	case strings.HasPrefix(normalized, "gpt-5.6-"):
		suffix := strings.TrimPrefix(normalized, "gpt-5.6-")
		if suffix == "max" || isKnownCodexModelSuffix(suffix) {
			return "gpt-5.6-sol"
		}
		return ""
	case strings.Contains(normalized, "gpt-5.5-pro"):
		return "gpt-5.5-pro"
	case strings.Contains(normalized, "gpt-5.5"):
		return "gpt-5.5"
	case strings.Contains(normalized, "gpt-5.4-mini"):
		return "gpt-5.4-mini"
	case strings.Contains(normalized, "gpt-5.4-nano"):
		return "gpt-5.4-nano"
	case strings.Contains(normalized, "gpt-5.4"):
		return "gpt-5.4"
	case strings.Contains(normalized, "gpt-5.2"):
		return "gpt-5.2"
	case strings.Contains(normalized, "gpt-5.3-codex-spark"):
		return "gpt-5.3-codex-spark"
	case strings.Contains(normalized, "gpt-5.3-codex"):
		return "gpt-5.3-codex"
	case strings.Contains(normalized, "gpt-5.3"):
		return "gpt-5.3-codex"
	case strings.Contains(normalized, "codex"):
		return "gpt-5.3-codex"
	case strings.Contains(normalized, "gpt-5"):
		return "gpt-5.4"
	default:
		return ""
	}
}

func isOpenAIGPT6AstraModel(model string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(model)
	if normalized == "gpt-6" || normalized == "gpt-6-astra" {
		return true
	}
	if suffix, ok := strings.CutPrefix(normalized, "gpt-6-astra-"); ok && (suffix == "max" || isKnownCodexModelSuffix(suffix)) {
		return true
	}
	if suffix, ok := strings.CutPrefix(normalized, "gpt-6-"); ok && (suffix == "max" || isKnownCodexModelSuffix(suffix)) {
		return true
	}
	return false
}

// isOpenAIGPT56Model 判断是否 GPT-5.6 系列模型；入参可为原始模型名
// （含大小写/路径/后缀变体）或已归一化的基名，两者均能正确识别。
func isOpenAIGPT56Model(model string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(model)
	if normalized == "gpt-5.6" {
		return true
	}
	if suffix, ok := strings.CutPrefix(normalized, "gpt-5.6-"); ok && (suffix == "max" || isKnownCodexModelSuffix(suffix)) {
		return true
	}
	for _, prefix := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		if normalized == prefix || strings.HasPrefix(normalized, prefix+"-") {
			return true
		}
	}
	return false
}

func appendUsageBillingModelCandidate(candidates []string, seen map[string]struct{}, model string) []string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return candidates
	}
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}

	add(trimmed)
	if canonical := canonicalizeOpenAIModelAliasSpelling(trimmed); canonical != "" {
		add(canonical)
	}
	if normalized := normalizeKnownOpenAICodexModel(trimmed); normalized != "" {
		add(normalized)
	}
	return candidates
}

func usageBillingModelCandidates(primary string, alternates ...string) []string {
	seen := make(map[string]struct{}, 1+len(alternates))
	candidates := appendUsageBillingModelCandidate(nil, seen, primary)
	for _, alternate := range alternates {
		candidates = appendUsageBillingModelCandidate(candidates, seen, alternate)
	}
	return candidates
}

func firstUsageBillingModel(candidates []string) string {
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
