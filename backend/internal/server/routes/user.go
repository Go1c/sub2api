package routes

import (
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// User-facing rate limits (per authenticated user_id when available).
const (
	userUsageListRPM         = 60  // list / get-by-id
	userUsageAggregateRPM    = 20  // stats + dashboard aggregates
	userAccessTokenCreateRPM = 10  // create uat_
	userUATAPIReadRPM        = 120 // general UAT-allowed keys/groups/subscriptions
	// Wallet balance polling (auth/me + user/profile). Keep low to protect DB under scripts.
	userWalletBalanceRPM = 30
)

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
	redisClient *redis.Client,
) {
	rateLimiter := appmiddleware.NewRateLimiter(redisClient)
	userKey := func(c *gin.Context) string {
		if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
			return fmt.Sprintf("user:%d", subject.UserID)
		}
		return "ip:" + c.ClientIP()
	}

	// Wallet queries accept JWT/UAT; debit adds a strict Header-JWT guard and
	// authenticates the server-side balance client inside the handler.
	wallet := v1.Group("/user/balance")
	wallet.Use(gin.HandlerFunc(jwtAuth))
	wallet.Use(middleware.WalletBackendModeGuard(settingService))
	wallet.Use(gin.HandlerFunc(auditLog))
	{
		wallet.GET("/transactions", h.BalanceWallet.ListTransactions)
		wallet.GET("/transactions/:txn_id", h.BalanceWallet.GetTransaction)
		wallet.POST("/debit", middleware.RequireWalletHeaderJWT(), h.BalanceWallet.Debit)
	}

	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	// 用户管理面变更类操作入审计（含 TOTP 启用/禁用、step-up 验证、密码修改等安全事件）
	authenticated.Use(gin.HandlerFunc(auditLog))
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			user.GET("/profile", rateLimiter.LimitWithOptions("user-wallet-balance", userWalletBalanceRPM, time.Minute, appmiddleware.RateLimitOptions{
				KeyFunc:     userKey,
				FailureMode: appmiddleware.RateLimitFailOpen,
			}), h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/aff", h.User.GetAffiliate)
			user.GET("/aff/logs", h.User.ListAffiliateInviteLogs)
			user.POST("/aff/transfer", h.User.TransferAffiliateQuota)
			user.POST("/account-bindings/email/send-code", h.User.SendEmailBindingCode)
			user.POST("/account-bindings/email", h.User.BindEmailIdentity)
			user.DELETE("/account-bindings/:provider", h.User.UnbindIdentity)
			user.POST("/auth-identities/bind/start", h.User.StartIdentityBinding)
			user.GET("/platform-quotas", h.User.GetMyPlatformQuotas)

			// 通知邮箱管理
			notifyEmail := user.Group("/notify-email")
			{
				notifyEmail.POST("/send-code", h.User.SendNotifyEmailCode)
				notifyEmail.POST("/verify", h.User.VerifyNotifyEmail)
				notifyEmail.PUT("/toggle", h.User.ToggleNotifyEmail)
				notifyEmail.DELETE("", h.User.RemoveNotifyEmail)
			}

			// Webhook 通知（余额/站内信/公告）
			user.POST("/webhook-balance-notify/test", h.User.SendWebhookBalanceNotifyTest)

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
				// 敏感操作二次验证：授予当前会话一段时间的 step-up 权限
				totp.POST("/step-up", h.Totp.StepUp)
			}

			// 用户长效 Access Token（仅 JWT 会话；opaque access token 鉴权会被路径守卫拒绝）
			accessTokens := user.Group("/access-tokens")
			{
				accessTokens.GET("", h.UserAccessToken.List)
				accessTokens.POST("", rateLimiter.LimitWithOptions("user-access-token-create", userAccessTokenCreateRPM, time.Minute, appmiddleware.RateLimitOptions{
					KeyFunc:     userKey,
					FailureMode: appmiddleware.RateLimitFailOpen,
				}), h.UserAccessToken.Create)
				accessTokens.DELETE("/:id", h.UserAccessToken.Revoke)
			}
		}

		// API Key管理（UAT 可写；按用户限流）
		keys := authenticated.Group("/keys")
		keys.Use(rateLimiter.LimitWithOptions("user-keys", userUATAPIReadRPM, time.Minute, appmiddleware.RateLimitOptions{
			KeyFunc:     userKey,
			FailureMode: appmiddleware.RateLimitFailOpen,
		}))
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		groups.Use(rateLimiter.LimitWithOptions("user-groups", userUATAPIReadRPM, time.Minute, appmiddleware.RateLimitOptions{
			KeyFunc:     userKey,
			FailureMode: appmiddleware.RateLimitFailOpen,
		}))
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		// 用户可用渠道（非管理员接口）
		channels := authenticated.Group("/channels")
		{
			channels.GET("/available", h.AvailableChannel.List)
		}

		// 使用记录（UAT 可读；list 与聚合分开配额）
		usage := authenticated.Group("/usage")
		{
			usageListLimit := rateLimiter.LimitWithOptions("user-usage-list", userUsageListRPM, time.Minute, appmiddleware.RateLimitOptions{
				KeyFunc:     userKey,
				FailureMode: appmiddleware.RateLimitFailOpen,
			})
			usageAggLimit := rateLimiter.LimitWithOptions("user-usage-agg", userUsageAggregateRPM, time.Minute, appmiddleware.RateLimitOptions{
				KeyFunc:     userKey,
				FailureMode: appmiddleware.RateLimitFailOpen,
			})
			usage.GET("", usageListLimit, h.Usage.List)
			usage.GET("/:id", usageListLimit, h.Usage.GetByID)
			usage.GET("/stats", usageAggLimit, h.Usage.Stats)
			// User dashboard endpoints
			usage.GET("/dashboard/stats", usageAggLimit, h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", usageAggLimit, h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", usageAggLimit, h.Usage.DashboardModels)
			usage.POST("/dashboard/api-keys-usage", usageAggLimit, h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// 站内信（用户收发）
		siteMessages := authenticated.Group("/site-messages")
		{
			siteMessages.GET("/inbox", h.SiteMessage.ListInbox)
			siteMessages.GET("/sent", h.SiteMessage.ListSent)
			siteMessages.GET("/unread-count", h.SiteMessage.UnreadCount)
			siteMessages.GET("/recipient/resolve", h.SiteMessage.ResolveRecipient)
			siteMessages.POST("", h.SiteMessage.Send)
			siteMessages.GET("/:id", h.SiteMessage.Get)
			siteMessages.POST("/:id/reply", h.SiteMessage.Reply)
			siteMessages.POST("/:id/read", h.SiteMessage.MarkRead)
		}

		lottery := authenticated.Group("/lottery")
		{
			lottery.GET("/active", h.Lottery.GetActive)
			lottery.POST("/:id/draw", h.Lottery.Draw)
		}

		// 发票申请
		invoices := authenticated.Group("/invoices")
		{
			invoices.GET("/overview", h.Invoice.Overview)
			invoices.GET("", h.Invoice.List)
			invoices.POST("", h.Invoice.Create)
			invoices.GET("/:id/download", h.Invoice.Download)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// 用户订阅（只读路径与 UAT 白名单重叠，按用户限流）
		subscriptions := authenticated.Group("/subscriptions")
		{
			subReadLimit := rateLimiter.LimitWithOptions("user-subscriptions-read", userUATAPIReadRPM, time.Minute, appmiddleware.RateLimitOptions{
				KeyFunc:     userKey,
				FailureMode: appmiddleware.RateLimitFailOpen,
			})
			subscriptions.GET("", subReadLimit, h.Subscription.List)
			subscriptions.GET("/active", subReadLimit, h.Subscription.GetActive)
			subscriptions.GET("/progress", subReadLimit, h.Subscription.GetProgress)
			subscriptions.GET("/summary", subReadLimit, h.Subscription.GetSummary)
			subscriptions.POST("/:id/reset-weekly-limit", h.Subscription.ResetWeeklyLimit)
		}

		// 渠道监控（用户只读）
		monitors := authenticated.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
	}
}
