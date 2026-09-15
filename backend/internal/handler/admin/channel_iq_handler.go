package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChannelIQHandler struct {
	svc *service.ChannelIQService
}

func NewChannelIQHandler(svc *service.ChannelIQService) *ChannelIQHandler {
	return &ChannelIQHandler{svc: svc}
}

func (h *ChannelIQHandler) Overview(c *gin.Context) {
	out, err := h.svc.GetOverview(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, out)
}

type channelIQSettingsRequest struct {
	GroupIDs        []int64 `json:"group_ids"`
	AutoEnabled     *bool   `json:"auto_enabled"`
	IntervalSeconds *int    `json:"interval_seconds"`
	Prompt          *string `json:"prompt"`
	Model           *string `json:"model"`
}

func (h *ChannelIQHandler) UpdateSettings(c *gin.Context) {
	var req channelIQSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	current, err := h.svc.GetOverview(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	next := current.Settings
	if req.GroupIDs != nil {
		next.GroupIDs = req.GroupIDs
	}
	if req.AutoEnabled != nil {
		next.AutoEnabled = *req.AutoEnabled
	}
	if req.IntervalSeconds != nil {
		next.IntervalSeconds = *req.IntervalSeconds
	}
	if req.Prompt != nil {
		next.Prompt = *req.Prompt
	}
	if req.Model != nil {
		next.Model = *req.Model
	}
	saved, err := h.svc.SaveSettings(c.Request.Context(), next)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, saved)
}

func (h *ChannelIQHandler) RunAll(c *gin.Context) {
	if err := h.svc.RunAll(c.Request.Context()); err != nil {
		if errors.Is(err, service.ErrChannelIQBusy) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"started": true})
}

func (h *ChannelIQHandler) RunOne(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.RunOne(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrChannelIQBusy) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"started": true})
}

func (h *ChannelIQHandler) Exclude(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.ExcludeAccount(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"excluded": true})
}

func (h *ChannelIQHandler) Restore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.RestoreAccount(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"restored": true})
}
