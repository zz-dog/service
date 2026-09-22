package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type orderCreateLimiter interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

func OrderCreateGuard(limiter orderCreateLimiter, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Get("userId")
		uerId := uid.(string)
		if uerId == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "用户未登录"})
			return
		}
		ok, err := limiter.TryLock(c, "order:create"+uerId, ttl)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(429, gin.H{"error": "请求过于频繁"})
			return
		}
		c.Next()
	}
}
