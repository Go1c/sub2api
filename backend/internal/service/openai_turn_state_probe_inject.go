package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// enforceTurnStateTicket switches the account only when that account set
// mode=enforce and no usable ticket is bound. Observe, the default, forwards.
func (s *OpenAIGatewayService) enforceTurnStateTicket(c *gin.Context, account *Account) error {
	if s == nil || s.turnStateTickets == nil || account == nil {
		return nil
	}
	if !account.IsOpenAI() || !account.UsesOpenAICodexProtocol() {
		return nil
	}
	sw := ParseTurnStateProbeAccountSwitch(account.Extra)
	if !sw.Enabled || sw.Mode != TurnStateMissingModeEnforce {
		return nil
	}
	if snapshot := stickySnapshot(c, account); snapshot != nil && strings.TrimSpace(snapshot.ticket.State) != "" && stickySnapshotError(c, account) == nil {
		return nil
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	if s.turnStateTickets.HasHolding(ctx, account) {
		return nil
	}
	return &UpstreamFailoverError{
		StatusCode:             http.StatusServiceUnavailable,
		RetryableOnSameAccount: false,
		Scope:                  GatewayFailureScopeAccount,
		Reason:                 GatewayFailureReason("ticket_missing"),
		NextAccountAction:      NextAccountRetry,
		ClientStatusCode:       http.StatusServiceUnavailable,
		ClientMessage:          "turn-state ticket unavailable",
	}
}

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
	state, ticket := s.pinnedTurnStateForRequest(c, ctx, account, identity, turnKey, model)
	if strings.TrimSpace(state) == "" {
		return
	}
	rememberTurnStateProbeSnapshot(c, account, ticket, turnKey)
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
	state, ticket := s.pinnedTurnStateForRequest(c, ctx, account, identity, turnKey, model)
	if strings.TrimSpace(state) == "" {
		return payload
	}
	rememberTurnStateProbeSnapshot(c, account, ticket, turnKey)
	path := "client_metadata." + openAICodexTurnStateHeader
	next, err := sjson.SetBytes(payload, path, state)
	if err != nil {
		return payload
	}
	s.noteOpenAICodexTurnStateOrigin(c, account, state)
	return next
}

// pinnedTurnStateForRequest reuses the snapshot taken at the first inject.
// Sticky requests keep the sticky ticket; other requests read CurrentTicket once.
func (s *OpenAIGatewayService) pinnedTurnStateForRequest(c *gin.Context, ctx context.Context, account *Account, identity, turnKey, model string) (string, *TurnStateTicketRecord) {
	if pinned := turnStateProbeSnapshotFromContext(c, account); pinned != nil && strings.TrimSpace(pinned.Ticket.State) != "" {
		state := pinned.Ticket.State
		return state, &pinned.Ticket
	}
	if snapshot := stickySnapshot(c, account); snapshot != nil && stickySnapshotError(c, account) == nil {
		ticket := snapshot.ticket
		if strings.TrimSpace(ticket.State) == "" {
			return "", nil
		}
		return ticket.State, &ticket
	}
	if s == nil || s.turnStateTickets == nil {
		return "", nil
	}
	if reader, ok := s.turnStateTickets.(TurnStateTicketReader); ok {
		ticket, ok := reader.CurrentTicket(ctx, account)
		if !ok || ticket == nil || strings.TrimSpace(ticket.State) == "" {
			return "", nil
		}
		if strings.TrimSpace(turnKey) != "" {
			if bound, boundOK := s.turnStateTickets.BindCurrent(ctx, account, identity, turnKey, model); boundOK && strings.TrimSpace(bound) != "" {
				return bound, ticket
			}
		}
		return ticket.State, ticket
	}
	state, ok := s.turnStateTickets.BindCurrent(ctx, account, identity, turnKey, model)
	if !ok || strings.TrimSpace(state) == "" {
		return "", nil
	}
	return state, &TurnStateTicketRecord{AccountID: account.ID, State: state, Identity: identity}
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
