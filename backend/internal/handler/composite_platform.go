package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func ensureCompositeTargetPlatform(c *gin.Context, apiKey *service.APIKey, model string) {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return
	}
	if _, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
		return
	}
	if platform, ok := service.DetectModelPlatform(model); ok {
		c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), platform))
	}
}

func compositeTargetPlatformAllowed(c *gin.Context, apiKey *service.APIKey, model string, allowed ...string) bool {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return true
	}
	ensureCompositeTargetPlatform(c, apiKey, model)
	platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	if !ok {
		return false
	}
	for _, allowedPlatform := range allowed {
		if platform == allowedPlatform {
			return true
		}
	}
	return false
}

func compositeTargetPlatformResolved(c *gin.Context, apiKey *service.APIKey, model string) bool {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return true
	}
	ensureCompositeTargetPlatform(c, apiKey, model)
	_, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	return ok
}

func openAICompatibleTextTargetAllowed(c *gin.Context, apiKey *service.APIKey, model string) bool {
	return compositeTargetPlatformAllowed(c, apiKey, model,
		service.PlatformOpenAI, service.PlatformGrok,
		service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek)
}

// isResponsesWebSocketCompositePlatform 限定 composite 分组在 Responses WebSocket
// 上可服务的目标平台。CN 供应商（kimi/zhipu/deepseek）刻意排除：其账号无法通过
// WSv2 ingress 的 transport 过滤，且 WS HTTP 桥没有面向 CN 的 Responses 转换，
// 放行只会把明确的策略拒绝变成误导性的 "no available account"。
func isResponsesWebSocketCompositePlatform(platform string) bool {
	switch platform {
	case service.PlatformOpenAI, service.PlatformGrok:
		return true
	default:
		return false
	}
}
