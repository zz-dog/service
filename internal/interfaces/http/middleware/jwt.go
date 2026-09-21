package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	"github.com/wsc-zz/service/pkg/response"
)

// JWTAuth 校验 Authorization 头中的 Bearer token，通过后将用户信息写入上下文。
func JWTAuth(blacklist TokenBlacklistChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, 401, "缺少Authorization头")
			c.Abort()
			return
		}
		const prefix = "Bearer "
		if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
			response.Unauthorized(c, 401, "Authorization头格式错误")
			c.Abort()
			return
		}
		token := authHeader[len(prefix):]
		claims, err := auth.ParseToken(token)
		if err != nil {
			response.Unauthorized(c, 401, "token已失效或非法，请重新登录")
			c.Abort()
			return
		}
		if claims.ID != "" && blacklist != nil {
			revoked, err := blacklist.IsRevoked(c.Request.Context(), claims.ID)
			if err != nil {
				// fail-open:Redis 故障时放行保证可用性;安全敏感场景应 fail-closed
				global.Logger.Warn("查询token黑名单失败，降级放行", zap.Error(err))
			} else if revoked {
				response.Unauthorized(c, 401, "token已失效，请重新登录")
				c.Abort()
				return
			}
		}
		// 将用户信息存入上下文，后续接口可以直接取
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("jti", claims.ID)
		if claims.ExpiresAt != nil {
			c.Set("tokenExp", claims.ExpiresAt.Time)
		}
		// 放行，执行后续接口
		c.Next()
	}
}

// TokenBlacklistChecker 中间件消费侧的窄接口:只关心"这枚 token 是否已被拉黑"。
// 由 router 组合根注入 Redis 实现,中间件本身不依赖 go-redis。
type TokenBlacklistChecker interface {
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
