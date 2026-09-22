package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	TurnStateMissingModeObserve = "observe"
	TurnStateMissingModeEnforce = "enforce"

	TurnStateClassIdentityChanged = "identity_changed"
	TurnStateClassState312        = "state_312"
	TurnStateClassExitChanged     = "exit_changed"
	TurnStateClassModelMismatch   = "response_model_mismatch"
	TurnStateClassIncomplete      = "incomplete_response"
	TurnStateClassNoSession       = "no_session_identifier"
	TurnStateClassStaleResponse   = "stale_response_ignored"
	TurnStateClassTransport       = "transport_failed"
	TurnStateClassResource        = "resource_limited"
	TurnStateClassOK              = "ok"

	TurnStateGateForward   = "forward"
	TurnStateGateSwitch    = "switch_account"
	TurnStateGateUseTicket = "use_ticket"
)

// TurnStateBusinessObservation is the request-scoped view used to classify one
// upstream response. It never carries the ticket text or cookie values.
type TurnStateBusinessObservation struct {
	Current            *TurnStateTicketRecord
	AccountIdentity    string
	ResponseGeneration int64
	RequestedModel     string
	ObservedModel      string
	ErrorCode          string
	ExitDigest         string
	SessionIdentifier  string
	Complete           bool
	TransportFailed    bool
	ResourceLimited    bool
}

// TurnStateBusinessDecision is the exclusive classification of one response.
type TurnStateBusinessDecision struct {
	Class            string
	UpdateTicket     bool
	InvalidateTicket bool
	WriteSessionBind bool
	ScheduleHarvest  bool
	DegradeScheduler bool
}

// ClassifyTurnStateBusinessResponse applies one exclusive class. The first
// match wins, so a stale or identity failure cannot also rewrite the ticket.
func ClassifyTurnStateBusinessResponse(obs TurnStateBusinessObservation) TurnStateBusinessDecision {
	current := obs.Current
	if current != nil && strings.TrimSpace(obs.AccountIdentity) != "" &&
		strings.TrimSpace(current.Identity) != "" &&
		obs.AccountIdentity != current.Identity {
		return turnStateDecision(TurnStateClassIdentityChanged, false, true, false, false)
	}
	if isTurnState312(obs.ErrorCode) {
		return turnStateDecision(TurnStateClassState312, false, true, false, true)
	}
	if current != nil && strings.TrimSpace(current.ExitDigest) != "" &&
		strings.TrimSpace(obs.ExitDigest) != "" && obs.ExitDigest != current.ExitDigest {
		return turnStateDecision(TurnStateClassExitChanged, false, true, false, false)
	}
	if turnStateModelsDiffer(obs.RequestedModel, obs.ObservedModel) {
		return turnStateDecision(TurnStateClassModelMismatch, false, false, false, false)
	}
	if !obs.Complete && !obs.TransportFailed && !obs.ResourceLimited {
		return turnStateDecision(TurnStateClassIncomplete, false, false, false, false)
	}
	if strings.TrimSpace(obs.SessionIdentifier) == "" && !obs.TransportFailed && !obs.ResourceLimited {
		return turnStateDecision(TurnStateClassNoSession, false, false, false, false)
	}
	if current != nil && current.Generation > 0 && obs.ResponseGeneration > 0 && obs.ResponseGeneration < current.Generation {
		return turnStateDecision(TurnStateClassStaleResponse, false, false, false, false)
	}
	if obs.TransportFailed {
		return turnStateDecision(TurnStateClassTransport, false, false, false, false)
	}
	if obs.ResourceLimited {
		return turnStateDecision(TurnStateClassResource, false, false, false, false)
	}
	out := turnStateDecision(TurnStateClassOK, false, false, true, false)
	out.WriteSessionBind = strings.TrimSpace(obs.SessionIdentifier) != ""
	return out
}

func turnStateDecision(class string, update, invalidate, bind, harvest bool) TurnStateBusinessDecision {
	return TurnStateBusinessDecision{
		Class:            class,
		UpdateTicket:     update,
		InvalidateTicket: invalidate,
		WriteSessionBind: bind,
		ScheduleHarvest:  harvest,
		DegradeScheduler: TurnStateRiskDegradesScheduler(class),
	}
}

func isTurnState312(code string) bool {
	code = strings.TrimSpace(strings.ToLower(code))
	return code == "312" || code == "turn_state_invalid" || strings.Contains(code, "turn_state_invalid")
}

func turnStateModelsDiffer(requested, observed string) bool {
	observed = strings.TrimSpace(observed)
	requested = strings.TrimSpace(requested)
	if observed == "" || requested == "" {
		return false
	}
	if strings.Contains(strings.ToLower(observed), "luna") && !strings.Contains(strings.ToLower(requested), "luna") {
		return true
	}
	return !strings.EqualFold(observed, requested)
}

// TurnStateRiskDegradesScheduler is true only for identity, ticket, and exit
// failures. Transport, 429, and a rewritten model stay off the batch demotion.
func TurnStateRiskDegradesScheduler(class string) bool {
	switch class {
	case TurnStateClassIdentityChanged, TurnStateClassState312, TurnStateClassExitChanged:
		return true
	default:
		return false
	}
}

// turnStateExitDigest is a short irreversible summary of the harvest proxy.
// The username, password, and full URL stay out of the ticket record.
func turnStateExitDigest(proxyURL string) string {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return ""
	}
	if parsed, err := url.Parse(proxyURL); err == nil && parsed.Host != "" {
		parsed.User = nil
		proxyURL = parsed.Host
	}
	sum := sha256.Sum256([]byte(proxyURL))
	return hex.EncodeToString(sum[:6])
}

// TurnStateMissingTicketGate keeps the current product behavior unless the
// account explicitly asked to switch when no usable ticket is present.
func TurnStateMissingTicketGate(mode string, ticketUsable bool) string {
	if ticketUsable {
		return TurnStateGateUseTicket
	}
	if mode == TurnStateMissingModeEnforce {
		return TurnStateGateSwitch
	}
	return TurnStateGateForward
}

const turnStateProbeSnapshotKey = "openai_turn_state_probe_snapshot"

// turnStateProbeRequestSnapshot is the ticket injected for one request.
// Later harvests must not replace it before this request finishes.
type turnStateProbeRequestSnapshot struct {
	Ticket            TurnStateTicketRecord
	SessionIdentifier string
}

func rememberTurnStateProbeSnapshot(c *gin.Context, account *Account, ticket *TurnStateTicketRecord, session string) {
	if c == nil || account == nil || ticket == nil || strings.TrimSpace(ticket.State) == "" {
		return
	}
	pinned := *ticket
	pinned.AccountID = account.ID
	c.Set(turnStateProbeSnapshotKey, &turnStateProbeRequestSnapshot{
		Ticket:            pinned,
		SessionIdentifier: strings.TrimSpace(session),
	})
}

func turnStateProbeSnapshotFromContext(c *gin.Context, account *Account) *turnStateProbeRequestSnapshot {
	if c == nil || account == nil {
		return nil
	}
	value, ok := c.Get(turnStateProbeSnapshotKey)
	if !ok {
		return nil
	}
	snapshot, _ := value.(*turnStateProbeRequestSnapshot)
	if snapshot == nil || snapshot.Ticket.AccountID != account.ID {
		return nil
	}
	return snapshot
}

// turnStateBusinessResponse is one finished Responses attempt. It carries the
// observed model and error code, never the ticket text.
type turnStateBusinessResponse struct {
	RequestedModel  string
	ObservedModel   string
	ErrorCode       string
	Complete        bool
	TransportFailed bool
	ResourceLimited bool
	StatusCode      int
}

func turnStateBusinessResponseFromTerminal(requestedModel, eventType string, payload []byte, statusCode int, transportFailed bool) turnStateBusinessResponse {
	eventType = strings.TrimSpace(eventType)
	complete := eventType == "response.completed" || eventType == "response.done" || eventType == "[DONE]"
	if eventType == "" && len(payload) > 0 && !transportFailed {
		status := strings.TrimSpace(gjson.GetBytes(payload, "status").String())
		complete = status == "" || status == "completed"
	}
	code := turnStateErrorCode(payload)
	resource := !isTurnState312(code) && (statusCode == http.StatusTooManyRequests || statusCode >= 500 || isOpenAIRequestScopedCapacityShed("", payload))
	return turnStateBusinessResponse{
		RequestedModel:  requestedModel,
		ObservedModel:   firstValidTrimmedGJSONString(payload, "response.model", "model"),
		ErrorCode:       code,
		Complete:        complete,
		TransportFailed: transportFailed,
		ResourceLimited: resource,
		StatusCode:      statusCode,
	}
}

func turnStateErrorCode(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	for _, path := range []string{"error.code", "response.error.code", "code"} {
		if code := strings.TrimSpace(gjson.GetBytes(payload, path).String()); code != "" {
			return code
		}
	}
	return ""
}

func (s *OpenAIGatewayService) noteTurnStateStreamOutcome(c *gin.Context, account *Account, observer *upstreamResponseModelObserver, requestedModel, eventType string, payload []byte, sawTerminal, sawFailed, transportFailed bool) {
	if !sawTerminal && !transportFailed {
		return
	}
	observed := turnStateBusinessResponseFromTerminal(requestedModel, eventType, payload, http.StatusOK, transportFailed || !sawTerminal)
	if observer != nil && strings.TrimSpace(observed.ObservedModel) == "" {
		observed.ObservedModel = observer.Model()
	}
	if sawFailed {
		observed.Complete = false
	}
	s.noteTurnStateBusinessResponse(c, account, observed)
}

func (s *OpenAIGatewayService) noteTurnStateBusinessPayload(c *gin.Context, account *Account, requestedModel, eventType string, payload []byte, statusCode int, transportFailed bool) {
	s.noteTurnStateBusinessResponse(c, account, turnStateBusinessResponseFromTerminal(requestedModel, eventType, payload, statusCode, transportFailed))
}

func (s *OpenAIGatewayService) noteTurnStateBusinessResponse(c *gin.Context, account *Account, resp turnStateBusinessResponse) {
	if s == nil || account == nil || !account.IsOpenAI() {
		return
	}
	if !ParseTurnStateProbeAccountSwitch(account.Extra).Enabled {
		return
	}
	snapshot := turnStateProbeSnapshotFromContext(c, account)
	if snapshot == nil || strings.TrimSpace(snapshot.Ticket.State) == "" {
		return
	}
	pinned := snapshot.Ticket
	observedExit := pinned.ExitDigest
	if sticky := stickySnapshot(c, account); sticky != nil && strings.TrimSpace(sticky.proxyURL) != "" {
		observedExit = turnStateExitDigest(sticky.proxyURL)
	}
	decision := ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current:            &pinned,
		AccountIdentity:    turnStateProbeTicketIdentity(account),
		ResponseGeneration: pinned.Generation,
		RequestedModel:     resp.RequestedModel,
		ObservedModel:      resp.ObservedModel,
		ErrorCode:          resp.ErrorCode,
		ExitDigest:         observedExit,
		SessionIdentifier:  snapshot.SessionIdentifier,
		Complete:           resp.Complete,
		TransportFailed:    resp.TransportFailed,
		ResourceLimited:    resp.ResourceLimited,
	})
	if decision.Class == "" || decision.Class == TurnStateClassOK {
		return
	}
	s.recordTurnStateHealthClass(c, account, decision.Class, resp)
	if decision.InvalidateTicket {
		s.invalidatePinnedTurnStateTicket(c, account, &pinned, decision)
	}
}

func (s *OpenAIGatewayService) recordTurnStateHealthClass(c *gin.Context, account *Account, class string, resp turnStateBusinessResponse) {
	if s == nil || s.requestHealth == nil || account == nil || strings.TrimSpace(class) == "" || class == TurnStateClassOK {
		return
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	slot := RequestHealthSlotOK
	if class != TurnStateClassNoSession && class != TurnStateClassStaleResponse {
		slot = RequestHealthSlotFail
	}
	s.requestHealth.Record(ctx, RequestHealthRecordInput{
		AccountID:  account.ID,
		ProxyID:    EgressProxyIDFrom(ctx, account),
		Slot:       slot,
		StatusCode: resp.StatusCode,
		Message:    class,
		Model:      firstNonEmpty(resp.ObservedModel, resp.RequestedModel),
		Class:      class,
	})
}

// invalidatePinnedTurnStateTicket drops the injected ticket when identity, 312,
// or the exit digest says it must not be reused. A newer harvest is left alone.
func (s *OpenAIGatewayService) invalidatePinnedTurnStateTicket(c *gin.Context, account *Account, pinned *TurnStateTicketRecord, decision TurnStateBusinessDecision) {
	probe, ok := s.turnStateTickets.(*TurnStateProbeService)
	if !ok || probe == nil || probe.tickets == nil || account == nil || pinned == nil {
		return
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	current, err := probe.tickets.Get(ctx, account.ID)
	if err != nil || current == nil || strings.TrimSpace(current.State) == "" {
		return
	}
	if pinned.Generation > 0 && current.Generation > pinned.Generation {
		return
	}
	now := time.Now()
	cleared := *current
	cleared.State = ""
	cleared.StateHash = ""
	cleared.StateLength = 0
	cleared.Status = turnStateProbeStatusCooldown
	cleared.LastError = decision.Class
	cleared.UpdatedAt = now
	cleared.UpdatedUnixMs = now.UnixMilli()
	if decision.ScheduleHarvest {
		cleared.RecheckAt = now
	}
	_ = probe.tickets.Put(ctx, cleared)
}

// turnStateRiskClassFromError reads a class previously attached as the
// failover reason. Empty means the error is not a turn-state classification.
func turnStateRiskClassFromError(err error) string {
	var failover *UpstreamFailoverError
	if err == nil || !errors.As(err, &failover) || failover == nil {
		return ""
	}
	switch string(failover.Reason) {
	case TurnStateClassTransport, TurnStateClassResource, TurnStateClassModelMismatch,
		TurnStateClassIdentityChanged, TurnStateClassState312, TurnStateClassExitChanged:
		return string(failover.Reason)
	default:
		return ""
	}
}
