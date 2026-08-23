package router

import (
	"github.com/gin-gonic/gin"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerCategoryRoutes(router *gin.RouterGroup, categorySvc *categoryapp.Service) {
	h := handler.NewCategoryHandler(categorySvc)
	categoryApi := router.Group("/category")
	{
		categoryApi.POST("/create", h.Create)  // 创建分类
		categoryApi.PUT("/:id", h.Update)      // 更新分类
		categoryApi.GET("/findAll", h.FindAll) // 查询所有分类

	}
}
