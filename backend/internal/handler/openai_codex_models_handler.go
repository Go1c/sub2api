package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// CodexModels serves the Codex models manifest for Codex clients.
//
// Codex CLI and the Codex desktop app refresh their model picker from
// GET {base_url}/models?client_version=... (custom provider mode) or
// GET /backend-api/codex/models (chatgpt_base_url mode). Both routes land
// here. The manifest is proxied verbatim using either a schedulable OAuth
// account's ChatGPT credentials or an API key account's custom
// OpenAI-compatible upstream, so clients pointed at the gateway see the
// account's live model catalog instead of a frozen local cache.
func (h *OpenAIGatewayHandler) CodexModels(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		h.errorResponse(c, http.StatusUnauthorized, "invalid_request_error", "API key group is required")
		return
	}
	if apiKey.Group.Platform != service.PlatformOpenAI {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Codex models manifest is only available for OpenAI groups")
		return
	}

	var manifest *service.CodexModelsManifest
	var fetchErr error
	excludedAccountIDs := make(map[int64]struct{}, 1)
	for attempt := 0; attempt < 2; attempt++ {
		account, err := h.gatewayService.SelectAccountForModelWithExclusions(c.Request.Context(), apiKey.GroupID, "", "", excludedAccountIDs)
		if err != nil {
			if fetchErr != nil {
				h.errorResponse(c, infraerrors.Code(fetchErr), "upstream_error", infraerrors.Message(fetchErr))
				return
			}
			h.errorResponse(c, http.StatusServiceUnavailable, "upstream_error", "No available OpenAI accounts")
			return
		}

		manifest, fetchErr = h.gatewayService.FetchCodexModelsManifest(c.Request.Context(), account, c.Query("client_version"), c.GetHeader("If-None-Match"))
		if fetchErr == nil {
			break
		}
		if attempt > 0 || account.ID <= 0 || !shouldFailoverCodexModelsAccount(fetchErr) {
			h.errorResponse(c, infraerrors.Code(fetchErr), "upstream_error", infraerrors.Message(fetchErr))
			return
		}
		excludedAccountIDs[account.ID] = struct{}{}
	}

	if manifest.ETag != "" {
		c.Header("ETag", manifest.ETag)
	}
	if manifest.NotModified {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json", manifest.Body)
}

func shouldFailoverCodexModelsAccount(err error) bool {
	switch infraerrors.Reason(err) {
	case "OPENAI_CODEX_MODELS_TOKEN_MISSING", "OPENAI_CODEX_MODELS_UPSTREAM_NOT_CONFIGURED":
		return true
	default:
		return false
	}
}
