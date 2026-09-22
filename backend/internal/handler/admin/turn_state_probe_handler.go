package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type TurnStateProbeHandler struct {
	svc *service.TurnStateProbeService
}

func NewTurnStateProbeHandler(svc *service.TurnStateProbeService) *TurnStateProbeHandler {
	return &TurnStateProbeHandler{svc: svc}
}

func (h *TurnStateProbeHandler) Overview(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "turn-state probe unavailable")
		return
	}
	out, err := h.svc.GetOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

type turnStateImportBatchRuntimeRequest struct {
	Suspended *bool `json:"suspended"`
}

func (h *TurnStateProbeHandler) SetImportBatchRuntime(c *gin.Context) {
	var req turnStateImportBatchRuntimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Suspended == nil {
		response.Error(c, http.StatusBadRequest, "suspended is required")
		return
	}
	service.SetOpenAIImportBatchRuntimeSuspended(*req.Suspended)
	response.Success(c, gin.H{"import_batch_runtime_suspended": service.OpenAIImportBatchRuntimeSuspended()})
}

func (h *TurnStateProbeHandler) UpdatePolicy(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "turn-state probe unavailable")
		return
	}
	req := service.TurnStateProbePolicy{OverloadThreshold: service.DefaultTurnStateProbePolicy().OverloadThreshold}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := h.svc.SavePolicy(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, saved)
}

type turnStateProbeEnabledRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *TurnStateProbeHandler) SetEnabled(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "turn-state probe unavailable")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	var req turnStateProbeEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Enabled == nil {
		response.Error(c, http.StatusBadRequest, "enabled is required")
		return
	}
	if err := h.svc.SetAccountEnabled(c.Request.Context(), id, *req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": *req.Enabled})
}

func (h *TurnStateProbeHandler) RunOne(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "turn-state probe unavailable")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.RunOne(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"started": true})
}

func (h *TurnStateProbeHandler) Clear(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "turn-state probe unavailable")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.ClearTicket(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"cleared": true})
}
