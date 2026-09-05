package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	orderapp "github.com/wsc-zz/service/internal/application/order"
	orderpo "github.com/wsc-zz/service/internal/infrastructure/persistence/order"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func registerOrderRoutes(api *gin.RouterGroup) {
	orderRepo := orderpo.NewOrderRepository(global.DB)
	productRepo := productpo.NewProductRepository(global.DB)
	orderSvc := orderapp.NewService(orderRepo, productRepo)
	h := handler.NewOrderHandler(orderSvc)
	orderApi := api.Group("/orders")
	{
		orderApi.POST("create", h.Create)
	}
}
