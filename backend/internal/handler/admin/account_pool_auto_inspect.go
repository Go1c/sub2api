package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetPoolAutoInspectConfig returns pool auto-inspect settings.
// GET /api/v1/admin/accounts/pool-auto-inspect/config
func (h *AccountHandler) GetPoolAutoInspectConfig(c *gin.Context) {
	if h.poolAutoInspect == nil {
		response.Success(c, service.AccountPoolAutoInspectStatus{
			AccountPoolAutoInspectConfig: *defaultPoolAutoInspectConfig(),
		})
		return
	}
	cfg, err := h.poolAutoInspect.GetConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get pool auto-inspect config")
		return
	}
	response.Success(c, cfg)
}

// UpdatePoolAutoInspectConfig updates pool auto-inspect settings.
// PUT /api/v1/admin/accounts/pool-auto-inspect/config
func (h *AccountHandler) UpdatePoolAutoInspectConfig(c *gin.Context) {
	if h.poolAutoInspect == nil {
		response.Error(c, http.StatusServiceUnavailable, "Pool auto-inspect is not available")
		return
	}
	var req service.AccountPoolAutoInspectConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	updated, err := h.poolAutoInspect.UpdateConfig(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, updated)
}

// RunPoolAutoInspect runs one pool auto-inspect cycle immediately.
// POST /api/v1/admin/accounts/pool-auto-inspect/run
func (h *AccountHandler) RunPoolAutoInspect(c *gin.Context) {
	if h.poolAutoInspect == nil {
		response.Error(c, http.StatusServiceUnavailable, "Pool auto-inspect is not available")
		return
	}
	status := h.poolAutoInspect.RunOnce(c.Request.Context(), true)
	response.Success(c, status)
}

func defaultPoolAutoInspectConfig() *service.AccountPoolAutoInspectConfig {
	return &service.AccountPoolAutoInspectConfig{
		Enabled:                 false,
		IntervalMinutes:         service.AccountPoolAutoInspectDefaultInterval,
		SuccessRateThreshold:    service.AccountPoolAutoInspectDefaultThreshold,
		MinSamples:              service.AccountPoolAutoInspectDefaultMinSamples,
		AddGroupIDs:             []int64{},
		RemoveModels:            []string{},
		OAuth401CooldownMinutes: 60,
	}
}
