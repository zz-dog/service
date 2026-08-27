package router

import (
	"github.com/gin-gonic/gin"
	productapp "github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerProductRouter(r *gin.RouterGroup, productSvc *productapp.Service) {
	h := handler.NewProductHandler(productSvc)
	productApi := r.Group("/product")
	{
		productApi.POST("/create", h.Create)
	}
}
