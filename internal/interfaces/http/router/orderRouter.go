package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	orderapp "github.com/wsc-zz/service/internal/application/order"
	orderpo "github.com/wsc-zz/service/internal/infrastructure/persistence/order"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	transaction "github.com/wsc-zz/service/internal/infrastructure/persistence/transaction"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"go.uber.org/zap"

	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"

	redisCache "github.com/wsc-zz/service/internal/infrastructure/cache/redis"
)

func registerOrderRoutes(api *gin.RouterGroup) {
	orderRepo := orderpo.NewOrderRepository(global.DB)
	productRepo := productpo.NewProductRepository(global.DB)
	orderSvc := orderapp.NewService(orderRepo, productRepo, transaction.NewRunner(global.DB))
	// 定时取消订单
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := orderSvc.AutoCancelExpired(context.Background(), 5*time.Minute); err != nil {
				global.Logger.Error("自动取消过期订单失败", zap.Error(err))
			}
		}
	}()

	// 注册订单相关路由
	blacklist := NewTokenBlacklist()
	limiter := redisCache.NewDedupStore(global.RedisClient)
	h := handler.NewOrderHandler(orderSvc)
	orderApi := api.Group("/orders", middleware.JWTAuth(blacklist))
	{
		orderApi.POST("create", middleware.OrderCreateGuard(limiter, 3*time.Second), h.Create)
		orderApi.POST("cancel", h.Cancel)
		orderApi.POST("list", h.List)
	}
}
