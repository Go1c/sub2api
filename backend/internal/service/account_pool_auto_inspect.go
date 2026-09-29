package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
	"unicode"
)

const (
	AccountPoolAutoInspectMinIntervalMinutes = 1
	AccountPoolAutoInspectMaxIntervalMinutes = 1440
	AccountPoolAutoInspectDefaultInterval    = 10
	AccountPoolAutoInspectDefaultJitter      = 60
	AccountPoolAutoInspectDefaultPause       = 1
	AccountPoolAutoInspectMaxModelLen        = 80
	AccountPoolAutoInspectMaxQuestionLen     = 2000
	AccountPoolAutoInspectMaxAnswerLen       = 200
	accountPoolAutoInspectLeaderLockKey      = "ops:pool_auto_inspect:leader"
	accountPoolAutoInspectDueKeyPrefix       = "ops:pool_auto_inspect:iq_due:"

	accountPoolAutoInspectDefaultModel = "gpt-6-astra"
	// Same candy quiz as channel-monitor IQ on dev and Turn-State probe.
	accountPoolAutoInspectDefaultQuestion = "黑色袋子中有苹果味、桃子味、西瓜味糖果;每种分为圆形和五角星形，可用手感区分形状。圆形依次有7、9、8颗;五角星形依次有7、6、4颗。事先决定摸出的数量，最少取多少颗，才能保证拿到不同形状的苹果味和桃子味糖果?"
	accountPoolAutoInspectDefaultAnswer   = "21"
	// accountPoolAutoInspectCheckedAtExtra records that this account has
	// completed one quiz. A first import has no such mark.
	accountPoolAutoInspectCheckedAtExtra = "pool_auto_inspect_checked_at"
	accountPoolIQFirstImportError        = "pool auto inspect: first import answered incorrectly"

	AccountPoolIQResultCorrect   = "correct"
	AccountPoolIQResultIncorrect = "incorrect"
	AccountPoolIQResultUntestable = "untestable"

	accountPoolAutoInspectLogLimit = 100

	AccountPoolIQLogMoved    = "moved"
	AccountPoolIQLogDisabled = "disabled"
)

// AccountPoolAutoInspectConfig is the admin IQ-group strategy.
// Each due account is asked one question on the configured model. A correct
// answer moves it onto CorrectGroupID; an incorrect answer moves it onto
// IncorrectGroupID. A real group change pauses scheduling for PauseMinutes.
type AccountPoolAutoInspectConfig struct {
	Enabled bool `json:"enabled"`

	IntervalMinutes int    `json:"interval_minutes"`
	JitterSeconds   int    `json:"jitter_seconds"`
	Model           string `json:"model"`
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	FuzzyMatch      bool   `json:"fuzzy_match"`
	CorrectGroupID   int64 `json:"correct_group_id"`
	IncorrectGroupID int64 `json:"incorrect_group_id"`
	PauseMinutes     int   `json:"pause_minutes"`
	// DisableFirstImportOnIncorrect marks a first-import account error and
	// unschedulable when its first completed quiz is wrong.
	DisableFirstImportOnIncorrect bool `json:"disable_first_import_on_incorrect"`
}

type AccountPoolAutoInspectStatus struct {
	AccountPoolAutoInspectConfig
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	LastResult string     `json:"last_result,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
}

// AccountPoolAutoInspectLogEntry is one visible change: a group switch or a
// first-import disable. Newest entries are stored first.
type AccountPoolAutoInspectLogEntry struct {
	At             time.Time `json:"at"`
	AccountID      int64     `json:"account_id"`
	AccountName    string    `json:"account_name"`
	Action         string    `json:"action"`
	FromGroupID    int64     `json:"from_group_id,omitempty"`
	FromGroupName  string    `json:"from_group_name,omitempty"`
	ToGroupID      int64     `json:"to_group_id,omitempty"`
	ToGroupName    string    `json:"to_group_name,omitempty"`
	Result         string    `json:"result,omitempty"`
}

type accountPoolAutoInspectLogFile struct {
	Entries []AccountPoolAutoInspectLogEntry `json:"entries"`
}

// DefaultAccountPoolAutoInspectConfig is the admin form default.
func DefaultAccountPoolAutoInspectConfig() *AccountPoolAutoInspectConfig {
	return defaultAccountPoolAutoInspectConfig()
}

func defaultAccountPoolAutoInspectConfig() *AccountPoolAutoInspectConfig {
	return &AccountPoolAutoInspectConfig{
		Enabled:          false,
		IntervalMinutes:  AccountPoolAutoInspectDefaultInterval,
		JitterSeconds:    AccountPoolAutoInspectDefaultJitter,
		Model:            accountPoolAutoInspectDefaultModel,
		Question:         accountPoolAutoInspectDefaultQuestion,
		Answer:           accountPoolAutoInspectDefaultAnswer,
		FuzzyMatch:       true,
		CorrectGroupID:   0,
		IncorrectGroupID: 0,
		PauseMinutes:     AccountPoolAutoInspectDefaultPause,
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
	maxJitter := accountPoolAutoInspectMaxJitterSeconds(cfg.IntervalMinutes)
	if cfg.JitterSeconds < 0 || cfg.JitterSeconds > maxJitter {
		cfg.JitterSeconds = defaults.JitterSeconds
		if cfg.JitterSeconds > maxJitter {
			cfg.JitterSeconds = maxJitter
		}
	}
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		cfg.Model = defaults.Model
	}
	if len(cfg.Model) > AccountPoolAutoInspectMaxModelLen {
		cfg.Model = cfg.Model[:AccountPoolAutoInspectMaxModelLen]
	}
	cfg.Question = strings.TrimSpace(cfg.Question)
	if cfg.Question == "" {
		cfg.Question = defaults.Question
	}
	if len(cfg.Question) > AccountPoolAutoInspectMaxQuestionLen {
		cfg.Question = cfg.Question[:AccountPoolAutoInspectMaxQuestionLen]
	}
	cfg.Answer = strings.TrimSpace(cfg.Answer)
	if cfg.Answer == "" {
		cfg.Answer = defaults.Answer
	}
	if len(cfg.Answer) > AccountPoolAutoInspectMaxAnswerLen {
		cfg.Answer = cfg.Answer[:AccountPoolAutoInspectMaxAnswerLen]
	}
	if cfg.CorrectGroupID < 0 {
		cfg.CorrectGroupID = 0
	}
	if cfg.IncorrectGroupID < 0 {
		cfg.IncorrectGroupID = 0
	}
	if cfg.PauseMinutes < 1 || cfg.PauseMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		cfg.PauseMinutes = defaults.PauseMinutes
	}
}

func validateAccountPoolAutoInspectConfig(cfg *AccountPoolAutoInspectConfig) error {
	if cfg == nil {
		return errors.New("invalid config")
	}
	if cfg.IntervalMinutes < AccountPoolAutoInspectMinIntervalMinutes || cfg.IntervalMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		return fmt.Errorf("interval_minutes must be %d-%d", AccountPoolAutoInspectMinIntervalMinutes, AccountPoolAutoInspectMaxIntervalMinutes)
	}
	maxJitter := accountPoolAutoInspectMaxJitterSeconds(cfg.IntervalMinutes)
	if cfg.JitterSeconds < 0 || cfg.JitterSeconds > maxJitter {
		return fmt.Errorf("jitter_seconds must be 0-%d", maxJitter)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return errors.New("model is required")
	}
	if strings.TrimSpace(cfg.Question) == "" || strings.TrimSpace(cfg.Answer) == "" {
		return errors.New("question and answer are required")
	}
	if cfg.Enabled && (cfg.CorrectGroupID <= 0 || cfg.IncorrectGroupID <= 0) {
		return errors.New("correct_group_id and incorrect_group_id are required when enabled")
	}
	if cfg.CorrectGroupID > 0 && cfg.CorrectGroupID == cfg.IncorrectGroupID {
		return errors.New("correct_group_id and incorrect_group_id must differ")
	}
	if cfg.PauseMinutes < 1 || cfg.PauseMinutes > AccountPoolAutoInspectMaxIntervalMinutes {
		return fmt.Errorf("pause_minutes must be 1-%d", AccountPoolAutoInspectMaxIntervalMinutes)
	}
	return nil
}

// accountPoolAutoInspectMaxJitterSeconds keeps the realized interval at least
// one minute: jitter cannot exceed half the base interval.
func accountPoolAutoInspectMaxJitterSeconds(intervalMinutes int) int {
	if intervalMinutes < 1 {
		intervalMinutes = 1
	}
	maxJitter := intervalMinutes * 60 / 2
	floor := intervalMinutes*60 - 60
	if maxJitter > floor {
		maxJitter = floor
	}
	if maxJitter < 0 {
		return 0
	}
	return maxJitter
}

// accountPoolAutoInspectNextDelay is base interval ± a uniform jitter in seconds.
// The result is at least one minute.
func accountPoolAutoInspectNextDelay(intervalMinutes, jitterSeconds int) time.Duration {
	base := time.Duration(intervalMinutes) * time.Minute
	if base < time.Minute {
		base = time.Minute
	}
	jitter := time.Duration(jitterSeconds) * time.Second
	if jitter <= 0 {
		return base
	}
	offset := time.Duration(rand.Int64N(int64(2*jitter)+1)) - jitter
	delay := base + offset
	if delay < time.Minute {
		return time.Minute
	}
	return delay
}

func accountPoolIQAnswerMatches(got, want string, fuzzy bool) bool {
	want = normalizeAccountPoolIQAnswer(want)
	got = normalizeAccountPoolIQAnswer(got)
	if want == "" || got == "" {
		return false
	}
	if fuzzy {
		return strings.Contains(got, want)
	}
	return got == want
}

func normalizeAccountPoolIQAnswer(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if r == ',' || r == '，' || r == '.' || r == '。' || r == ':' || r == '：' || r == ';' || r == '；' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// planAccountPoolIQGroups replaces one IQ group with the other and keeps every
// unrelated group. Already being on the target group is not a change.
func planAccountPoolIQGroups(current []int64, correctGroupID, incorrectGroupID, targetGroupID int64) (next []int64, changed bool) {
	if targetGroupID <= 0 {
		return append([]int64(nil), current...), false
	}
	seen := map[int64]struct{}{}
	next = make([]int64, 0, len(current)+1)
	for _, id := range current {
		if id <= 0 {
			continue
		}
		if id == correctGroupID || id == incorrectGroupID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		next = append(next, id)
	}
	if _, ok := seen[targetGroupID]; !ok {
		next = append(next, targetGroupID)
	}
	changed = !samePositiveInt64Set(current, next)
	return next, changed
}

// samePositiveInt64Set ignores non-positive ids. It is separate from
// sameInt64Set in admin_user.go, which compares every value including zeros.
func samePositiveInt64Set(a, b []int64) bool {
	left := map[int64]struct{}{}
	for _, id := range a {
		if id > 0 {
			left[id] = struct{}{}
		}
	}
	right := map[int64]struct{}{}
	for _, id := range b {
		if id > 0 {
			right[id] = struct{}{}
		}
	}
	if len(left) != len(right) {
		return false
	}
	for id := range left {
		if _, ok := right[id]; !ok {
			return false
		}
	}
	return true
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

func accountPoolAutoInspectReady(cfg *AccountPoolAutoInspectConfig) bool {
	if cfg == nil || !cfg.Enabled {
		return false
	}
	return cfg.CorrectGroupID > 0 && cfg.IncorrectGroupID > 0 && cfg.CorrectGroupID != cfg.IncorrectGroupID
}

func accountPoolIQGroupName(groups map[int64]Group, id int64) string {
	if groups == nil || id <= 0 {
		return ""
	}
	return strings.TrimSpace(groups[id].Name)
}

// accountPoolIQSourceGroup is the IQ group the account is leaving.
// An account on neither IQ group has no source name.
func accountPoolIQSourceGroup(current []int64, correctGroupID, incorrectGroupID, target int64) int64 {
	other := incorrectGroupID
	if target == incorrectGroupID {
		other = correctGroupID
	}
	for _, id := range current {
		if id == other && other > 0 && other != target {
			return other
		}
	}
	return 0
}

func trimAccountPoolIQLogName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) <= 120 {
		return name
	}
	return name[:120]
}

func parseAccountPoolAutoInspectLog(raw string) []AccountPoolAutoInspectLogEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var file accountPoolAutoInspectLogFile
	if err := json.Unmarshal([]byte(raw), &file); err != nil {
		return nil
	}
	if len(file.Entries) > accountPoolAutoInspectLogLimit {
		return file.Entries[:accountPoolAutoInspectLogLimit]
	}
	return file.Entries
}

func marshalAccountPoolAutoInspectLog(entries []AccountPoolAutoInspectLogEntry) (string, error) {
	if len(entries) > accountPoolAutoInspectLogLimit {
		entries = entries[:accountPoolAutoInspectLogLimit]
	}
	raw, err := json.Marshal(accountPoolAutoInspectLogFile{Entries: entries})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func groupAcceptsAccountPlatform(groupPlatform, accountPlatform string) bool {
	groupPlatform = strings.ToLower(strings.TrimSpace(groupPlatform))
	accountPlatform = strings.ToLower(strings.TrimSpace(accountPlatform))
	if groupPlatform == "" || groupPlatform == PlatformComposite {
		return true
	}
	return groupPlatform == accountPlatform
}
