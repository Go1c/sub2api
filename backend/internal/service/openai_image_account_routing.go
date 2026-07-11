package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

const (
	OpenAIImageAccountRoutingOff     = "off"
	OpenAIImageAccountRoutingAudit   = "audit"
	OpenAIImageAccountRoutingEnforce = "enforce"
)

func normalizeOpenAIImageAccountRoutingMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case OpenAIImageAccountRoutingOff:
		return OpenAIImageAccountRoutingOff
	case OpenAIImageAccountRoutingAudit:
		return OpenAIImageAccountRoutingAudit
	case "", OpenAIImageAccountRoutingEnforce:
		return OpenAIImageAccountRoutingEnforce
	default:
		return OpenAIImageAccountRoutingEnforce
	}
}

func (s *OpenAIGatewayService) openAIImageAccountRoutingMode() string {
	if s == nil || s.cfg == nil {
		return OpenAIImageAccountRoutingEnforce
	}
	return normalizeOpenAIImageAccountRoutingMode(s.cfg.Gateway.OpenAIImageAccountRoutingMode)
}

func (s *OpenAIGatewayService) isOpenAIAccountImageRoutingCompatible(account *Account, requiresImageGeneration bool) bool {
	if account == nil {
		return false
	}
	if account.IsGrok() {
		return !requiresImageGeneration
	}
	if !account.IsOpenAI() {
		return false
	}
	if account.SupportsOpenAIImageGenerationRouting(requiresImageGeneration) {
		return true
	}

	mode := s.openAIImageAccountRoutingMode()
	if mode == OpenAIImageAccountRoutingOff {
		return true
	}

	logFields := []any{
		"account_id", account.ID,
		"account_name", account.Name,
		"routing_mode", account.OpenAIImageGenerationRoutingMode(),
		"requires_image_generation", requiresImageGeneration,
		"enforcement_mode", mode,
	}
	if mode == OpenAIImageAccountRoutingAudit {
		slog.Warn("openai image account routing would block account", logFields...)
		return true
	}
	slog.Debug("openai image account routing skipped account", logFields...)
	return false
}

func newOpenAIImageAccountRoutingFailoverError(account *Account, requiresImageGeneration bool) *UpstreamFailoverError {
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	var message string
	if requiresImageGeneration {
		message = "selected OpenAI account is not enabled for image generation"
	} else {
		message = "selected OpenAI account is reserved for image generation"
	}
	body := []byte(fmt.Sprintf(`{"error":{"type":"unavailable_error","message":%q,"account_id":%d}}`, message, accountID))
	return &UpstreamFailoverError{
		StatusCode:   http.StatusServiceUnavailable,
		ResponseBody: body,
	}
}
