package handler

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type desktopPaymentHandoffService interface {
	Issue(ctx context.Context, userID int64) (*service.DesktopPaymentHandoffTicket, error)
	Consume(ctx context.Context, token string) (*service.DesktopPaymentHandoffSession, error)
}

type DesktopPaymentHandoffHandler struct {
	service desktopPaymentHandoffService
}

type DesktopPaymentHandoffIssueResponse struct {
	HandoffURL string `json:"handoff_url"`
	ExpiresIn  int    `json:"expires_in"`
}

func NewDesktopPaymentHandoffHandler(service desktopPaymentHandoffService) *DesktopPaymentHandoffHandler {
	return &DesktopPaymentHandoffHandler{service: service}
}

func (h *DesktopPaymentHandoffHandler) Issue(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticket, err := h.service.Issue(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, DesktopPaymentHandoffIssueResponse{
		HandoffURL: "/api/v1/desktop/payment-handoff/consume?token=" + url.QueryEscape(ticket.Token),
		ExpiresIn:  ticket.ExpiresIn,
	})
}

func (h *DesktopPaymentHandoffHandler) Consume(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")

	session, err := h.service.Consume(c.Request.Context(), c.Query("token"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setLumioWebSessionCookie(c, session.AccessToken, session.ExpiresIn)
	c.Redirect(http.StatusSeeOther, session.RedirectURL)
}

func setLumioWebSessionCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     service.LumioWebSessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isRequestHTTPS(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearLumioWebSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     service.LumioWebSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isRequestHTTPS(c),
		SameSite: http.SameSiteLaxMode,
	})
}
