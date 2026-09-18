package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) ImportCodex2FA(c *gin.Context) {
	if h.codexLoginService == nil || !h.codexLoginService.Available() {
		response.Error(c, http.StatusServiceUnavailable, "请配置登录 Worker 与固定 TOTP_ENCRYPTION_KEY")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	var req struct {
		Documents []string `json:"documents"`
		service.CodexLoginOptions
	}
	if c.ShouldBindJSON(&req) != nil || len(req.Documents) == 0 || len(req.Documents) > 20 {
		response.BadRequest(c, "导入参数无效，最多 20 个文件")
		return
	}
	ids, failures, err := h.codexLoginService.Import(c.Request.Context(), req.Documents, req.CodexLoginOptions)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Accepted(c, gin.H{"job_ids": ids, "errors": failures})
}

func (h *AccountHandler) ListCodex2FAJobs(c *gin.Context) {
	if h.codexLoginService == nil {
		response.Success(c, gin.H{"available": false, "jobs": []any{}})
		return
	}
	jobs, err := h.codexLoginService.Jobs(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "读取登录任务失败")
		return
	}
	response.Success(c, gin.H{"available": h.codexLoginService.Available(), "jobs": jobs})
}

func (h *AccountHandler) RetryCodex2FA(c *gin.Context) {
	if h.codexLoginService == nil {
		response.BadRequest(c, "登录功能未配置")
		return
	}
	id, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "任务 ID 无效")
		return
	}
	if err = h.codexLoginService.Retry(c.Request.Context(), id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"queued": true})
}
