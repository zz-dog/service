package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/interfaces/http/corspolicy"
)

// CORSMiddleware 返回跨域中间件，主服务与 identity 服务共用同一份配置。
// 放行判定、方法与请求头清单来自 corspolicy 包(与网关共用同一份策略):
// 网关也要答预检与错误响应的 CORS,两处配置漂移会让预检死在浏览器侧。
// AllowHeaders 必须包含 X-Merchant-Id:N:N 下商户端靠它声明"当前店铺",
// 不进白名单的话预检就被掐死在浏览器侧,请求根本到不了业务代码。
func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc:  corspolicy.IsAllowedOrigin,
		AllowMethods:     corspolicy.AllowMethods,
		AllowHeaders:     corspolicy.AllowHeaders,
		ExposeHeaders:    corspolicy.ExposeHeaders,
		AllowCredentials: true,
		MaxAge:           corspolicy.MaxAge,
	})
}
