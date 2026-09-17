package router

import (
	"net"
	"net/url"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware 返回跨域中间件，主服务与 identity 服务共用同一份配置。
// 放行本地与局域网来源，便于开发期跨端联调。
func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc:  isAllowedOrigin,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// isAllowedOrigin 判断来源是否允许：localhost / 回环 / 私有网段 IP（任意端口）。
func isAllowedOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "::1" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	return false
}
