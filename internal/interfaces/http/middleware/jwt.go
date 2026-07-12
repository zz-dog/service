package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/infrastructure/auth"
	"github.com/wsc-zz/service/pkg/response"
)

// JWTAuth 校验 Authorization 头中的 Bearer token，通过后将用户信息写入上下文。
func JWTAuth() gin.HandlerFunc {
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

		// 将用户信息存入上下文，后续接口可以直接取
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		// 放行，执行后续接口
		c.Next()
	}
}
