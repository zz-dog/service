package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerCategoryRoutes(router *gin.RouterGroup) {
	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySpecRepo := categorypo.NewCategorySpecRepository(global.DB)

	//spec
	specRepo := specpo.NewSpecRepository(global.DB)
	categorySvc := categoryapp.NewService(categoryRepo, specRepo, categorySpecRepo)
	h := handler.NewCategoryHandler(categorySvc)
	categoryApi := router.Group("/category")
	{
		categoryApi.POST("/create", h.Create)           // 创建分类
		categoryApi.PUT("/:id", h.Update)               // 更新分类
		categoryApi.DELETE("/:id", h.Delete)            // 删除分类
		categoryApi.GET("/findAll", h.FindAll)          // 查询所有分类
		categoryApi.POST("/bindSpec", h.BindSpec)       // 绑定规格到分类
		categoryApi.DELETE("/unbindSpec", h.UnbindSpec) // 解除规格绑定
		categoryApi.GET("/findSpecs/:id", h.ListSpecs)  // 查询分类绑定的规格

	}
}
