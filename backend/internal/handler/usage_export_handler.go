package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Export handles incremental usage dump for downstream resellers.
// GET /api/v1/usage/export
func (h *UsageHandler) Export(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	afterID, since, limit, errMsg := parseUsageExportQuery(c)
	if errMsg != "" {
		response.BadRequest(c, errMsg)
		return
	}

	result, err := h.usageService.Export(c.Request.Context(), subject.UserID, afterID, since, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UsageExportFromService(result))
}

func parseUsageExportQuery(c *gin.Context) (afterID int64, since *time.Time, limit int, errMsg string) {
	limit = service.UsageExportDefaultLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, nil, 0, "Invalid limit"
		}
		limit = parsed
	}

	if raw := strings.TrimSpace(c.Query("after_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, nil, 0, "Invalid after_id"
		}
		afterID = parsed
	}

	if raw := strings.TrimSpace(c.Query("since")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, raw)
		}
		if err != nil {
			return 0, nil, 0, "Invalid since, use RFC3339"
		}
		if time.Since(parsed) > service.UsageExportMaxLookback {
			return 0, nil, 0, "since cannot be more than 24 hours ago"
		}
		since = &parsed
	}

	if afterID > 0 {
		since = nil
	}
	if afterID <= 0 && since == nil {
		return 0, nil, 0, "after_id or since is required"
	}
	return afterID, since, limit, ""
}
