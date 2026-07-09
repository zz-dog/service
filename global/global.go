package global

import (
	"github.com/spf13/viper"
)

var Conf Config

type Config struct {
	Service ServerCfg `yaml:"service"`
	MySQL   MySQLCfg  `yaml:"mysql"`
	Jwt     JwtCfg    `yaml:"jwt"`
}

type ServerCfg struct {
	Port int `yaml:"port"`
}

type MySQLCfg struct {
	DSN string `yaml:"dsn"`
}
type JwtCfg struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

// InitViper 加载yaml配置
func InitViper() {
	v := viper.New()
	// 指定配置文件路径
	v.SetConfigFile("./config/config.yaml")
	v.SetConfigType("yaml")
	// 读取文件
	if err := v.ReadInConfig(); err != nil {
		panic("读取配置失败：" + err.Error())
	}
	// 映射到结构体
	if err := v.Unmarshal(&Conf); err != nil {
		panic("解析配置失败：" + err.Error())
	}
}
