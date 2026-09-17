package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	AccountPoolAutoInspectMinIntervalMinutes  = 1
	AccountPoolAutoInspectMaxIntervalMinutes  = 1440
	AccountPoolAutoInspectDefaultInterval     = 5
	AccountPoolAutoInspectDefaultThreshold    = 50
	AccountPoolAutoInspectDefaultMinSamples   = 4
	AccountPoolAutoInspectMaxGroups           = 20
	AccountPoolAutoInspectMaxModels           = 50
	AccountPoolAutoInspectMaxModelLen         = 80
	accountPoolAutoInspectLeaderLockKey       = "ops:pool_auto_inspect:leader"
	accountPoolAutoInspectOAuth401CooldownKey = "ops:pool_auto_inspect:oauth401:"
)

// AccountPoolAutoInspectConfig is the admin-facing pool health automation.
// It scans request-health bars on a timer, joins configured groups and strips
// configured whitelist models when every sampled IP is below the success-rate
// threshold, and can Telegram-notify OAuth 401 scheduling stops.
type AccountPoolAutoInspectConfig struct {
	Enabled bool `json:"enabled"`

	IntervalMinutes         int      `json:"interval_minutes"`
	SuccessRateThreshold    int      `json:"success_rate_threshold"`
	MinSamples              int      `json:"min_samples"`
	AddGroupIDs             []int64  `json:"add_group_ids"`
	RemoveModels            []string `json:"remove_models"`
	NotifyOAuth401          bool     `json:"notify_oauth_401"`
	OAuth401CooldownMinutes int      `json:"oauth_401_cooldown_minutes"`

	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   string `json:"telegram_chat_id"`
}

type AccountPoolAutoInspectStatus struct {
	AccountPoolAutoInspectConfig
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	LastResult    string     `json:"last_result,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	TelegramReady bool       `json:"telegram_ready"`
}

type poolAutoInspectHealthVerdict struct {
	Unhealthy bool
	Rate      float64
	Samples   int
	OK        int
	Fail      int
}

type poolAutoInspectRemediation struct {
	GroupIDs       []int64
	GroupsChanged  bool
	AddedGroupIDs  []int64
	Mapping        map[string]any
	MappingChanged bool
	RemovedModels  []string
	HasWork        bool
}

func defaultAccountPoolAutoInspectConfig() *AccountPoolAutoInspectConfig {
	return &AccountPoolAutoInspectConfig{
		Enabled:                 false,
		IntervalMinutes:         AccountPoolAutoInspectDefaultInterval,
		SuccessRateThreshold:    AccountPoolAutoInspectDefaultThreshold,
		MinSamples:              AccountPoolAutoInspectDefaultMinSamples,
		AddGroupIDs:             []int64{},
		RemoveModels:            []string{},
		NotifyOAuth401:          false,
		OAuth401CooldownMinutes: 60,
	}
}

func normalizeAccountPoolAutoInspectConfig(cfg *AccountPoolAutoInspectConfig) {
	if cfg == nil {
		return
	}
	defaults := defaultAccountPoolAutoInspectConfig()
	if cfg.IntervalMinutes < AccountPoolAutoInspectMinIntervalMinutes || cfg.IntervalMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		cfg.IntervalMinutes = defaults.IntervalMinutes
	}
	if cfg.SuccessRateThreshold < 1 || cfg.SuccessRateThreshold > 100 {
		cfg.SuccessRateThreshold = defaults.SuccessRateThreshold
	}
	if cfg.MinSamples < 1 || cfg.MinSamples > RequestHealthMaxEvents {
		cfg.MinSamples = defaults.MinSamples
	}
	if cfg.OAuth401CooldownMinutes < 1 || cfg.OAuth401CooldownMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		cfg.OAuth401CooldownMinutes = defaults.OAuth401CooldownMinutes
	}
	cfg.AddGroupIDs = normalizePositiveIDs(cfg.AddGroupIDs, AccountPoolAutoInspectMaxGroups)
	cfg.RemoveModels = normalizeAccountPoolAutoInspectModels(cfg.RemoveModels)
	cfg.TelegramBotToken = strings.TrimSpace(cfg.TelegramBotToken)
	cfg.TelegramChatID = strings.TrimSpace(cfg.TelegramChatID)
}

func validateAccountPoolAutoInspectConfig(cfg *AccountPoolAutoInspectConfig) error {
	if cfg == nil {
		return errors.New("invalid config")
	}
	if cfg.IntervalMinutes < AccountPoolAutoInspectMinIntervalMinutes || cfg.IntervalMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		return fmt.Errorf("interval_minutes must be %d-%d", AccountPoolAutoInspectMinIntervalMinutes, AccountPoolAutoInspectMaxIntervalMinutes)
	}
	if cfg.SuccessRateThreshold < 1 || cfg.SuccessRateThreshold > 100 {
		return errors.New("success_rate_threshold must be 1-100")
	}
	if cfg.MinSamples < 1 || cfg.MinSamples > RequestHealthMaxEvents {
		return fmt.Errorf("min_samples must be 1-%d", RequestHealthMaxEvents)
	}
	if cfg.OAuth401CooldownMinutes < 1 || cfg.OAuth401CooldownMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		return fmt.Errorf("oauth_401_cooldown_minutes must be 1-%d", AccountPoolAutoInspectMaxIntervalMinutes)
	}
	return nil
}

func normalizePositiveIDs(ids []int64, limit int) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return []int64{}
	}
	return out
}

func normalizeAccountPoolAutoInspectModels(models []string) []string {
	if len(models) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, raw := range models {
		model := strings.TrimSpace(raw)
		if model == "" {
			continue
		}
		if len(model) > AccountPoolAutoInspectMaxModelLen {
			model = model[:AccountPoolAutoInspectMaxModelLen]
		}
		key := strings.ToLower(model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
		if len(out) >= AccountPoolAutoInspectMaxModels {
			break
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func evaluateAccountPoolAutoInspectHealth(dto AccountRequestHealthDTO, thresholdPercent, minSamples int) poolAutoInspectHealthVerdict {
	if thresholdPercent < 1 {
		thresholdPercent = AccountPoolAutoInspectDefaultThreshold
	}
	if minSamples < 1 {
		minSamples = AccountPoolAutoInspectDefaultMinSamples
	}
	threshold := float64(thresholdPercent) / 100

	var okCount, failCount int
	sampledLines := 0
	healthyLine := false
	for _, line := range dto.Lines {
		lineOK, lineFail := countRequestHealthOutcomes(line.Outcomes)
		lineSamples := lineOK + lineFail
		if lineSamples == 0 {
			continue
		}
		sampledLines++
		okCount += lineOK
		failCount += lineFail
		if float64(lineOK)/float64(lineSamples) >= threshold {
			healthyLine = true
		}
	}

	verdict := poolAutoInspectHealthVerdict{
		OK:      okCount,
		Fail:    failCount,
		Samples: okCount + failCount,
	}
	if verdict.Samples == 0 {
		return verdict
	}
	verdict.Rate = float64(okCount) / float64(verdict.Samples)
	if verdict.Samples < minSamples || sampledLines == 0 {
		return verdict
	}
	// Account is unhealthy only when overall success is below the threshold
	// and every sampled IP/line is also below it ("每个 IP 都打不进去").
	verdict.Unhealthy = verdict.Rate < threshold && !healthyLine
	return verdict
}

func countRequestHealthOutcomes(outcomes []RequestHealthOutcomeDTO) (okCount, failCount int) {
	for _, item := range outcomes {
		switch item.Slot {
		case RequestHealthSlotOK:
			okCount++
		case RequestHealthSlotFail:
			failCount++
		}
	}
	return okCount, failCount
}

func planAccountPoolAutoInspectRemediation(account Account, cfg *AccountPoolAutoInspectConfig) poolAutoInspectRemediation {
	return planAccountPoolAutoInspectRemediationWithGroups(account, cfg, nil, false)
}

func planAccountPoolAutoInspectRemediationWithGroups(account Account, cfg *AccountPoolAutoInspectConfig, groups map[int64]Group, haveDirectory bool) poolAutoInspectRemediation {
	plan := poolAutoInspectRemediation{}
	if cfg == nil {
		return plan
	}
	addIDs := compatibleAddGroupIDs(account, cfg.AddGroupIDs, groups, haveDirectory)
	if len(addIDs) > 0 {
		merged := mergeUniqueInt64(account.GroupIDs, addIDs)
		added := differenceInt64(merged, account.GroupIDs)
		plan.GroupIDs = merged
		plan.AddedGroupIDs = added
		plan.GroupsChanged = len(added) > 0
	}
	if len(cfg.RemoveModels) > 0 {
		mapping, ok := accountModelMappingRaw(account)
		if ok {
			next, removed := removeModelsFromMapping(mapping, cfg.RemoveModels)
			if len(removed) > 0 {
				plan.Mapping = next
				plan.RemovedModels = removed
				plan.MappingChanged = true
			}
		}
	}
	plan.HasWork = plan.GroupsChanged || plan.MappingChanged
	return plan
}

func accountModelMappingRaw(account Account) (map[string]any, bool) {
	if len(account.Credentials) == 0 {
		return nil, false
	}
	raw, ok := account.Credentials["model_mapping"].(map[string]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out, true
}

func removeModelsFromMapping(mapping map[string]any, models []string) (map[string]any, []string) {
	if len(mapping) == 0 || len(models) == 0 {
		return mapping, nil
	}
	drop := make(map[string]string, len(models))
	for _, model := range models {
		key := strings.ToLower(strings.TrimSpace(model))
		if key == "" {
			continue
		}
		drop[key] = strings.TrimSpace(model)
	}
	if len(drop) == 0 {
		return mapping, nil
	}
	removedSeen := map[string]struct{}{}
	removed := make([]string, 0)
	next := make(map[string]any, len(mapping))
	for key, value := range mapping {
		valueText, _ := value.(string)
		if _, ok := drop[strings.ToLower(strings.TrimSpace(key))]; ok {
			if _, exists := removedSeen[key]; !exists {
				removedSeen[key] = struct{}{}
				removed = append(removed, key)
			}
			continue
		}
		if valueText != "" {
			if _, ok := drop[strings.ToLower(strings.TrimSpace(valueText))]; ok {
				if _, exists := removedSeen[key]; !exists {
					removedSeen[key] = struct{}{}
					removed = append(removed, key)
				}
				continue
			}
		}
		next[key] = value
	}
	return next, removed
}

func compatibleAddGroupIDs(account Account, ids []int64, groups map[int64]Group, haveDirectory bool) []int64 {
	if len(ids) == 0 {
		return nil
	}
	if !haveDirectory {
		return ids
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		group, ok := groups[id]
		if !ok {
			continue
		}
		if groupAcceptsAccountPlatform(group.Platform, account.Platform) {
			out = append(out, id)
		}
	}
	return out
}

func groupAcceptsAccountPlatform(groupPlatform, accountPlatform string) bool {
	groupPlatform = strings.ToLower(strings.TrimSpace(groupPlatform))
	accountPlatform = strings.ToLower(strings.TrimSpace(accountPlatform))
	if groupPlatform == "" || groupPlatform == PlatformComposite {
		return true
	}
	return groupPlatform == accountPlatform
}

func mergeUniqueInt64(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range append(append([]int64{}, a...), b...) {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func differenceInt64(have, existing []int64) []int64 {
	seen := make(map[int64]struct{}, len(existing))
	for _, id := range existing {
		seen[id] = struct{}{}
	}
	out := make([]int64, 0)
	for _, id := range have {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, id)
	}
	return out
}

func accountStoppedByOAuth401(account Account, now time.Time) (bool, string) {
	reason := strings.TrimSpace(account.TempUnschedulableReason)
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now) && looksLikeOAuth401(reason) {
		return true, reason
	}
	if account.Status == StatusError && looksLikeOAuth401(account.ErrorMessage) {
		return true, strings.TrimSpace(account.ErrorMessage)
	}
	if !account.Schedulable && looksLikeOAuth401(account.ErrorMessage) {
		return true, strings.TrimSpace(account.ErrorMessage)
	}
	return false, ""
}

func looksLikeOAuth401(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	if strings.Contains(text, "oauth 401") || strings.Contains(text, "token revoked (401)") {
		return true
	}
	if !strings.Contains(text, "401") {
		return false
	}
	return strings.Contains(text, "authentication failed") ||
		strings.Contains(text, "unauthorized") ||
		strings.Contains(text, "invalid or expired credentials") ||
		strings.Contains(text, "refresh_token")
}

func resolveAccountPoolAutoInspectTelegram(cfg *AccountPoolAutoInspectConfig, fallback *OpsAccountErrorAlertConfig) (token, chatID string) {
	if cfg != nil {
		token = strings.TrimSpace(cfg.TelegramBotToken)
		chatID = strings.TrimSpace(cfg.TelegramChatID)
	}
	if token != "" && chatID != "" {
		return token, chatID
	}
	if fallback != nil {
		if token == "" {
			token = strings.TrimSpace(fallback.TelegramBotToken)
		}
		if chatID == "" {
			chatID = strings.TrimSpace(fallback.TelegramChatID)
		}
	}
	return token, chatID
}

func cloneAccountCredentials(credentials map[string]any) map[string]any {
	if credentials == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(credentials))
	for k, v := range credentials {
		out[k] = v
	}
	return out
}

func marshalAccountPoolAutoInspectConfig(cfg *AccountPoolAutoInspectConfig) (string, error) {
	if cfg == nil {
		cfg = defaultAccountPoolAutoInspectConfig()
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseAccountPoolAutoInspectConfig(raw string) *AccountPoolAutoInspectConfig {
	cfg := defaultAccountPoolAutoInspectConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return defaultAccountPoolAutoInspectConfig()
	}
	normalizeAccountPoolAutoInspectConfig(cfg)
	return cfg
}
