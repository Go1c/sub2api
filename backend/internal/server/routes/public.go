package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers read-only endpoints that are intentionally
// reachable without JWT authentication.
func RegisterPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	public := v1.Group("/public")
	{
		monitors := public.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
	}
}
