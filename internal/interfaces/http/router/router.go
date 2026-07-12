package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

// InitRouter 初始化路由，注入用户应用服务。
func InitRouter(userSvc *userapp.Service) *gin.Engine {
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

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/user/register", h.Register)
		apiGroup.POST("/user/login", h.Login)
	}
	return r
}
