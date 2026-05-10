package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *OpsHandler) requireUserRequestMonitorService(c *gin.Context) (*service.OpsUserRequestMonitorService, bool) {
	if h == nil || h.userRequestMonitorService == nil {
		response.Error(c, http.StatusServiceUnavailable, "User request monitor service not available")
		return nil, false
	}
	return h.userRequestMonitorService, true
}

// CreateUserRequestMonitor creates an admin-only targeted request body monitor.
// POST /api/v1/admin/ops/user-request-monitors
func (h *OpsHandler) CreateUserRequestMonitor(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req service.OpsCreateUserRequestMonitorInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	req.CreatedBy = subject.UserID

	monitor, err := svc.CreateMonitor(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, monitor)
}

// ListUserRequestMonitors lists targeted user request monitors.
// GET /api/v1/admin/ops/user-request-monitors
func (h *OpsHandler) ListUserRequestMonitors(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	filter := &service.OpsUserRequestMonitorFilter{
		Status:    strings.TrimSpace(c.Query("status")),
		UserQuery: strings.TrimSpace(c.Query("user_query")),
		Page:      page,
		PageSize:  pageSize,
	}
	items, total, err := svc.ListMonitors(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

// StopUserRequestMonitor stops an active monitor.
// POST /api/v1/admin/ops/user-request-monitors/:id/stop
func (h *OpsHandler) StopUserRequestMonitor(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	id, ok := parsePositiveOpsID(c, "id")
	if !ok {
		return
	}
	monitor, err := svc.StopMonitor(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, monitor)
}

// ListUserRequestCaptures lists captures without raw body content.
// GET /api/v1/admin/ops/user-request-monitors/:id/captures
func (h *OpsHandler) ListUserRequestCaptures(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	monitorID, ok := parsePositiveOpsID(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	filter := &service.OpsUserRequestCaptureFilter{
		MonitorID: monitorID,
		Page:      page,
		PageSize:  pageSize,
	}
	items, total, err := svc.ListCaptures(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, filter.Page, filter.PageSize)
}

// GetUserRequestCapture returns one capture including the raw body.
// GET /api/v1/admin/ops/user-request-monitors/:id/captures/:capture_id
func (h *OpsHandler) GetUserRequestCapture(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	monitorID, ok := parsePositiveOpsID(c, "id")
	if !ok {
		return
	}
	captureID, ok := parsePositiveOpsID(c, "capture_id")
	if !ok {
		return
	}
	capture, err := svc.GetCapture(c.Request.Context(), monitorID, captureID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, capture)
}

// DeleteUserRequestCapture immediately deletes one captured body.
// DELETE /api/v1/admin/ops/user-request-monitors/:id/captures/:capture_id
func (h *OpsHandler) DeleteUserRequestCapture(c *gin.Context) {
	svc, ok := h.requireUserRequestMonitorService(c)
	if !ok {
		return
	}
	monitorID, ok := parsePositiveOpsID(c, "id")
	if !ok {
		return
	}
	captureID, ok := parsePositiveOpsID(c, "capture_id")
	if !ok {
		return
	}
	if err := svc.DeleteCapture(c.Request.Context(), monitorID, captureID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func parsePositiveOpsID(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}
