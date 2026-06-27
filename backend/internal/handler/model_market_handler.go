package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ModelMarketHandler struct {
	modelMarketService *service.ModelMarketService
}

func NewModelMarketHandler(modelMarketService *service.ModelMarketService) *ModelMarketHandler {
	return &ModelMarketHandler{modelMarketService: modelMarketService}
}

// GetPublic returns the public model market snapshot.
// GET /api/v1/model-market/public
func (h *ModelMarketHandler) GetPublic(c *gin.Context) {
	payload, err := h.modelMarketService.GetPublic(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

// GetAdmin returns model market config plus current channel-derived candidates.
// GET /api/v1/admin/settings/model-market
func (h *ModelMarketHandler) GetAdmin(c *gin.Context) {
	payload, err := h.modelMarketService.GetAdmin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

// UpdateAdmin saves model market display configuration.
// PUT /api/v1/admin/settings/model-market
func (h *ModelMarketHandler) UpdateAdmin(c *gin.Context) {
	var req service.ModelMarketConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	payload, err := h.modelMarketService.SetConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}
