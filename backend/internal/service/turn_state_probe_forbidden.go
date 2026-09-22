package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Only revive skips written by the old, unclassified HTTP 403 branch. New
// explicit credential/account rejections have a distinct reason and stay stopped.
func turnStateProbeSkipBlocks(ticket *TurnStateTicketRecord, now time.Time) bool {
	if ticket == nil || ticket.Status != turnStateProbeStatusSkipped {
		return false
	}
	if !strings.Contains(ticket.LastError, `reason="TURN_STATE_PROBE_FORBIDDEN"`) {
		return true
	}
	return ticket.UpdatedAt.Add(turnStateProbeForbiddenDelay(1)).After(now)
}

func turnStateProbeForbiddenDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	if failures >= 5 {
		return 30 * time.Second
	}
	return 2 * time.Second << (failures - 1)
}

// Inspect a bounded response, never persist or log free-form body text: error
// messages can echo tokens, account data and other sensitive request fields.
func inspectTurnStateProbeForbidden(resp *http.Response) (string, bool) {
	const limit = 16 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	complete := err == nil && len(body) <= limit
	kind, class := "other", "unknown"
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]))
	switch contentType {
	case "application/json", "text/html", "text/plain":
	default:
		contentType = "other"
	}
	if complete && json.Valid(body) {
		kind = "json"
		if isOpenAIUpstreamAccessStateError("", body) {
			class = "account_access_state"
		} else if openAIStreamCredentialAuthFailure(body) {
			class = "credentials"
		}
	} else if contentType == "text/html" || strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(body))), "<!doctype html") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(body))), "<html") {
		kind = "html"
	}
	diagnostic := fmt.Sprintf("status=403 content_type=%q body_kind=%s error_class=%s body_complete=%t request_id=%q openai_request_id=%q cf_ray=%q cf_challenge=%t",
		contentType, kind, class, complete,
		turnStateProbeDiagnosticID(resp.Header.Get("x-request-id")),
		turnStateProbeDiagnosticID(resp.Header.Get("openai-request-id")),
		turnStateProbeDiagnosticID(resp.Header.Get("cf-ray")),
		resp.Header.Get("cf-mitigated") == "challenge")
	return diagnostic, class != "unknown"
}

func turnStateProbeDiagnosticID(value string) string {
	if len(value) > 128 {
		return "redacted"
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-_.:", c)) {
			return "redacted"
		}
	}
	return value
}
