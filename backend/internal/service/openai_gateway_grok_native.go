package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const grokNativeResponsesJSONReadLimit int64 = 4 << 20

// DoGrokNativeResponsesJSON posts a non-streaming Responses body to xAI and
// returns the raw JSON. Unlike forwardGrokResponses it does not write to the
// client, so callers can extract tool sources and emit a unified payload.
func (s *OpenAIGatewayService) DoGrokNativeResponsesJSON(ctx context.Context, c *gin.Context, account *Account, body []byte) ([]byte, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, errors.New("http upstream not configured")
	}
	if account == nil {
		return nil, errors.New("account is required")
	}
	if !account.IsGrok() {
		return nil, errors.New("grok account required")
	}

	token, _, err := s.getRequestCredential(ctx, c, account)
	if err != nil {
		return nil, err
	}

	upstreamModel := grokDefaultResponsesModel
	if json.Valid(body) {
		if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); model != "" {
			upstreamModel = model
		} else if patched, patchErr := sjson.SetBytes(body, "model", grokDefaultResponsesModel); patchErr == nil {
			body = patched
		}
		stripped, stripErr := stripRejectedGrokSearchInclude(body)
		if stripErr != nil {
			return nil, stripErr
		}
		body = stripped
	}
	ctx = withGrokTeamRateLimitModel(ctx, upstreamModel)

	targetURL, err := buildGrokResponsesURL(account, s.cfg)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build grok responses request: %w", err)
	}
	upstreamReq.Header.Set("Authorization", "Bearer "+token)
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "application/json")
	if account.IsGrokOAuth() {
		applyGrokCLIHeaders(upstreamReq.Header)
	} else {
		upstreamReq.Header.Set("User-Agent", grokUpstreamUserAgent)
	}
	account.ApplyHeaderOverrides(upstreamReq.Header)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway, Reason: GatewayFailureReason("grok_search_transport")}
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, grokNativeResponsesJSONReadLimit))
	if readErr != nil {
		return nil, &UpstreamFailoverError{
			StatusCode: http.StatusBadGateway,
			Reason:     GatewayFailureReason("grok_search_read"),
		}
	}
	if resp.StatusCode >= 400 {
		s.handleGrokAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBytes)
		if s.shouldFailoverGrokUpstreamError(resp.StatusCode, respBytes) {
			retryable, retryDelay, retryDeadline, retryMax := grokSameAccountRetryMetadata(account, resp.StatusCode, respBytes)
			return nil, &UpstreamFailoverError{
				StatusCode:               resp.StatusCode,
				ResponseBody:             respBytes,
				ResponseHeaders:          resp.Header.Clone(),
				RetryableOnSameAccount:   retryable,
				RequestScopedTransient:   retryable && resp.StatusCode == http.StatusTooManyRequests,
				SameAccountRetryDelay:    retryDelay,
				SameAccountRetryDeadline: retryDeadline,
				SameAccountRetryMax:      retryMax,
			}
		}
		msg := string(respBytes)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("grok upstream %d: %s", resp.StatusCode, msg)
	}
	s.updateGrokUsageFromResponse(ctx, account, resp.Header, resp.StatusCode)
	return respBytes, nil
}
