package service

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type snapshotTurnStateLookup struct {
	ticket *TurnStateTicketRecord
}

func (s snapshotTurnStateLookup) BindCurrent(context.Context, *Account, string, string, string) (string, bool) {
	if s.ticket == nil || s.ticket.State == "" {
		return "", false
	}
	return s.ticket.State, true
}

func (s snapshotTurnStateLookup) HasHolding(context.Context, *Account) bool {
	return s.ticket != nil && s.ticket.State != ""
}

func (s snapshotTurnStateLookup) CurrentTicket(context.Context, *Account) (*TurnStateTicketRecord, bool) {
	if s.ticket == nil || s.ticket.State == "" {
		return nil, false
	}
	cp := *s.ticket
	return &cp, true
}

type recordingRequestHealth struct {
	mu   sync.Mutex
	last RequestHealthEvent
}

func (r *recordingRequestHealth) Append(_ context.Context, ev RequestHealthEvent) error {
	r.mu.Lock()
	r.last = ev
	r.mu.Unlock()
	return nil
}

func (r *recordingRequestHealth) event() RequestHealthEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last
}

func (r *recordingRequestHealth) List(context.Context, int64, int64, int) ([]RequestHealthEvent, error) {
	return nil, nil
}

func (r *recordingRequestHealth) Runtime(context.Context, int64, int64) (int, *time.Time) {
	return 0, nil
}

func turnStateTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}

func holdingTicket() *TurnStateTicketRecord {
	now := time.Now()
	return &TurnStateTicketRecord{
		AccountID:   7,
		Identity:    "acct-7",
		State:       "astra-ticket",
		StateHash:   TurnStateProbeStateHash("astra-ticket"),
		StateLength: 332,
		Model:       "gpt-6-astra",
		Status:      turnStateProbeStatusHolding,
		Generation:  4,
		HarvestedAt: now.Add(-time.Minute),
		ExpiresAt:   now.Add(3 * time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
		ExitDigest:  "digest-a",
	}
}

func TestClassifyTurnStateBusinessResponse(t *testing.T) {
	current := holdingTicket()

	decision := ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, ResponseGeneration: 4, RequestedModel: "gpt-6-astra", ObservedModel: "gpt-5.6-luna", Complete: true,
	})
	require.Equal(t, TurnStateClassModelMismatch, decision.Class)
	require.False(t, decision.UpdateTicket)
	require.False(t, decision.InvalidateTicket)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, ResponseGeneration: 4, ErrorCode: "312", Complete: true,
	})
	require.Equal(t, TurnStateClassState312, decision.Class)
	require.True(t, decision.InvalidateTicket)
	require.True(t, decision.ScheduleHarvest)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, ResponseGeneration: 4, Complete: false, RequestedModel: "gpt-6-astra", ObservedModel: "gpt-6-astra",
	})
	require.Equal(t, TurnStateClassIncomplete, decision.Class)
	require.False(t, decision.UpdateTicket)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, ResponseGeneration: 4, SessionIdentifier: "", Complete: true,
		RequestedModel: "gpt-6-astra", ObservedModel: "gpt-6-astra",
	})
	require.Equal(t, TurnStateClassNoSession, decision.Class)
	require.False(t, decision.WriteSessionBind)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, ResponseGeneration: 3, Complete: true, SessionIdentifier: "s:1",
		RequestedModel: "gpt-6-astra", ObservedModel: "gpt-6-astra",
	})
	require.Equal(t, TurnStateClassStaleResponse, decision.Class)
	require.False(t, decision.UpdateTicket)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, AccountIdentity: "acct-other", ResponseGeneration: 4, Complete: true, SessionIdentifier: "s:1",
	})
	require.Equal(t, TurnStateClassIdentityChanged, decision.Class)
	require.True(t, decision.InvalidateTicket)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, AccountIdentity: "acct-7", ResponseGeneration: 4, ExitDigest: "digest-b",
		Complete: true, SessionIdentifier: "s:1", RequestedModel: "gpt-6-astra", ObservedModel: "gpt-6-astra",
	})
	require.Equal(t, TurnStateClassExitChanged, decision.Class)
	require.True(t, decision.InvalidateTicket)

	decision = ClassifyTurnStateBusinessResponse(TurnStateBusinessObservation{
		Current: current, AccountIdentity: "acct-7", ResponseGeneration: 4, ExitDigest: "digest-a",
		TransportFailed: true, SessionIdentifier: "s:1",
	})
	require.Equal(t, TurnStateClassTransport, decision.Class)
	require.False(t, decision.InvalidateTicket)
	require.False(t, decision.DegradeScheduler)
}

func TestTurnStateRiskClassDegrade(t *testing.T) {
	require.True(t, TurnStateRiskDegradesScheduler(TurnStateClassIdentityChanged))
	require.True(t, TurnStateRiskDegradesScheduler(TurnStateClassState312))
	require.True(t, TurnStateRiskDegradesScheduler(TurnStateClassExitChanged))
	require.False(t, TurnStateRiskDegradesScheduler(TurnStateClassTransport))
	require.False(t, TurnStateRiskDegradesScheduler(TurnStateClassResource))
	require.False(t, TurnStateRiskDegradesScheduler(TurnStateClassModelMismatch))
}

func TestTurnStateProbeSnapshotPinsInjectedGeneration(t *testing.T) {
	svc := &OpenAIGatewayService{}
	current := holdingTicket()
	lookup := &snapshotTurnStateLookup{ticket: current}
	svc.SetTurnStateTicketLookup(lookup)
	account := turnStateProbeInjectAccount(true)
	account.ID = current.AccountID
	c := turnStateTestContext()
	h := http.Header{}

	svc.applyTurnStateProbeHTTP(c, account, h, []byte(`{"model":"gpt-6-astra","prompt_cache_key":"thread-1"}`), "gpt-6-astra")
	require.Equal(t, current.State, h.Get(openAICodexTurnStateHeader))

	pinned := turnStateProbeSnapshotFromContext(c, account)
	require.NotNil(t, pinned)
	require.Equal(t, current.Generation, pinned.Ticket.Generation)
	require.Equal(t, current.ExitDigest, pinned.Ticket.ExitDigest)
	require.Equal(t, "p:thread-1", pinned.SessionIdentifier)

	lookup.ticket = &TurnStateTicketRecord{
		AccountID: current.AccountID, Identity: current.Identity, State: "later-ticket",
		Generation: current.Generation + 1, ExitDigest: "digest-b", Status: turnStateProbeStatusHolding,
	}
	h2 := http.Header{}
	svc.applyTurnStateProbeHTTP(c, account, h2, nil, "gpt-6-astra")
	require.Equal(t, current.State, h2.Get(openAICodexTurnStateHeader))
	require.Equal(t, current.Generation, turnStateProbeSnapshotFromContext(c, account).Ticket.Generation)
}

func TestNoteTurnStateBusinessResponse_ModelMismatchDoesNotClear(t *testing.T) {
	store := newMemoryTurnStateTicketStore()
	probe := &TurnStateProbeService{tickets: store}
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	probe.rememberPolicy(policy)
	account := turnStateProbeInjectAccount(true)
	account.ID = 7
	current := holdingTicket()
	current.Identity = turnStateProbeTicketIdentity(account)
	current.PolicyRevision = policy.Revision
	require.NoError(t, store.Put(context.Background(), *current))
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(probe)
	health := &recordingRequestHealth{}
	svc.SetRequestHealthService(&AccountRequestHealthService{store: health})
	c := turnStateTestContext()

	h := http.Header{}
	svc.applyTurnStateProbeHTTP(c, account, h, []byte(`{"prompt_cache_key":"thread-1"}`), "gpt-6-astra")
	svc.noteTurnStateBusinessResponse(c, account, turnStateBusinessResponse{
		RequestedModel: "gpt-6-astra",
		ObservedModel:  "gpt-5.6-luna",
		Complete:       true,
		StatusCode:     http.StatusOK,
	})

	kept, err := store.Get(context.Background(), current.AccountID)
	require.NoError(t, err)
	require.Equal(t, current.State, kept.State)
	require.Equal(t, current.Generation, kept.Generation)
	require.Eventually(t, func() bool {
		ev := health.event()
		return ev.Class == TurnStateClassModelMismatch && ev.Slot == RequestHealthSlotFail && !strings.Contains(ev.Message, current.State)
	}, time.Second, 10*time.Millisecond)
}

func TestNoteTurnStateBusinessResponse_312ClearsAndSchedules(t *testing.T) {
	store := newMemoryTurnStateTicketStore()
	probe := &TurnStateProbeService{tickets: store}
	policy := DefaultTurnStateProbePolicy()
	policy.Enabled = true
	probe.rememberPolicy(policy)
	account := turnStateProbeInjectAccount(true)
	account.ID = 7
	current := holdingTicket()
	current.Identity = turnStateProbeTicketIdentity(account)
	current.PolicyRevision = policy.Revision
	require.NoError(t, store.Put(context.Background(), *current))
	svc := &OpenAIGatewayService{}
	svc.SetTurnStateTicketLookup(probe)
	c := turnStateTestContext()

	h := http.Header{}
	svc.applyTurnStateProbeHTTP(c, account, h, nil, "gpt-6-astra")
	svc.noteTurnStateBusinessResponse(c, account, turnStateBusinessResponse{
		ErrorCode:  "312",
		Complete:   true,
		StatusCode: http.StatusOK,
	})

	kept, err := store.Get(context.Background(), current.AccountID)
	require.NoError(t, err)
	require.Empty(t, kept.State)
	require.Equal(t, turnStateProbeStatusCooldown, kept.Status)
	require.Equal(t, "state_312", kept.LastError)
	require.False(t, kept.RecheckAt.After(time.Now().Add(time.Second)))
	require.Equal(t, current.Generation, kept.Generation)

	later := *current
	later.Generation = current.Generation + 1
	later.State = "fresh-ticket"
	later.UpdatedAt = time.Now().Add(time.Second)
	later.UpdatedUnixMs = later.UpdatedAt.UnixMilli()
	require.NoError(t, store.Put(context.Background(), later))
	svc.noteTurnStateBusinessResponse(c, account, turnStateBusinessResponse{
		ErrorCode: "312", Complete: true, StatusCode: http.StatusOK,
	})
	kept, err = store.Get(context.Background(), current.AccountID)
	require.NoError(t, err)
	require.Equal(t, "fresh-ticket", kept.State)
	require.Equal(t, later.Generation, kept.Generation)
}

func TestTurnStateMissingModeDefaultsObserve(t *testing.T) {
	require.Equal(t, TurnStateMissingModeObserve, ParseTurnStateProbeAccountSwitch(nil).Mode)
	require.Equal(t, TurnStateMissingModeObserve, ParseTurnStateProbeAccountSwitch(map[string]any{
		TurnStateProbeExtraKey: map[string]any{"enabled": true},
	}).Mode)
	require.Equal(t, TurnStateMissingModeEnforce, ParseTurnStateProbeAccountSwitch(map[string]any{
		TurnStateProbeExtraKey: map[string]any{"enabled": true, "mode": "enforce"},
	}).Mode)
	require.Equal(t, TurnStateGateForward, TurnStateMissingTicketGate(TurnStateMissingModeObserve, false))
	require.Equal(t, TurnStateGateSwitch, TurnStateMissingTicketGate(TurnStateMissingModeEnforce, false))
	require.Equal(t, TurnStateGateUseTicket, TurnStateMissingTicketGate(TurnStateMissingModeEnforce, true))
}
