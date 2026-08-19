package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerProductRouter(r *gin.RouterGroup, h *handler.ProductHandler) {
	productApi := r.Group("/Product")
	{
		productApi.POST("/Create", h.Create)
	}
}
