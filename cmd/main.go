package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/infrastructure/security"
	"github.com/wsc-zz/service/internal/interfaces/http/router"
)

func main() {
	// 1. 初始化基础设施：配置、日志、数据库
	global.InitViper()
	global.InitZap()
	global.InitMysql()

	// 2. 自动迁移持久化对象，确保表已创建/更新
	if err := global.DB.AutoMigrate(&userpo.UserPO{}); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 3. 组合根：依赖注入装配（唯一感知所有层的地方）
	userRepo := userpo.NewUserRepository(global.DB)
	hasher := security.NewBcryptHasher()
	tokenIssuer := auth.NewJWTTokenIssuer()
	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer)

	// 4. 启动 HTTP 服务
	r := router.InitRouter(userSvc)
	if err := r.Run(":" + fmt.Sprint(global.Conf.Service.Port)); err != nil {
		panic(err)
	}
}
