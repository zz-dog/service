package middleware

import (
	"demo/pkg/jwt"
	"demo/pkg/response"

	"github.com/gin-gonic/gin"
)

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
		claims, err := jwt.ParseToken(token)
		if err != nil {
			response.Unauthorized(c, 401, "token已失效或非法，请重新登录")
			c.Abort()
			return
		}

		// 将用户信息存入上下文，后续接口可以直接取
		c.Set("userId", claims.UserId)
		c.Set("username", claims.Username)
		// 放行，执行后续接口
		c.Next()
	}
}
