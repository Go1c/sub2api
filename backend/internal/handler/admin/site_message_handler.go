package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SiteMessageHandler handles admin site-message operations.
type SiteMessageHandler struct {
	siteMessageService *service.SiteMessageService
}

func NewSiteMessageHandler(siteMessageService *service.SiteMessageService) *SiteMessageHandler {
	return &SiteMessageHandler{siteMessageService: siteMessageService}
}

// SendToUser sends a site message to the selected user from the admin user table.
// POST /api/v1/admin/site-messages/users/:id
func (h *SiteMessageHandler) SendToUser(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	recipientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || recipientID <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req dto.AdminSendSiteMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	message, err := h.siteMessageService.AdminSendToUser(c.Request.Context(), service.AdminSendSiteMessageInput{
		AdminID:     subject.UserID,
		RecipientID: recipientID,
		Subject:     req.Subject,
		Content:     req.Content,
		SendEmail:   req.ShouldSendEmail(),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageFromService(message))
}

// SearchRecipients handles admin fuzzy recipient search.
// GET /api/v1/admin/site-messages/recipients?query=...
func (h *SiteMessageHandler) SearchRecipients(c *gin.Context) {
	query := strings.TrimSpace(firstNonEmpty(c.Query("query"), c.Query("q"), c.Query("search")))
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	recipients, err := h.siteMessageService.SearchRecipients(c.Request.Context(), query, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageRecipientsFromService(recipients))
}
