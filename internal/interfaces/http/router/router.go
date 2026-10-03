package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/wsc-zz/service/docs"
)

// InitRouter 初始化路由，注入用户应用服务。

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

func InitOrderRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	orderGroup := r.Group("/api")
	registerOrderRoutes(orderGroup) // /api/order/**
	return r
}
