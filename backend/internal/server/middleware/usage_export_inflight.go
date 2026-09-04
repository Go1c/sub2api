package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const usageExportInflightTTL = 30 * time.Second

// UsageExportInflight allows one in-flight export per user. Redis errors fail closed.
func UsageExportInflight(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, ok := GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "User not authenticated",
			})
			return
		}
		if rdb == nil {
			abortUsageExportBusy(c)
			return
		}
		key := fmt.Sprintf("usage-export-lock:user:%d", subject.UserID)
		acquired, err := rdb.SetNX(c.Request.Context(), key, "1", usageExportInflightTTL).Result()
		if err != nil || !acquired {
			abortUsageExportBusy(c)
			return
		}
		defer func() {
			_ = rdb.Del(context.Background(), key).Err()
		}()
		c.Next()
	}
}

func abortUsageExportBusy(c *gin.Context) {
	c.Header("Retry-After", strconv.Itoa(int(usageExportInflightTTL.Seconds())))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":    http.StatusTooManyRequests,
		"message": "usage export already in progress, retry later",
		"reason":  "USAGE_EXPORT_IN_FLIGHT",
	})
}
