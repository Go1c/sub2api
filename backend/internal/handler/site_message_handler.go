package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SiteMessageHandler handles user site-message operations.
type SiteMessageHandler struct {
	siteMessageService *service.SiteMessageService
}

func NewSiteMessageHandler(siteMessageService *service.SiteMessageService) *SiteMessageHandler {
	return &SiteMessageHandler{siteMessageService: siteMessageService}
}

// ListInbox handles listing received site messages.
// GET /api/v1/site-messages/inbox
func (h *SiteMessageHandler) ListInbox(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	params := siteMessagePagination(c)
	items, page, err := h.siteMessageService.ListInbox(c.Request.Context(), userID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.SiteMessagesFromService(items), page.Total, page.Page, page.PageSize)
}

// ListSent handles listing sent site messages.
// GET /api/v1/site-messages/sent
func (h *SiteMessageHandler) ListSent(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	params := siteMessagePagination(c)
	items, page, err := h.siteMessageService.ListSent(c.Request.Context(), userID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.SiteMessagesFromService(items), page.Total, page.Page, page.PageSize)
}

// UnreadCount handles counting unread received site messages.
// GET /api/v1/site-messages/unread-count
func (h *SiteMessageHandler) UnreadCount(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	count, err := h.siteMessageService.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"count": count})
}

// ResolveRecipient resolves an exact email or numeric user ID for regular users.
// GET /api/v1/site-messages/recipient/resolve?query=...
func (h *SiteMessageHandler) ResolveRecipient(c *gin.Context) {
	if _, ok := siteMessageUserID(c); !ok {
		return
	}
	query := strings.TrimSpace(firstQuery(c, "query", "q"))
	recipient, err := h.siteMessageService.ResolveRecipient(c.Request.Context(), query, false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageRecipientFromService(recipient))
}

// Get handles reading a visible message; received unread messages are marked read.
// GET /api/v1/site-messages/:id
func (h *SiteMessageHandler) Get(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	messageID, ok := parsePositiveID(c, "id", "Invalid site message ID")
	if !ok {
		return
	}
	message, err := h.siteMessageService.Get(c.Request.Context(), userID, messageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageFromService(message))
}

// Send handles composing a new site message.
// POST /api/v1/site-messages
func (h *SiteMessageHandler) Send(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	var req dto.CreateSiteMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	message, err := h.siteMessageService.Send(c.Request.Context(), service.SendSiteMessageInput{
		SenderID:       userID,
		RecipientQuery: req.Recipient,
		Subject:        req.Subject,
		Content:        req.Content,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageFromService(message))
}

// Reply handles replying to a visible site message.
// POST /api/v1/site-messages/:id/reply
func (h *SiteMessageHandler) Reply(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	messageID, ok := parsePositiveID(c, "id", "Invalid site message ID")
	if !ok {
		return
	}
	var req dto.ReplySiteMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	message, err := h.siteMessageService.Reply(c.Request.Context(), userID, messageID, req.Content)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageFromService(message))
}

// MarkRead handles explicitly marking a received message read.
// POST /api/v1/site-messages/:id/read
func (h *SiteMessageHandler) MarkRead(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	messageID, ok := parsePositiveID(c, "id", "Invalid site message ID")
	if !ok {
		return
	}
	if err := h.siteMessageService.MarkRead(c.Request.Context(), userID, messageID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func siteMessageUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return 0, false
	}
	return subject.UserID, true
}

func siteMessagePagination(c *gin.Context) pagination.PaginationParams {
	page, pageSize := response.ParsePagination(c)
	return pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
}

func parsePositiveID(c *gin.Context, param, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, message)
		return 0, false
	}
	return id, true
}

func firstQuery(c *gin.Context, keys ...string) string {
	for _, key := range keys {
		if value := c.Query(key); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
