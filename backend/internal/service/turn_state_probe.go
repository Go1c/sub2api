package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	TurnStateProbeExtraKey = "turn_state_probe"

	turnStateProbeDefaultModel          = "gpt-6-astra"
	turnStateProbeDefaultMinLength      = 160
	turnStateProbeDefaultRecheckMinutes = 20
	turnStateProbeDefaultRPM            = 6
	turnStateProbeDefaultSessionMinutes = 5
	turnStateProbeDefaultRegion         = "Random"
	turnStateProbeDefaultQuestion       = "黑色袋子中有苹果味、桃子味、西瓜味糖果;每种分为圆形和五角星形，可用手感区分形状。圆形依次有7、9、8颗;五角星形依次有7、6、4颗。事先决定摸出的数量，最少取多少颗，才能保证拿到不同形状的苹果味和桃子味糖果?"
	turnStateProbeDefaultAnswer         = "21"

	turnStateProbeMinLength         = 1
	turnStateProbeMaxLength         = 4096
	turnStateProbeMinRecheckMinutes = 5
	turnStateProbeMaxRecheckMinutes = 1440
	turnStateProbeMinRPM            = 1
	turnStateProbeMaxRPM            = 60
	turnStateProbeMinSessionMinutes = 1
	turnStateProbeMaxSessionMinutes = 60
)

var (
	ErrTurnStateProbeInvalid  = infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "Turn-State 探测配置无效")
	ErrTurnStateProbeBusy     = infraerrors.Conflict("TURN_STATE_PROBE_BUSY", "该账号正在探测")
	ErrTurnStateProbeDisabled = infraerrors.BadRequest("TURN_STATE_PROBE_DISABLED", "渠道策略或账号开关未开启")
)

var turnStateProbeSpaceRE = regexp.MustCompile(`\s+`)

type TurnStateProbeDynamicExit struct {
	Host           string `json:"host"`
	Username       string `json:"username"`
	Password       string `json:"password,omitempty"`
	PasswordSet    bool   `json:"password_set"`
	Region         string `json:"region"`
	SessionMinutes int    `json:"session_minutes"`
}

type TurnStateProbePolicy struct {
	OverloadThreshold   int                       `json:"overload_threshold"`
	Enabled             bool                      `json:"enabled"`
	ProxyIDs            []int64                   `json:"proxy_ids"`
	Dynamic             TurnStateProbeDynamicExit `json:"dynamic"`
	Model               string                    `json:"model"`
	LengthFilterEnabled bool                      `json:"length_filter_enabled"`
	MinStateLength      int                       `json:"min_state_length"`
	Question            string                    `json:"question"`
	Answer              string                    `json:"answer"`
	FuzzyMatch          bool                      `json:"fuzzy_match"`
	RecheckMinutes      int                       `json:"recheck_minutes"`
	RPM                 int                       `json:"rpm"`
	Revision            int64                     `json:"revision"`
	UpdatedAt           time.Time                 `json:"updated_at,omitempty"`
}

func DefaultTurnStateProbePolicy() TurnStateProbePolicy {
	return TurnStateProbePolicy{
		OverloadThreshold: 3,
		Enabled:           false,
		ProxyIDs:          []int64{},
		Dynamic: TurnStateProbeDynamicExit{
			Region:         turnStateProbeDefaultRegion,
			SessionMinutes: turnStateProbeDefaultSessionMinutes,
		},
		Model:               turnStateProbeDefaultModel,
		LengthFilterEnabled: true,
		MinStateLength:      turnStateProbeDefaultMinLength,
		Question:            turnStateProbeDefaultQuestion,
		Answer:              turnStateProbeDefaultAnswer,
		FuzzyMatch:          true,
		RecheckMinutes:      turnStateProbeDefaultRecheckMinutes,
		RPM:                 turnStateProbeDefaultRPM,
	}
}

func NormalizeTurnStateProbePolicy(p TurnStateProbePolicy) (TurnStateProbePolicy, error) {
	out := p
	out.Model = strings.TrimSpace(out.Model)
	if out.Model == "" {
		out.Model = turnStateProbeDefaultModel
	}
	out.Question = strings.TrimSpace(out.Question)
	if out.Question == "" {
		out.Question = turnStateProbeDefaultQuestion
	}
	out.Answer = strings.TrimSpace(out.Answer)
	if out.Answer == "" {
		out.Answer = turnStateProbeDefaultAnswer
	}
	if out.MinStateLength == 0 {
		out.MinStateLength = turnStateProbeDefaultMinLength
	}
	if out.MinStateLength < turnStateProbeMinLength || out.MinStateLength > turnStateProbeMaxLength {
		return TurnStateProbePolicy{}, infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "长度门槛须在 1–4096 之间")
	}
	if out.RecheckMinutes == 0 {
		out.RecheckMinutes = turnStateProbeDefaultRecheckMinutes
	}
	if out.RecheckMinutes < turnStateProbeMinRecheckMinutes || out.RecheckMinutes > turnStateProbeMaxRecheckMinutes {
		return TurnStateProbePolicy{}, infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "复查间隔须在 5–1440 分钟之间")
	}
	if out.OverloadThreshold < 0 || out.OverloadThreshold > RequestHealthMaxEvents {
		return TurnStateProbePolicy{}, infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "连续 overloaded 次数须在 0–20 之间（0 为关闭）")
	}
	// Renewal is fixed at 20 minutes, including previously saved policies.
	out.RecheckMinutes = turnStateProbeDefaultRecheckMinutes
	if out.RPM == 0 {
		out.RPM = turnStateProbeDefaultRPM
	}
	if out.RPM < turnStateProbeMinRPM || out.RPM > turnStateProbeMaxRPM {
		return TurnStateProbePolicy{}, infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "每分钟调用预算须在 1–60 之间")
	}
	out.Dynamic.Host = strings.TrimSpace(out.Dynamic.Host)
	out.Dynamic.Username = strings.TrimSpace(out.Dynamic.Username)
	out.Dynamic.Region = strings.TrimSpace(out.Dynamic.Region)
	if out.Dynamic.Region == "" {
		out.Dynamic.Region = turnStateProbeDefaultRegion
	}
	if out.Dynamic.SessionMinutes == 0 {
		out.Dynamic.SessionMinutes = turnStateProbeDefaultSessionMinutes
	}
	if out.Dynamic.SessionMinutes < turnStateProbeMinSessionMinutes || out.Dynamic.SessionMinutes > turnStateProbeMaxSessionMinutes {
		return TurnStateProbePolicy{}, infraerrors.BadRequest("TURN_STATE_PROBE_INVALID", "动态出口会话时长须在 1–60 分钟之间")
	}
	if out.Dynamic.Password != "" {
		out.Dynamic.PasswordSet = true
	}
	if out.ProxyIDs == nil {
		out.ProxyIDs = []int64{}
	}
	seen := make(map[int64]struct{}, len(out.ProxyIDs))
	ids := make([]int64, 0, len(out.ProxyIDs))
	for _, id := range out.ProxyIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	out.ProxyIDs = ids
	return out, nil
}

func (p TurnStateProbePolicy) HasProbeExit() bool {
	if strings.TrimSpace(p.Dynamic.Host) != "" && strings.TrimSpace(p.Dynamic.Username) != "" && (p.Dynamic.Password != "" || p.Dynamic.PasswordSet) {
		return true
	}
	return len(p.ProxyIDs) > 0
}

func (p TurnStateProbePolicy) Public() TurnStateProbePolicy {
	out := p
	if out.Dynamic.Password != "" {
		out.Dynamic.PasswordSet = true
		out.Dynamic.Password = ""
	}
	if out.ProxyIDs == nil {
		out.ProxyIDs = []int64{}
	}
	return out
}

func (p TurnStateProbePolicy) RecheckAfter() time.Duration {
	return time.Duration(turnStateProbeDefaultRecheckMinutes) * time.Minute
}

type TurnStateProbeAccountSwitch struct {
	Enabled bool `json:"enabled"`
}

func ParseTurnStateProbeAccountSwitch(extra map[string]any) TurnStateProbeAccountSwitch {
	out := TurnStateProbeAccountSwitch{Enabled: false}
	if extra == nil {
		return out
	}
	raw, ok := extra[TurnStateProbeExtraKey]
	if !ok || raw == nil {
		return out
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return out
	}
	var parsed struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		return out
	}
	if parsed.Enabled != nil {
		out.Enabled = *parsed.Enabled
	}
	return out
}

func EnsureTurnStateProbeExtra(platform, accountType string, extra map[string]any, defaultEnabled bool) map[string]any {
	if platform != PlatformOpenAI || accountType != AccountTypeOAuth {
		return extra
	}
	if extra == nil {
		extra = map[string]any{}
	}
	if _, exists := extra[TurnStateProbeExtraKey]; exists {
		return extra
	}
	extra[TurnStateProbeExtraKey] = map[string]any{"enabled": defaultEnabled}
	return extra
}

func TurnStateProbeSwitchMap(enabled bool) map[string]any {
	return map[string]any{"enabled": enabled}
}

type TurnStateProbeAttempt struct {
	RequestedModel string
	ObservedModel  string
	State          string
	AnswerText     string
	StatusCode     int
}

func EvaluateTurnStateProbeAttempt(attempt TurnStateProbeAttempt, policy TurnStateProbePolicy) (ok bool, reason string) {
	if attempt.StatusCode != 0 && attempt.StatusCode != 200 {
		return false, fmt.Sprintf("http_%d", attempt.StatusCode)
	}
	observed := strings.TrimSpace(attempt.ObservedModel)
	requested := strings.TrimSpace(policy.Model)
	if requested == "" {
		requested = strings.TrimSpace(attempt.RequestedModel)
	}
	if observed != "" {
		lower := strings.ToLower(observed)
		if strings.Contains(lower, "luna") {
			return false, "model_remapped_luna"
		}
		if requested != "" && !strings.EqualFold(observed, requested) {
			return false, "model_mismatch"
		}
	}
	state := strings.TrimSpace(attempt.State)
	if state == "" {
		return false, "missing_turn_state"
	}
	if policy.LengthFilterEnabled && len(state) < policy.MinStateLength {
		return false, "state_too_short"
	}
	if !TurnStateProbeAnswerMatches(attempt.AnswerText, policy.Answer, policy.FuzzyMatch) {
		return false, "quiz_mismatch"
	}
	return true, ""
}

func TurnStateProbeAnswerMatches(got, want string, fuzzy bool) bool {
	want = normalizeTurnStateProbeAnswer(want)
	got = normalizeTurnStateProbeAnswer(got)
	if want == "" || got == "" {
		return false
	}
	if fuzzy {
		return strings.Contains(got, want)
	}
	return got == want
}

func normalizeTurnStateProbeAnswer(s string) string {
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
	return turnStateProbeSpaceRE.ReplaceAllString(b.String(), "")
}

func TurnStateProbeStateHash(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:6])
}

func BuildTurnStateProbeDynamicProxyURL(exit TurnStateProbeDynamicExit, sid string) (string, error) {
	host := strings.TrimSpace(exit.Host)
	user := strings.TrimSpace(exit.Username)
	pass := exit.Password
	if host == "" || user == "" || pass == "" {
		return "", fmt.Errorf("dynamic probe exit incomplete")
	}
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return "", fmt.Errorf("dynamic probe sid required")
	}
	region := strings.TrimSpace(exit.Region)
	if region == "" {
		region = turnStateProbeDefaultRegion
	}
	minutes := exit.SessionMinutes
	if minutes < turnStateProbeMinSessionMinutes {
		minutes = turnStateProbeDefaultSessionMinutes
	}
	username := user + "-region-" + region + "-sid-" + sid + "-t-" + strconv.Itoa(minutes)
	u := &url.URL{
		Scheme: "http",
		User:   url.UserPassword(username, pass),
		Host:   host,
	}
	return u.String(), nil
}

type TurnStateTicketRecord struct {
	AccountID        int64     `json:"account_id"`
	Identity         string    `json:"identity"`
	State            string    `json:"state"`
	StateHash        string    `json:"state_hash"`
	StateLength      int       `json:"state_length"`
	Model            string    `json:"model"`
	PolicyRevision   int64     `json:"policy_revision"`
	Status           string    `json:"status"`
	Attempts         int       `json:"attempts,omitempty"`
	ForbiddenRetries int       `json:"forbidden_retries,omitempty"`
	LastError        string    `json:"last_error,omitempty"`
	RecheckAt        time.Time `json:"recheck_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	HarvestedAt      time.Time `json:"harvested_at,omitempty"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
}

func (r TurnStateTicketRecord) Summary() TurnStateProbeAccountItem {
	item := TurnStateProbeAccountItem{
		AccountID:      r.AccountID,
		Status:         r.Status,
		StateHash:      r.StateHash,
		StateLength:    r.StateLength,
		Model:          r.Model,
		PolicyRevision: r.PolicyRevision,
		LastError:      r.LastError,
	}
	if !r.RecheckAt.IsZero() {
		t := r.RecheckAt
		item.RecheckAt = &t
	}
	if !r.UpdatedAt.IsZero() {
		t := r.UpdatedAt
		item.LastProbedAt = &t
	}
	return item
}

type TurnStateProbeAccountItem struct {
	AccountID      int64      `json:"account_id"`
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	Status         string     `json:"status"`
	StateHash      string     `json:"state_hash,omitempty"`
	StateLength    int        `json:"state_length,omitempty"`
	Model          string     `json:"model,omitempty"`
	PolicyRevision int64      `json:"policy_revision,omitempty"`
	RecheckAt      *time.Time `json:"recheck_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	LastProbedAt   *time.Time `json:"last_probed_at,omitempty"`
}

type TurnStateProbeOverview struct {
	Policy   TurnStateProbePolicy        `json:"policy"`
	Accounts []TurnStateProbeAccountItem `json:"accounts"`
}

type TurnStateTicketStore interface {
	Get(ctx context.Context, accountID int64) (*TurnStateTicketRecord, error)
	Put(ctx context.Context, rec TurnStateTicketRecord) error
	Delete(ctx context.Context, accountID int64) error
	BindTurn(ctx context.Context, accountID int64, turnKey, state string, ttl time.Duration) (string, error)
	GetTurnBind(ctx context.Context, accountID int64, turnKey string) (string, error)
	TryLock(ctx context.Context, accountID int64, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, accountID int64) error
	AllowRPM(ctx context.Context, bucket string, rpm int) (bool, error)
}

type TurnStateTicketLookup interface {
	BindCurrent(ctx context.Context, account *Account, identity, turnKey, model string) (state string, ok bool)
	// HasHolding reports a bindable holding ticket without pinning a turn.
	HasHolding(ctx context.Context, account *Account) bool
}

func MergeTurnStateProbePassword(next, prev TurnStateProbePolicy) TurnStateProbePolicy {
	if strings.TrimSpace(next.Dynamic.Password) == "" && prev.Dynamic.Password != "" {
		next.Dynamic.Password = prev.Dynamic.Password
		next.Dynamic.PasswordSet = true
	}
	return next
}

func TurnStateProbePolicyChanged(prev, next TurnStateProbePolicy) bool {
	prev.UpdatedAt = time.Time{}
	next.UpdatedAt = time.Time{}
	prev.Revision = 0
	next.Revision = 0
	a, _ := json.Marshal(prev)
	b, _ := json.Marshal(next)
	return !bytes.Equal(a, b)
}

func DecodeTurnStateProbePolicyJSON(raw string) (TurnStateProbePolicy, error) {
	p := DefaultTurnStateProbePolicy()
	if strings.TrimSpace(raw) == "" {
		return p, nil
	}
	d := json.NewDecoder(bytes.NewReader([]byte(raw)))
	if err := d.Decode(&p); err != nil {
		return DefaultTurnStateProbePolicy(), err
	}
	return NormalizeTurnStateProbePolicy(p)
}
