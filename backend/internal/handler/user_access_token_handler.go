package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// UserAccessTokenHandler manages user opaque access tokens (JWT session only).
type UserAccessTokenHandler struct {
	svc *service.UserAccessTokenService
}

// NewUserAccessTokenHandler creates a UserAccessTokenHandler.
func NewUserAccessTokenHandler(svc *service.UserAccessTokenService) *UserAccessTokenHandler {
	return &UserAccessTokenHandler{svc: svc}
}

// CreateUserAccessTokenRequest is the create body.
type CreateUserAccessTokenRequest struct {
	Name          string `json:"name" binding:"required"`
	ExpiresInDays *int   `json:"expires_in_days"`
}

// UserAccessTokenResponse is the API response for list/create.
// Token (plaintext) is only set on create.
type UserAccessTokenResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Token       string  `json:"token,omitempty"`
	TokenPrefix string  `json:"token_prefix"`
	ExpiresAt   string  `json:"expires_at"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
	RevokedAt   *string `json:"revoked_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	Status      string  `json:"status"` // active | revoked | expired
}

// Create handles POST /api/v1/user/access-tokens
func (h *UserAccessTokenHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	// Access tokens must not create more access tokens.
	if middleware2.IsUserAccessTokenAuth(c) {
		response.Forbidden(c, "Access token cannot manage access tokens")
		return
	}

	var req CreateUserAccessTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: name is required")
		return
	}

	created, err := h.svc.Create(c.Request.Context(), subject.UserID, service.CreateUserAccessTokenInput{
		Name:          strings.TrimSpace(req.Name),
		ExpiresInDays: req.ExpiresInDays,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, toUserAccessTokenResponse(created, true))
}

// List handles GET /api/v1/user/access-tokens
func (h *UserAccessTokenHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if middleware2.IsUserAccessTokenAuth(c) {
		response.Forbidden(c, "Access token cannot manage access tokens")
		return
	}

	items, err := h.svc.List(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]UserAccessTokenResponse, 0, len(items))
	for i := range items {
		out = append(out, toUserAccessTokenResponse(&items[i], false))
	}
	response.Success(c, out)
}

// Revoke handles DELETE /api/v1/user/access-tokens/:id
func (h *UserAccessTokenHandler) Revoke(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if middleware2.IsUserAccessTokenAuth(c) {
		response.Forbidden(c, "Access token cannot manage access tokens")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid token id")
		return
	}

	if err := h.svc.Revoke(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"success": true})
}

func toUserAccessTokenResponse(t *service.UserAccessToken, includeToken bool) UserAccessTokenResponse {
	if t == nil {
		return UserAccessTokenResponse{}
	}
	resp := UserAccessTokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		TokenPrefix: t.TokenPrefix,
		ExpiresAt:   t.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
		Status:      userAccessTokenStatus(t),
	}
	if includeToken {
		resp.Token = t.Token
	}
	if t.LastUsedAt != nil {
		s := t.LastUsedAt.UTC().Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	if t.RevokedAt != nil {
		s := t.RevokedAt.UTC().Format(time.RFC3339)
		resp.RevokedAt = &s
	}
	return resp
}

func userAccessTokenStatus(t *service.UserAccessToken) string {
	if t == nil {
		return "expired"
	}
	if t.RevokedAt != nil {
		return "revoked"
	}
	if !t.ExpiresAt.After(time.Now().UTC()) {
		return "expired"
	}
	return "active"
}
