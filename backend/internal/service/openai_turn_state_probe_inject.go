package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *OpenAIGatewayService) applyTurnStateProbeHTTP(c *gin.Context, account *Account, h http.Header, body []byte, model string) {
	if s == nil || s.turnStateTickets == nil || account == nil || h == nil {
		return
	}
	if !account.IsOpenAI() || !account.UsesOpenAICodexProtocol() {
		return
	}
	if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	identity := openAICodexTurnStateOwner(c, account)
	turnKey := openAITurnStateProbeTurnKey(h, body)
	state, ok := s.turnStateTickets.BindCurrent(ctx, account, identity, turnKey, model)
	if !ok || strings.TrimSpace(state) == "" {
		return
	}
	h.Set(openAICodexTurnStateHeader, state)
	s.noteOpenAICodexTurnStateOrigin(c, account, state)
}

func (s *OpenAIGatewayService) applyTurnStateProbeWSFrame(c *gin.Context, account *Account, payload []byte, model string) []byte {
	if s == nil || s.turnStateTickets == nil || account == nil || len(payload) == 0 {
		return payload
	}
	if !account.IsOpenAI() || !account.UsesOpenAICodexProtocol() {
		return payload
	}
	if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return payload
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	identity := openAICodexTurnStateOwner(c, account)
	turnKey := openAITurnStateProbeWSTurnKey(payload)
	state, ok := s.turnStateTickets.BindCurrent(ctx, account, identity, turnKey, model)
	if !ok || strings.TrimSpace(state) == "" {
		return payload
	}
	path := "client_metadata." + openAICodexTurnStateHeader
	next, err := sjson.SetBytes(payload, path, state)
	if err != nil {
		return payload
	}
	s.noteOpenAICodexTurnStateOrigin(c, account, state)
	return next
}

func openAITurnStateProbeTurnKey(h http.Header, body []byte) string {
	if h != nil {
		if v := strings.TrimSpace(h.Get("session_id")); v != "" {
			return "s:" + v
		}
		if v := strings.TrimSpace(h.Get("conversation_id")); v != "" {
			return "c:" + v
		}
	}
	if len(body) > 0 {
		if v := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); v != "" {
			return "p:" + v
		}
	}
	return ""
}

func openAITurnStateProbeWSTurnKey(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	if v := strings.TrimSpace(gjson.GetBytes(payload, "client_metadata.thread_id").String()); v != "" {
		return "t:" + v
	}
	if v := strings.TrimSpace(gjson.GetBytes(payload, "client_metadata.session_id").String()); v != "" {
		return "s:" + v
	}
	if v := strings.TrimSpace(gjson.GetBytes(payload, "prompt_cache_key").String()); v != "" {
		return "p:" + v
	}
	return ""
}
