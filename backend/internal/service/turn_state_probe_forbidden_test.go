package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTurnStateProbeForbiddenPreservesTicketAndBacksOff(t *testing.T) {
	ctx := context.Background()
	upstream := &recordingTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>blocked secret-token</html>"))
	})}
	account := newOAuthProbeAccount(431, true)
	tickets := newMemoryTurnStateTicketStore()
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	policy := enableTurnStateProbePolicy(t, svc)
	harvested := time.Now().Add(-20 * time.Minute)
	expires := harvested.Add(time.Hour)
	require.NoError(t, tickets.Put(ctx, TurnStateTicketRecord{AccountID: account.ID, State: "valid-old-state", Status: "holding", PolicyRevision: policy.Revision, HarvestedAt: harvested, ExpiresAt: expires, RecheckAt: time.Now().Add(-time.Second)}))
	for _, delay := range []time.Duration{2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute, 30 * time.Minute, 30 * time.Minute} {
		// Simulate a fresh RPM window and a due scheduled retry without sleeping.
		tickets.rpm = map[string]int{}
		before := time.Now()
		require.Error(t, svc.ProbeAccount(ctx, account.ID))
		rec, err := tickets.Get(ctx, account.ID)
		require.NoError(t, err)
		require.Equal(t, "cooldown", rec.Status)
		require.Equal(t, "http_403", rec.LastError)
		require.Equal(t, "valid-old-state", rec.State)
		require.Equal(t, expires, rec.ExpiresAt)
		require.WithinDuration(t, before.Add(delay), rec.RecheckAt, 2*time.Second)
		require.True(t, svc.HasHolding(ctx, account))
		require.False(t, svc.ticketDue(ctx, account.ID, policy, before.Add(delay-time.Second)))
		require.True(t, svc.ticketDue(ctx, account.ID, policy, before.Add(delay+time.Second)))
		rec.RecheckAt = time.Now().Add(-time.Second)
		require.NoError(t, tickets.Put(ctx, *rec))
	}
	upstream.handler = turnStateProbeSuccessSSE(strings.Repeat("N", 292))
	tickets.rpm = map[string]int{}
	require.NoError(t, svc.ProbeAccount(ctx, account.ID))
	rec, err := tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "holding", rec.Status)
	require.Equal(t, strings.Repeat("N", 292), rec.State)
	// A new 403 after success starts at the initial delay.
	upstream.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) })
	rec.RecheckAt = time.Now().Add(-time.Second)
	require.NoError(t, tickets.Put(ctx, *rec))
	tickets.rpm = map[string]int{}
	require.Error(t, svc.ProbeAccount(ctx, account.ID))
	rec, err = tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(2*time.Minute), rec.RecheckAt, 2*time.Second)
	rec.HarvestedAt = time.Now().Add(-2 * time.Hour)
	rec.ExpiresAt = time.Now().Add(-time.Hour)
	require.NoError(t, tickets.Put(ctx, *rec))
	require.False(t, svc.HasHolding(ctx, account))
}

func TestTurnStateProbeRecoversOnlyLegacyForbiddenSkip(t *testing.T) {
	for _, tc := range []struct {
		reason  string
		recover bool
	}{
		{`error: code=403 reason="TURN_STATE_PROBE_FORBIDDEN" message="探测上游返回 403"`, true},
		{`error: code=401 reason="TURN_STATE_PROBE_UNAUTHORIZED"`, false},
		{`error: code=403 reason="TURN_STATE_PROBE_ACCOUNT_FORBIDDEN"`, false},
		{"account_error", false},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			ctx := context.Background()
			account := newOAuthProbeAccount(432, true)
			tickets := newMemoryTurnStateTicketStore()
			upstream := &recordingTurnStateHTTPUpstream{handler: turnStateProbeSuccessSSE(strings.Repeat("N", 292))}
			svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
			policy := enableTurnStateProbePolicy(t, svc)
			rec := TurnStateTicketRecord{AccountID: account.ID, Status: "skipped", LastError: tc.reason, PolicyRevision: policy.Revision, UpdatedAt: time.Now()}
			require.NoError(t, tickets.Put(ctx, rec))
			require.False(t, svc.ticketDue(ctx, account.ID, policy, time.Now()))
			require.NoError(t, svc.ProbeAccount(ctx, account.ID))
			require.Nil(t, upstream.lastReq, "fresh legacy failures must still cool down")
			rec.UpdatedAt = time.Now().Add(-time.Hour)
			require.NoError(t, tickets.Put(ctx, rec))
			ids, err := svc.listDueAccountIDs(ctx, policy)
			require.NoError(t, err)
			require.Equal(t, tc.recover, len(ids) == 1)
			require.NoError(t, svc.ProbeAccount(ctx, account.ID))
			got, err := tickets.Get(ctx, account.ID)
			require.NoError(t, err)
			if tc.recover {
				require.Equal(t, "holding", got.Status)
			} else {
				require.Equal(t, "skipped", got.Status)
			}
		})
	}
}

func TestTurnStateProbeExplicitAccount403StopsAndClearsTicket(t *testing.T) {
	for _, body := range []string{`{"error":{"code":"account_disabled"}}`, `{"detail":{"code":"deactivated_workspace"}}`, `{"error":{"code":"token_invalidated"}}`} {
		t.Run(body, func(t *testing.T) {
			ctx := context.Background()
			account := newOAuthProbeAccount(433, true)
			tickets := newMemoryTurnStateTicketStore()
			upstream := &recordingTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403); _, _ = w.Write([]byte(body)) })}
			svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
			policy := enableTurnStateProbePolicy(t, svc)
			require.NoError(t, tickets.Put(ctx, TurnStateTicketRecord{AccountID: account.ID, State: "valid-old-state", Status: "holding", PolicyRevision: policy.Revision, UpdatedAt: time.Now(), RecheckAt: time.Now().Add(-time.Second)}))
			require.Error(t, svc.ProbeAccount(ctx, account.ID))
			rec, err := tickets.Get(ctx, account.ID)
			require.NoError(t, err)
			require.Equal(t, "skipped", rec.Status)
			require.Empty(t, rec.State)
			require.Contains(t, rec.LastError, "TURN_STATE_PROBE_ACCOUNT_FORBIDDEN")
			require.False(t, svc.ticketDue(ctx, account.ID, policy, time.Now().Add(24*time.Hour)))
		})
	}
}

func TestTurnStateProbeForbiddenDiagnosticsDoNotLeakBody(t *testing.T) {
	for _, tc := range []struct {
		name, body, kind, class string
		terminal                bool
	}{
		{"html", "<html>secret-token account_disabled</html>", "html", "unknown", false},
		{"policy", `{"error":{"code":"permission_denied","message":"account_disabled secret-token"}}`, "json", "unknown", false},
		{"credentials", `{"error":{"code":"token_revoked","message":"secret-token"}}`, "json", "credentials", true},
		{"workspace", `{"error":{"code":"deactivated_workspace","message":"secret-token"}}`, "json", "account_access_state", true},
		{"malformed", `{"error":{"code":"account_disabled"},`, "other", "unknown", false},
		{"oversized", `{"error":{"code":"account_disabled","message":"` + strings.Repeat("x", 17000) + `"}}`, "other", "unknown", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.body))}
			resp.Header.Set("X-Request-Id", "req-test-123")
			resp.Header.Set("Openai-Request-Id", "req-test-456")
			resp.Header.Set("Cf-Ray", "1234-SIN")
			resp.Header.Set("Content-Type", "secret-token")
			diagnostic, terminal := inspectTurnStateProbeForbidden(resp)
			require.Equal(t, tc.terminal, terminal)
			require.Contains(t, diagnostic, "body_kind="+tc.kind)
			require.Contains(t, diagnostic, "error_class="+tc.class)
			require.Contains(t, diagnostic, `request_id="req-test-123"`)
			require.Contains(t, diagnostic, `openai_request_id="req-test-456"`)
			require.Contains(t, diagnostic, `cf_ray="1234-SIN"`)
			require.NotContains(t, diagnostic, "secret-token")
			require.NotContains(t, diagnostic, tc.body)
		})
	}
	require.Equal(t, "redacted", turnStateProbeDiagnosticID("Bearer secret-token"))
	require.Equal(t, "redacted", turnStateProbeDiagnosticID(strings.Repeat("x", 129)))
	require.Equal(t, "redacted", turnStateProbeDiagnosticID("request\nforged-log"))
}

func TestTurnStateProbeForbiddenBackoffSurvivesRPMDeferral(t *testing.T) {
	ctx := context.Background()
	account := newOAuthProbeAccount(434, true)
	tickets := newMemoryTurnStateTicketStore()
	upstream := &recordingTurnStateHTTPUpstream{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) })}
	svc := newTurnStateProbeServiceForTest(t, account, tickets, upstream)
	policy := enableTurnStateProbePolicy(t, svc)
	require.Error(t, svc.ProbeAccount(ctx, account.ID))
	rec, err := tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	rec.RecheckAt = time.Now().Add(-time.Second)
	require.NoError(t, tickets.Put(ctx, *rec))
	tickets.rpm["global"] = policy.RPM
	require.ErrorIs(t, svc.ProbeAccount(ctx, account.ID), ErrTurnStateProbeRateLimited)
	rec, err = tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	rec.RecheckAt = time.Now().Add(-time.Second)
	require.NoError(t, tickets.Put(ctx, *rec))
	tickets.rpm = map[string]int{}
	require.Error(t, svc.ProbeAccount(ctx, account.ID))
	rec, err = tickets.Get(ctx, account.ID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(4*time.Minute), rec.RecheckAt, time.Second)
}
