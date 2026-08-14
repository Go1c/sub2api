//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type desktopPaymentHandoffServiceStub struct {
	ticket       *service.DesktopPaymentHandoffTicket
	issueErr     error
	issueUserID  int64
	issueCalls   int
	session      *service.DesktopPaymentHandoffSession
	consumeErr   error
	consumeToken string
	consumeCalls int
}

func (s *desktopPaymentHandoffServiceStub) Issue(
	_ context.Context,
	userID int64,
) (*service.DesktopPaymentHandoffTicket, error) {
	s.issueCalls++
	s.issueUserID = userID
	return s.ticket, s.issueErr
}

func (s *desktopPaymentHandoffServiceStub) Consume(
	_ context.Context,
	token string,
) (*service.DesktopPaymentHandoffSession, error) {
	s.consumeCalls++
	s.consumeToken = token
	return s.session, s.consumeErr
}

func TestDesktopPaymentHandoffHandlerIssueReturnsOpaqueRelativeURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &desktopPaymentHandoffServiceStub{
		ticket: &service.DesktopPaymentHandoffTicket{Token: "dph_opaque", ExpiresIn: 60},
	}
	h := NewDesktopPaymentHandoffHandler(stub)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/payment-handoff", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})

	h.Issue(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, int64(42), stub.issueUserID)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	data, ok := envelope.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "/api/v1/desktop/payment-handoff/consume?token=dph_opaque", data["handoff_url"])
	require.Equal(t, float64(60), data["expires_in"])
	require.NotContains(t, data["handoff_url"], "Bearer")
	require.NotContains(t, data["handoff_url"], "sk-")
}

func TestDesktopPaymentHandoffHandlerIssueRequiresAuthenticatedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &desktopPaymentHandoffServiceStub{}
	h := NewDesktopPaymentHandoffHandler(stub)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/payment-handoff", nil)

	h.Issue(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, stub.issueCalls)
}

func TestDesktopPaymentHandoffHandlerConsumeSetsSecureHostOnlyCookieAndIgnoresRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &desktopPaymentHandoffServiceStub{session: &service.DesktopPaymentHandoffSession{
		AccessToken: "jwt-secret",
		RedirectURL: "/payment?desktop_handoff=1",
		ExpiresIn:   900,
	}}
	h := NewDesktopPaymentHandoffHandler(stub)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/desktop/payment-handoff/consume?token=dph_ok&redirect=https://evil.example",
		nil,
	)
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	h.Consume(c)

	require.Equal(t, http.StatusSeeOther, recorder.Code)
	require.Equal(t, "dph_ok", stub.consumeToken)
	require.Equal(t, "/payment?desktop_handoff=1", recorder.Header().Get("Location"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	cookie := findCookie(recorder.Result().Cookies(), service.LumioWebSessionCookieName)
	require.NotNil(t, cookie)
	require.Equal(t, "jwt-secret", cookie.Value)
	require.True(t, cookie.HttpOnly)
	require.True(t, cookie.Secure)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	require.Equal(t, "/", cookie.Path)
	require.Empty(t, cookie.Domain)
	require.Equal(t, 900, cookie.MaxAge)
}

func TestDesktopPaymentHandoffHandlerConsumeReturnsGoneWithoutCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &desktopPaymentHandoffServiceStub{consumeErr: service.ErrDesktopPaymentHandoffInvalid}
	h := NewDesktopPaymentHandoffHandler(stub)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/desktop/payment-handoff/consume?token=dph_invalid",
		nil,
	)

	h.Consume(c)

	require.Equal(t, http.StatusGone, recorder.Code)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	require.Nil(t, findCookie(recorder.Result().Cookies(), service.LumioWebSessionCookieName))
}
