//go:build unit

package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type desktopRouteHandoffStub struct{}

func (s *desktopRouteHandoffStub) Issue(
	_ context.Context,
	_ int64,
) (*service.DesktopPaymentHandoffTicket, error) {
	return &service.DesktopPaymentHandoffTicket{Token: "dph_route", ExpiresIn: 60}, nil
}

func (s *desktopRouteHandoffStub) Consume(
	_ context.Context,
	_ string,
) (*service.DesktopPaymentHandoffSession, error) {
	return &service.DesktopPaymentHandoffSession{
		AccessToken: "jwt-route",
		RedirectURL: "/payment?desktop_handoff=1",
		ExpiresIn:   900,
	}, nil
}

func TestRegisterDesktopRoutesSeparatesAuthenticatedIssueAndPublicConsume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	handlers := &handler.Handlers{
		DesktopHandoff: handler.NewDesktopPaymentHandoffHandler(&desktopRouteHandoffStub{}),
	}
	router := gin.New()
	deny := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	noAudit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	RegisterDesktopRoutes(router.Group("/api/v1"), handlers, deny, noAudit, nil, rdb)

	issue := httptest.NewRecorder()
	router.ServeHTTP(issue, httptest.NewRequest(http.MethodPost, "/api/v1/desktop/payment-handoff", nil))
	require.Equal(t, http.StatusUnauthorized, issue.Code)

	consume := httptest.NewRecorder()
	router.ServeHTTP(
		consume,
		httptest.NewRequest(http.MethodGet, "/api/v1/desktop/payment-handoff/consume?token=dph_route", nil),
	)
	require.Equal(t, http.StatusSeeOther, consume.Code)
	require.Equal(t, "/payment?desktop_handoff=1", consume.Header().Get("Location"))
}

func TestRegisterDesktopRoutesIssueReceivesAuthenticatedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	handlers := &handler.Handlers{
		DesktopHandoff: handler.NewDesktopPaymentHandoffHandler(&desktopRouteHandoffStub{}),
	}
	router := gin.New()
	allow := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	noAudit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	RegisterDesktopRoutes(router.Group("/api/v1"), handlers, allow, noAudit, nil, rdb)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/desktop/payment-handoff", nil),
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"expires_in":60`)
}

func TestRegisterDesktopRoutesConsumeRateLimitFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	handlers := &handler.Handlers{
		DesktopHandoff: handler.NewDesktopPaymentHandoffHandler(&desktopRouteHandoffStub{}),
	}
	router := gin.New()
	allow := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() })
	noAudit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	RegisterDesktopRoutes(router.Group("/api/v1"), handlers, allow, noAudit, nil, rdb)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/desktop/payment-handoff/consume?token=dph_route",
		nil,
	)
	request.RemoteAddr = "203.0.113.10:12345"
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, recorder.Body.String(), "rate limit exceeded")
}
