package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerCatergoryRoutes(router *gin.RouterGroup, h *handler.CategoryHandler) {
	categoryApi := router.Group("/category")
	{
		categoryApi.POST("/create", h.Create) // 创建分类
		categoryApi.PUT("/:id", h.Update)     // 更新分类
	}
}
