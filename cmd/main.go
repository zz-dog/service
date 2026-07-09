package main

import (
	"demo/global"
	"demo/internal/model"
	"demo/internal/router"
	"fmt"

	"go.uber.org/zap"
)

func main() {
	global.InitViper()
	global.InitZap()
	// 3.初始化mysql
	global.InitMysql()
	// 在启动服务器前执行自动迁移，确保表已创建/更新
	if err := global.DB.AutoMigrate(&model.User{}); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	r := router.InitRouter()
	if err := r.Run(":" + fmt.Sprint(global.Conf.Service.Port)); err != nil {
		panic(err)
	}
}
