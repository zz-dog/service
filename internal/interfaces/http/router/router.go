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
	registerCategoryRoutes(apiGroup)
	RegisterSpecRoutes(apiGroup)
	registerProductRouter(apiGroup)
	registerOrderRoutes(apiGroup)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

// InitCatalogRouter catalog 服务的路由入口:分类/规格/商品挂在 /api/catalog 前缀下。
// 网关按最长前缀把 /api/catalog/** 转发到 catalog 服务,因此路径必须带此前缀。
func InitCatalogRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	catalogGroup := r.Group("/api/catalog")
	registerCategoryRoutes(catalogGroup) // /api/catalog/category/**
	RegisterSpecRoutes(catalogGroup)     // /api/catalog/spec/**
	registerProductRouter(catalogGroup)  // /api/catalog/product/**
	return r
}
