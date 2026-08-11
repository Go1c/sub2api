package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	desktopPaymentHandoffIssueLimit   = 10
	desktopPaymentHandoffConsumeLimit = 30
	desktopPaymentHandoffRateWindow   = time.Minute
)

func RegisterDesktopRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	auditLog servermiddleware.AuditLogMiddleware,
	settingService *service.SettingService,
	redisClient *redis.Client,
) {
	if h == nil || h.DesktopHandoff == nil {
		return
	}

	rateLimiter := appmiddleware.NewRateLimiter(redisClient)
	desktop := v1.Group("/desktop")
	desktop.GET(
		"/payment-handoff/consume",
		rateLimiter.LimitWithOptions(
			"desktop-payment-handoff-consume",
			desktopPaymentHandoffConsumeLimit,
			desktopPaymentHandoffRateWindow,
			appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose},
		),
		h.DesktopHandoff.Consume,
	)

	issue := desktop.Group("")
	issue.Use(gin.HandlerFunc(jwtAuth))
	issue.Use(servermiddleware.BackendModeUserGuard(settingService))
	issue.Use(gin.HandlerFunc(auditLog))
	issue.POST(
		"/payment-handoff",
		rateLimiter.LimitWithOptions(
			"desktop-payment-handoff-issue",
			desktopPaymentHandoffIssueLimit,
			desktopPaymentHandoffRateWindow,
			appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose},
		),
		h.DesktopHandoff.Issue,
	)
}
