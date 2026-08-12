package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestUserRateLimitConstants(t *testing.T) {
	require.Equal(t, 60, userUsageListRPM)
	require.Equal(t, 20, userUsageAggregateRPM)
	require.Equal(t, 10, userAccessTokenCreateRPM)
	require.Equal(t, 120, userUATAPIReadRPM)
	require.Equal(t, 30, userWalletBalanceRPM)
}

func TestUserUsageRateLimitFailOpenWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := appmiddleware.NewRateLimiter(redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	}))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
		c.Next()
	})
	r.GET("/usage", limiter.LimitWithOptions("user-usage-list", 1, time.Minute, appmiddleware.RateLimitOptions{
		KeyFunc: func(c *gin.Context) string {
			if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
				return "user:7"
			}
			return "ip:" + c.ClientIP()
		},
		FailureMode: appmiddleware.RateLimitFailOpen,
	}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
