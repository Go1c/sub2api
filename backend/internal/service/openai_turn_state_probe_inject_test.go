package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type injectTurnStateTicketLookup string

func (s injectTurnStateTicketLookup) BindCurrent(context.Context, *Account, string, string, string) (string, bool) {
	return string(s), string(s) != ""
}

func (s injectTurnStateTicketLookup) HasHolding(context.Context, *Account) bool {
	return string(s) != ""
}

func turnStateProbeInjectAccount(enabled bool) *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			TurnStateProbeExtraKey: map[string]any{"enabled": enabled},
		},
	}
}

func TestApplyTurnStateProbeHTTP_ReplacesClientBlob(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(injectTurnStateTicketLookup("probe-state"))
	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-blob")
	account := turnStateProbeInjectAccount(true)

	svc.applyTurnStateProbeHTTP(nil, account, h, []byte(`{"model":"gpt-6-astra"}`), "gpt-6-astra")
	require.Equal(t, "probe-state", h.Get(openAICodexTurnStateHeader))
}

func TestApplyTurnStateProbeHTTP_NoTicketPreservesUnknownBlob(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(injectTurnStateTicketLookup(""))
	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "unknown-blob")
	account := turnStateProbeInjectAccount(true)

	svc.applyTurnStateProbeHTTP(nil, account, h, nil, "gpt-6-astra")
	require.Equal(t, "unknown-blob", h.Get(openAICodexTurnStateHeader))

	svc.guardOpenAICodexTurnStateEcho(nil, account, h)
	require.Equal(t, "unknown-blob", h.Get(openAICodexTurnStateHeader), "无溯源的未知 blob 不被守卫剥离")
}

func TestApplyTurnStateProbeWSFrame_ReplacesClientMetadata(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(injectTurnStateTicketLookup("probe-state"))
	account := turnStateProbeInjectAccount(true)
	frame := []byte(`{"model":"gpt-6-astra","client_metadata":{"x-codex-turn-state":"client-blob"}}`)

	out := svc.applyTurnStateProbeWSFrame(nil, account, frame, gjson.GetBytes(frame, "model").String())
	require.Equal(t, "probe-state", gjson.GetBytes(out, "client_metadata."+openAICodexTurnStateHeader).String())
	require.NotEqual(t, "client-blob", gjson.GetBytes(out, "client_metadata."+openAICodexTurnStateHeader).String())
}

func TestApplyTurnStateProbe_DisabledSwitchNoInject(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(injectTurnStateTicketLookup("probe-state"))
	account := turnStateProbeInjectAccount(false)

	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-blob")
	svc.applyTurnStateProbeHTTP(nil, account, h, nil, "gpt-6-astra")
	require.Equal(t, "client-blob", h.Get(openAICodexTurnStateHeader))

	frame := []byte(`{"model":"gpt-6-astra","client_metadata":{"x-codex-turn-state":"client-blob"}}`)
	out := svc.applyTurnStateProbeWSFrame(nil, account, frame, "gpt-6-astra")
	require.Equal(t, "client-blob", gjson.GetBytes(out, "client_metadata."+openAICodexTurnStateHeader).String())
}

func TestApplyTurnStateProbe_NilLookupNoPanicNoInject(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := turnStateProbeInjectAccount(true)

	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-blob")
	require.NotPanics(t, func() {
		svc.applyTurnStateProbeHTTP(nil, account, h, nil, "gpt-6-astra")
	})
	require.Equal(t, "client-blob", h.Get(openAICodexTurnStateHeader))

	frame := []byte(`{"model":"gpt-6-astra","client_metadata":{"x-codex-turn-state":"client-blob"}}`)
	var out []byte
	require.NotPanics(t, func() {
		out = svc.applyTurnStateProbeWSFrame(nil, account, frame, "gpt-6-astra")
	})
	require.Equal(t, "client-blob", gjson.GetBytes(out, "client_metadata."+openAICodexTurnStateHeader).String())
}
