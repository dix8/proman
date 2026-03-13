package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/ratelimit"
)

func PublicRateLimit(limiter *ratelimit.RedisLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := CurrentProjectID(c)
		if projectID == 0 {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40103, "项目 Token 无效"))
			c.Abort()
			return
		}

		key := fmt.Sprintf("rate_limit:public:project:%d", projectID)
		allowed, err := limiter.Allow(c.Request.Context(), key, 60, time.Minute)
		if err != nil {
			_ = c.Error(apperror.Internal(err))
			c.Abort()
			return
		}
		if !allowed {
			_ = c.Error(apperror.New(http.StatusTooManyRequests, 42902, "对外接口触发限流"))
			c.Abort()
			return
		}

		c.Next()
	}
}
