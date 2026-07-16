package service

import "strings"

const (
	OpenAIImageGenerationRoutingMixed     = "mixed"
	OpenAIImageGenerationRoutingImageOnly = "image_only"
	OpenAIImageGenerationRoutingTextOnly  = "text_only"

	openAIImageGenerationRoutingExtraKey = "openai_image_generation_routing"
)

func normalizeOpenAIImageGenerationRoutingMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case OpenAIImageGenerationRoutingImageOnly, "image-only", "image":
		return OpenAIImageGenerationRoutingImageOnly
	case OpenAIImageGenerationRoutingTextOnly, "text-only", "text":
		return OpenAIImageGenerationRoutingTextOnly
	case "", OpenAIImageGenerationRoutingMixed:
		return OpenAIImageGenerationRoutingMixed
	default:
		return OpenAIImageGenerationRoutingMixed
	}
}

func (a *Account) OpenAIImageGenerationRoutingMode() string {
	if a == nil || a.Platform != PlatformOpenAI {
		return OpenAIImageGenerationRoutingMixed
	}
	if mode, ok := stringOverrideFromMap(a.Extra, openAIImageGenerationRoutingExtraKey); ok {
		return normalizeOpenAIImageGenerationRoutingMode(mode)
	}
	openaiConfig, _ := a.Extra[PlatformOpenAI].(map[string]any)
	if mode, ok := stringOverrideFromMap(openaiConfig, openAIImageGenerationRoutingExtraKey); ok {
		return normalizeOpenAIImageGenerationRoutingMode(mode)
	}
	if enabled, ok := a.Extra["openai_image_generation_enabled"].(bool); ok && !enabled {
		return OpenAIImageGenerationRoutingTextOnly
	}
	if enabled, ok := openaiConfig["openai_image_generation_enabled"].(bool); ok && !enabled {
		return OpenAIImageGenerationRoutingTextOnly
	}
	return OpenAIImageGenerationRoutingMixed
}

func (a *Account) SupportsOpenAIImageGenerationRouting(requiresImageGeneration bool) bool {
	switch a.OpenAIImageGenerationRoutingMode() {
	case OpenAIImageGenerationRoutingImageOnly:
		return requiresImageGeneration
	case OpenAIImageGenerationRoutingTextOnly:
		return !requiresImageGeneration
	default:
		return true
	}
}
