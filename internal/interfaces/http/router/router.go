package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/wsc-zz/service/docs"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

// InitRouter 初始化路由，注入用户应用服务。

// InitCatalogRouter catalog 服务的路由入口:分类/规格/商品挂在 /api/catalog 前缀下。
// 网关按最长前缀把 /api/catalog/** 转发到 catalog 服务,因此路径必须带此前缀。
func InitCatalogRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	catalogGroup := r.Group("/api/catalog")

	// C 端读接口:/client 段与 /admin、/merchant 对称,一端一段;
	// 鉴权模型不变(category/spec 挂 JWTAuth,product 公开),只挪路径。
	// 仍在 /api/catalog 前缀下,网关按前缀转发,路由表不用改
	clientGroup := catalogGroup.Group("/client")
	registerCategoryRoutes(clientGroup) // /api/catalog/client/category/**
	registerSpecRoutes(clientGroup)     // /api/catalog/client/spec/**
	registerProductRoutes(clientGroup)  // /api/catalog/client/product/**

	// 管理端写接口:/admin 段统一收口;/api/catalog/admin/** 仍命中网关的
	// /api/catalog 前缀(最长前缀匹配),网关路由表不用改
	registerCatalogAdminRoutes(catalogGroup)

	// 商户端商品接口:/merchant 段(JWTAuth,无 AdminAuth)——商户是普通登录用户,
	// 资格由应用层推导店铺并校验归属,路由层只管认证
	registerCatalogMerchantRoutes(catalogGroup)
	return r
}

// registerCatalogMerchantRoutes catalog 商户端路由:商户经营自己的商品与媒体,
// 鉴权模型是"登录即可,归属后置校验",与 /admin 组(JWTAuth+AdminAuth 双闸)分开——
// 分组的依据是授权模型,不是"有多少种身份"
func registerCatalogMerchantRoutes(catalogGroup *gin.RouterGroup) {
	blacklist := NewTokenBlacklist()
	merchantApi := catalogGroup.Group("/merchant", middleware.JWTAuth(blacklist))
	registerMerchantProductRoutes(merchantApi)
	registerMerchantMediaRoutes(merchantApi) // 商户端媒体上传/签名:对象强制落 merchant/<id>/ 命名空间
}

// registerCatalogAdminRoutes catalog 管理端路由:全部挂在 /admin 段下,由 AdminAuth
// 一处收口——C 端用户 token 到不了任何一个写接口。跨多个域的"端"归这个文件集中注册,
// 一个域内部的"C端/管理端"归各自域文件,两个维度各有各的归属。
// JWTAuth 在前(认证:你是谁),AdminAuth 在后(授权:你够不够格),链式组合
func registerCatalogAdminRoutes(catalogGroup *gin.RouterGroup) {
	blacklist := NewTokenBlacklist()
	adminApi := catalogGroup.Group("/admin", middleware.JWTAuth(blacklist), middleware.AdminAuth())
	registerCategoryAdminRoutes(adminApi)
	registerSpecAdminRoutes(adminApi)
	registerProductAdminRoutes(adminApi)
	registerMediaRoutes(adminApi) // 媒体上传/签名是管理端能力,一并收口
}

// InitIdentityRouter identity 服务的路由入口:用户/地址挂在 /api 前缀下。
// 网关按 /api/user 前缀转发到 identity(见 config.yaml routes),此前缀不能改
func InitIdentityRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	identityGroup := r.Group("/api")
	registerUserRoutes(identityGroup) // /api/user/**
	return r
}

func InitOrderRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	orderGroup := r.Group("/api")
	registerOrderRoutes(orderGroup) // /api/order/**(订单是纯 C 端业务,暂无管理端接口)
	return r
}
