package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"

	categoryCache "github.com/wsc-zz/service/internal/infrastructure/cache/redis"
)

// newCategoryHandler 分类应用服务的装配:管理端/C 端两个注册函数共用一套装配,
// 同一个应用服务,两个 HTTP 翻译壳——拆的是接口层,不是应用层
func newCategoryHandler() *handler.CategoryHandler {
	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySpecRepo := categorypo.NewCategorySpecRepository(global.DB)

	//spec
	specRepo := specpo.NewSpecRepository(global.DB)
	categoryCache := categoryCache.NewCategoryCache(global.RedisClient)
	categorySvc := categoryapp.NewService(categoryRepo, specRepo, categorySpecRepo, categoryCache)
	return handler.NewCategoryHandler(categorySvc)
}

// registerCategoryAdminRoutes 分类管理端写接口:挂在 /api/catalog/admin/category/**,
// 由 admin.go 的 JWTAuth+AdminAuth 组统一收口,本函数不再自己挂鉴权
func registerCategoryAdminRoutes(r *gin.RouterGroup) {
	h := newCategoryHandler()
	categoryApi := r.Group("/category")
	{
		categoryApi.POST("/create", h.Create)           // 创建分类
		categoryApi.PUT("/:id", h.Update)               // 更新分类
		categoryApi.DELETE("/:id", h.Delete)            // 删除分类
		categoryApi.POST("/bindSpec", h.BindSpec)       // 绑定规格到分类
		categoryApi.DELETE("/unbindSpec", h.UnbindSpec) // 解除规格绑定
	}
}

// registerCategoryRoutes 分类 C 端读接口:保持原路径原鉴权
// (挂不挂 JWT 是产品决策,现状是什么就保持什么)
func registerCategoryRoutes(r *gin.RouterGroup) {
	h := newCategoryHandler()
	blacklist := NewTokenBlacklist()
	categoryApi := r.Group("/category", middleware.JWTAuth(blacklist))
	{
		categoryApi.GET("/findAll", h.FindAll)         // 查询所有分类
		categoryApi.GET("/findSpecs/:id", h.ListSpecs) // 查询分类绑定的规格
	}
}
