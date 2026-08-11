package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers read-only endpoints that are intentionally
// reachable without JWT authentication.
func RegisterPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	if h.Setting != nil {
		desktop := v1.Group("/desktop")
		{
			desktop.GET("/config", h.Setting.GetLumioDesktopConfig)
		}
	}

	public := v1.Group("/public")
	{
		monitors := public.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
	}

	if h.ModelMarket != nil {
		modelMarket := v1.Group("/model-market")
		{
			modelMarket.GET("/public", h.ModelMarket.GetPublic)
		}
	}
}
