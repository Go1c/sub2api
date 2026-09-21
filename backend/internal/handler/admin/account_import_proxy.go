package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) GetImportProxyDefault(c *gin.Context) {
	binding, err := service.ResolveCodexImportProxyDefault(c.Request.Context(), h.adminService)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, binding)
}
