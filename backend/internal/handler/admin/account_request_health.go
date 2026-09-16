package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type batchRequestHealthRequest struct {
	AccountIDs []int64 `json:"account_ids"`
	Window     int     `json:"window"`
}

// GetBatchRequestHealth returns the last-N request health bars for the current account page.
// POST /api/v1/admin/accounts/request-health/batch
func (h *AccountHandler) GetBatchRequestHealth(c *gin.Context) {
	var req batchRequestHealthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 || h.requestHealth == nil {
		response.Success(c, gin.H{"items": []any{}})
		return
	}

	items, err := h.requestHealth.ListForAccounts(c.Request.Context(), accountIDs, req.Window)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}
