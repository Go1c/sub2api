package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProxyIPGroupHandler handles admin IP group CRUD.
type ProxyIPGroupHandler struct {
	adminService service.AdminService
}

func NewProxyIPGroupHandler(adminService service.AdminService) *ProxyIPGroupHandler {
	return &ProxyIPGroupHandler{adminService: adminService}
}

type CreateProxyIPGroupRequest struct {
	Name             string  `json:"name" binding:"required"`
	PerIPConcurrency int     `json:"per_ip_concurrency"`
	ProxyIDs         []int64 `json:"proxy_ids"`
}

type UpdateProxyIPGroupRequest struct {
	Name             string `json:"name"`
	PerIPConcurrency *int   `json:"per_ip_concurrency"`
}

type SetProxyIPGroupMembersRequest struct {
	ProxyIDs []int64 `json:"proxy_ids"`
}

// List GET /api/v1/admin/proxy-ip-groups
func (h *ProxyIPGroupHandler) List(c *gin.Context) {
	groups, err := h.adminService.ListProxyIPGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminProxyIPGroup, 0, len(groups))
	for i := range groups {
		if mapped := dto.ProxyIPGroupFromService(&groups[i]); mapped != nil {
			out = append(out, *mapped)
		}
	}
	response.Success(c, out)
}

// GetByID GET /api/v1/admin/proxy-ip-groups/:id
func (h *ProxyIPGroupHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid IP group ID")
		return
	}
	group, err := h.adminService.GetProxyIPGroup(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxyIPGroupFromService(group))
}

// Create POST /api/v1/admin/proxy-ip-groups
func (h *ProxyIPGroupHandler) Create(c *gin.Context) {
	var req CreateProxyIPGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	executeAdminIdempotentJSON(c, "admin.proxy_ip_groups.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		group, err := h.adminService.CreateProxyIPGroup(ctx, &service.CreateProxyIPGroupInput{
			Name:             strings.TrimSpace(req.Name),
			PerIPConcurrency: req.PerIPConcurrency,
			ProxyIDs:         req.ProxyIDs,
		})
		if err != nil {
			return nil, err
		}
		return dto.ProxyIPGroupFromService(group), nil
	})
}

// Update PUT /api/v1/admin/proxy-ip-groups/:id
func (h *ProxyIPGroupHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid IP group ID")
		return
	}
	var req UpdateProxyIPGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	group, err := h.adminService.UpdateProxyIPGroup(c.Request.Context(), id, &service.UpdateProxyIPGroupInput{
		Name:             strings.TrimSpace(req.Name),
		PerIPConcurrency: req.PerIPConcurrency,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxyIPGroupFromService(group))
}

// Delete DELETE /api/v1/admin/proxy-ip-groups/:id
func (h *ProxyIPGroupHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid IP group ID")
		return
	}
	if err := h.adminService.DeleteProxyIPGroup(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// SetMembers PUT /api/v1/admin/proxy-ip-groups/:id/members
func (h *ProxyIPGroupHandler) SetMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid IP group ID")
		return
	}
	var req SetProxyIPGroupMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	group, err := h.adminService.SetProxyIPGroupMembers(c.Request.Context(), id, req.ProxyIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxyIPGroupFromService(group))
}
