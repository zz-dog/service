package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/wsc-zz/service/docs"
)

// InitRouter 初始化路由，注入用户应用服务。

func InitRouter() *gin.Engine {
	var r = gin.Default()
	// 允许本地开发前端跨域访问
	r.Use(CORSMiddleware())

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
