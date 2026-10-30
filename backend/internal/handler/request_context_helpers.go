package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func shouldSkipOpsClientClosed(c *gin.Context, ops *service.OpsService, status int) bool {
	if status != statusClientClosedRequest || ops == nil {
		return false
	}
	if !ops.OpsAdvancedSettingsSnapshot().IgnoreContextCanceled {
		return false
	}
	return !hasOpsUpstreamErrorContext(c)
}
