package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/wsc-zz/service/docs"
)

// InitRouter 初始化路由，注入用户应用服务。

func InitRouter() *gin.Engine {
	var r = gin.Default()
	// 允许本地开发前端跨域访问
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// productH := handler.NewProductHandler(productSvc)
	var apiGroup = r.Group("/api")
	registerUserRoutes(apiGroup)
	registerCategoryRoutes(apiGroup)
	RegisterSpecRoutes(apiGroup)
	registerProductRouter(apiGroup)
	registerOrderRoutes(apiGroup)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
