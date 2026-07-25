package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	orderapp "github.com/wsc-zz/service/internal/application/order"
	userapp "github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

// InitRouter 初始化路由，注入用户应用服务。
func InitRouter(userSvc *userapp.Service, orderSvc *orderapp.Service) *gin.Engine {
	r := gin.Default()

	// 允许本地开发前端跨域访问
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handler.NewHandler(userSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/user/register", h.Register)
		apiGroup.POST("/user/login", h.Login)

		// 订单：全部需要登录（JWT 中间件校验 token 并写入 userId）
		orderGroup := apiGroup.Group("/order", middleware.JWTAuth())
		{
			orderGroup.POST("", orderH.Create) // 创建订单
			// orderGroup.GET("", orderH.List)               // 订单列表（分页）
			orderGroup.GET("/:id", orderH.GetByID) // 订单详情
			// orderGroup.POST("/:id/pay", orderH.Pay)       // 模拟支付
			// orderGroup.POST("/:id/cancel", orderH.Cancel) // 取消订单
		}
	}
	return r
}
