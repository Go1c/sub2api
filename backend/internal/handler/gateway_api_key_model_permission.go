package handler

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func apiKeyModelPermissionDenied(apiKey *service.APIKey, model string) bool {
	return apiKey != nil && !apiKey.AllowsModel(model)
}

func apiKeyModelPermissionDeniedMessage(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "当前密钥无权限使用该模型"
	}
	return fmt.Sprintf("当前密钥无权限使用模型：%s", model)
}

func apiKeyAllowedModelSet(apiKey *service.APIKey) map[string]struct{} {
	if apiKey == nil {
		return nil
	}
	allowedModels := service.NormalizeAPIKeyAllowedModels(apiKey.AllowedModels)
	if len(allowedModels) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(allowedModels))
	for _, model := range allowedModels {
		out[model] = struct{}{}
	}
	return out
}

func modelAllowedBySet(allowed map[string]struct{}, model string) bool {
	if len(allowed) == 0 {
		return true
	}
	model = strings.TrimSpace(model)
	if _, ok := allowed[model]; ok {
		return true
	}
	if !strings.HasPrefix(model, "models/") {
		_, ok := allowed["models/"+model]
		return ok
	}
	if strings.HasPrefix(model, "models/") {
		_, ok := allowed[strings.TrimPrefix(model, "models/")]
		return ok
	}
	return false
}

func filterGeminiModelsByAllowedSet(models []gemini.Model, allowed map[string]struct{}) []gemini.Model {
	if len(allowed) == 0 {
		return models
	}
	out := make([]gemini.Model, 0, len(models))
	for _, model := range models {
		if modelAllowedBySet(allowed, model.Name) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityGeminiModelsByAllowedSet(models []antigravity.GeminiModel, allowed map[string]struct{}) []antigravity.GeminiModel {
	if len(allowed) == 0 {
		return models
	}
	out := make([]antigravity.GeminiModel, 0, len(models))
	for _, model := range models {
		if modelAllowedBySet(allowed, model.Name) {
			out = append(out, model)
		}
	}
	return out
}
