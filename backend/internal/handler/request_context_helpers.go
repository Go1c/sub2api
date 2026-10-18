package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func effectiveAPIKeyPlatform(c *gin.Context, apiKey *service.APIKey) string {
	if c != nil && c.Request != nil {
		if platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
			return platform
		}
	}
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func stampForwardRequestedReasoningEffort(result *service.ForwardResult, requested *string) {
	if result == nil || result.RequestedReasoningEffort != nil {
		return
	}
	result.RequestedReasoningEffort = requested
}

func clientRequestedModel(c *gin.Context, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if c == nil || c.Request == nil {
		return fallback
	}
	if model, ok := service.RequestedPublicModelFromContext(c.Request.Context()); ok {
		return model
	}
	return fallback
}

func clientRequestedUsageFields(c *gin.Context, mapping service.ChannelMappingResult, fallbackModel, upstreamModel string) service.ChannelUsageFields {
	return mapping.ToUsageFields(clientRequestedModel(c, fallbackModel), upstreamModel)
}

func openAIChannelForwardModel(mapping service.ChannelMappingResult, requestedModel string) string {
	if mapping.Mapped {
		if mappedModel := strings.TrimSpace(mapping.MappedModel); mappedModel != "" {
			return mappedModel
		}
	}
	return requestedModel
}

func findBlockedCyberSessionKey(ctx context.Context, gatewayService *service.OpenAIGatewayService, apiKeyID int64, c *gin.Context, body []byte) string {
	if gatewayService == nil {
		return ""
	}
	key := service.CyberSessionBlockKey(apiKeyID, c, body)
	if key == "" || !gatewayService.IsCyberSessionBlocked(ctx, key) {
		return ""
	}
	return key
}

func shouldSkipOpsClientClosed(c *gin.Context, ops *service.OpsService, status int) bool {
	if status != statusClientClosedRequest || ops == nil {
		return false
	}
	if !ops.OpsAdvancedSettingsSnapshot().IgnoreContextCanceled {
		return false
	}
	return !hasOpsUpstreamErrorContext(c)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
