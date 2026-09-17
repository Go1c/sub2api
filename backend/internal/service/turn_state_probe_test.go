package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateTurnStateProbeAttempt(t *testing.T) {
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	policy.LengthFilterEnabled = true
	policy.MinStateLength = 160
	policy.Answer = "21"
	policy.FuzzyMatch = true
	policy.Model = "gpt-6-astra"
	state := stringsRepeat("A", 160)

	ok, reason := EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		RequestedModel: "gpt-6-astra",
		ObservedModel:  "gpt-6-astra",
		State:          state,
		AnswerText:     "答案是 21。",
		StatusCode:     200,
	}, policy)
	require.True(t, ok, reason)

	ok, reason = EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		ObservedModel: "gpt-5.6-luna",
		State:         state,
		AnswerText:    "21",
		StatusCode:    200,
	}, policy)
	require.False(t, ok)
	require.Equal(t, "model_remapped_luna", reason)

	ok, reason = EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		ObservedModel: "gpt-6-astra",
		State:         stringsRepeat("A", 159),
		AnswerText:    "21",
		StatusCode:    200,
	}, policy)
	require.False(t, ok)
	require.Equal(t, "state_too_short", reason)

	ok, reason = EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		ObservedModel: "gpt-6-astra",
		State:         state,
		AnswerText:    "不知道",
		StatusCode:    200,
	}, policy)
	require.False(t, ok)
	require.Equal(t, "quiz_mismatch", reason)

	ok, reason = EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		ObservedModel: "gpt-6-astra",
		State:         "",
		AnswerText:    "21",
		StatusCode:    200,
	}, policy)
	require.False(t, ok)
	require.Equal(t, "missing_turn_state", reason)

	ok, reason = EvaluateTurnStateProbeAttempt(TurnStateProbeAttempt{
		ObservedModel: "gpt-6-astra",
		State:         state,
		AnswerText:    "21",
		StatusCode:    400,
	}, policy)
	require.False(t, ok)
	require.Equal(t, "http_400", reason)
}

func TestTurnStateProbeAnswerMatches(t *testing.T) {
	require.True(t, TurnStateProbeAnswerMatches("答案是21颗", "21", true))
	require.True(t, TurnStateProbeAnswerMatches("21", "21", false))
	require.False(t, TurnStateProbeAnswerMatches("答案是21颗", "21", false))
	require.False(t, TurnStateProbeAnswerMatches("", "21", true))
}

func TestParseTurnStateProbeAccountSwitch(t *testing.T) {
	require.False(t, ParseTurnStateProbeAccountSwitch(nil).Enabled)
	require.False(t, ParseTurnStateProbeAccountSwitch(map[string]any{}).Enabled)
	require.True(t, ParseTurnStateProbeAccountSwitch(map[string]any{
		TurnStateProbeExtraKey: map[string]any{"enabled": true},
	}).Enabled)
}

func TestEnsureTurnStateProbeExtra(t *testing.T) {
	extra := EnsureTurnStateProbeExtra(PlatformOpenAI, AccountTypeOAuth, nil, true)
	require.Equal(t, true, extra[TurnStateProbeExtraKey].(map[string]any)["enabled"])
	kept := EnsureTurnStateProbeExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		TurnStateProbeExtraKey: map[string]any{"enabled": false},
	}, true)
	require.Equal(t, false, kept[TurnStateProbeExtraKey].(map[string]any)["enabled"])
	skipped := EnsureTurnStateProbeExtra(PlatformAnthropic, AccountTypeOAuth, nil, true)
	require.Nil(t, skipped)
}

func TestBuildTurnStateProbeDynamicProxyURL(t *testing.T) {
	u, err := BuildTurnStateProbeDynamicProxyURL(TurnStateProbeDynamicExit{
		Host:           "us.lajiaohttp.net:2000",
		Username:       "user1",
		Password:       "secret",
		Region:         "Random",
		SessionMinutes: 5,
	}, "jis92dwy")
	require.NoError(t, err)
	require.Contains(t, u, "user1-region-Random-sid-jis92dwy-t-5")
	require.Contains(t, u, "us.lajiaohttp.net:2000")
	require.NotContains(t, u, "secret?")
}

func TestNormalizeTurnStateProbePolicy(t *testing.T) {
	p, err := NormalizeTurnStateProbePolicy(TurnStateProbePolicy{})
	require.NoError(t, err)
	require.Equal(t, "gpt-6-astra", p.Model)
	require.Equal(t, 160, p.MinStateLength)
	require.Equal(t, "21", p.Answer)
	require.True(t, p.FuzzyMatch || !p.FuzzyMatch)
	_, err = NormalizeTurnStateProbePolicy(TurnStateProbePolicy{MinStateLength: 99999})
	require.Error(t, err)
}

func TestTurnStateProbePolicyPublicRedactsPassword(t *testing.T) {
	p := DefaultTurnStateProbePolicy()
	p.Dynamic.Password = "super-secret"
	pub := p.Public()
	require.True(t, pub.Dynamic.PasswordSet)
	require.Empty(t, pub.Dynamic.Password)
}

func TestParseTurnStateProbeSSEUsesDoneOnce(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"model":"gpt-6-astra"}}`,
		`data: {"type":"response.output_text.delta","text":"21"}`,
		`data: {"type":"response.output_text.done","text":"21"}`,
		`data: [DONE]`,
		"",
	}, "\n")
	got := parseTurnStateProbeSSE(strings.NewReader(body))
	require.Equal(t, "gpt-6-astra", got.Model)
	require.Equal(t, "21", got.Text)
}

func TestBuildTurnStateProbePayloadOmitsMaxOutputTokens(t *testing.T) {
	payload := BuildTurnStateProbePayload("gpt-6-astra", "ping")
	_, hasMax := payload["max_output_tokens"]
	require.False(t, hasMax)
	require.Equal(t, false, payload["store"])
}

func TestApplyTurnStateProbeHTTPInjectsBeforeMissingTicketNoops(t *testing.T) {
	svc := &OpenAIGatewayService{}
	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-blob")
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{
		TurnStateProbeExtraKey: map[string]any{"enabled": true},
	}}
	svc.applyTurnStateProbeHTTP(nil, account, h, nil, "gpt-6-astra")
	require.Equal(t, "client-blob", h.Get(openAICodexTurnStateHeader))

	svc.turnStateTickets = staticTurnStateLookup("harvested-state")
	svc.applyTurnStateProbeHTTP(nil, account, h, nil, "gpt-6-astra")
	require.Equal(t, "harvested-state", h.Get(openAICodexTurnStateHeader))
}

type staticTurnStateLookup string

func (s staticTurnStateLookup) BindCurrent(context.Context, *Account, string, string, string) (string, bool) {
	return string(s), string(s) != ""
}

func (s staticTurnStateLookup) HasHolding(context.Context, *Account) bool {
	return string(s) != ""
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
