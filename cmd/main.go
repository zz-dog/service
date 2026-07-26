package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	orderapp "github.com/wsc-zz/service/internal/application/order"
	userapp "github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	orderpo "github.com/wsc-zz/service/internal/infrastructure/persistence/order"

	userpo "github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/infrastructure/security"
	"github.com/wsc-zz/service/internal/interfaces/http/router"

	categoryapp "github.com/wsc-zz/service/internal/application/category"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
)

// @title           Demo Service API
// @version         1.0
// @description     DDD 架构示例服务（用户 / 订单 / 分类）
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey  ApiKeyAuth
// @in                           header
// @name                         Authorization
func main() {
	// 1. 初始化基础设施：配置、日志、数据库
	global.InitViper()
	global.InitZap()
	global.InitMysql()

	// 2. 自动迁移持久化对象，确保表已创建/更新
	if err := global.DB.AutoMigrate(
		&userpo.UserPO{},
		&orderpo.OrderPO{},
		&orderpo.OrderItemPO{},
		&categorypo.CategoryPO{},
	); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 3. 组合根：依赖注入装配（唯一感知所有层的地方）
	userRepo := userpo.NewUserRepository(global.DB)
	hasher := security.NewBcryptHasher()
	tokenIssuer := auth.NewJWTTokenIssuer()
	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer)

	orderRepo := orderpo.NewOrderRepository(global.DB)
	orderSvc := orderapp.NewService(orderRepo)

	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySvc := categoryapp.NewService(categoryRepo)
	//product
	// productRpo := orderpo.NewProductRepository(global.DB)
	// 4. 启动 HTTP 服务
	r := router.InitRouter(userSvc, orderSvc, categorySvc)
	if err := r.Run(":" + fmt.Sprint(global.Conf.Service.Port)); err != nil {
		panic(err)
	}
}
