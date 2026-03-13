package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/ratelimit"
)

func LoginRateLimit(limiter *ratelimit.RedisLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:login:%s", c.ClientIP())
		allowed, err := limiter.Allow(c.Request.Context(), key, 10, time.Minute)
		if err != nil {
			_ = c.Error(apperror.Internal(err))
			c.Abort()
			return
		}
		if !allowed {
			_ = c.Error(apperror.New(http.StatusTooManyRequests, 42901, "登录接口触发限流"))
			c.Abort()
			return
		}

		c.Next()
	}
}
