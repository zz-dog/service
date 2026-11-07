package global

import (
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局DB实例
var DB *gorm.DB

func InitMysql() {
	db, err := gorm.Open(mysql.Open(Conf.MySQL.DSN()), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info), // 打印SQL日志
		TranslateError: true,                                // 唯一索引冲突翻译成 gorm.ErrDuplicatedKey,仓储层才能 errors.Is 判重
	})
	if err != nil {
		Logger.Error("mysql连接失败", zap.Error(err))
		panic("mysql连接失败:" + err.Error())
	}
	DB = db
	Logger.Info("mysql连接成功")
}
