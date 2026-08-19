package global

import (
	"fmt"
	"os"
	"path/filepath"

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
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
}

// DSN 拼接 MySQL 连接串
func (c MySQLCfg) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
}
type JwtCfg struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

// findConfigFile 定位 config.yaml：
// 1. 从当前工作目录逐级向上查找 config/config.yaml（兼容 go run、任意目录运行）
// 2. 兜底查找可执行文件所在目录及其上级（兼容二进制部署到 /opt/service 等）
func findConfigFile() string {
	// 候选根目录：当前目录 + 可执行文件目录
	var roots []string
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}

	for _, root := range roots {
		dir := root
		for {
			candidate := filepath.Join(dir, "config", "config.yaml")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

// InitViper 加载yaml配置
func InitViper() {
	v := viper.New()
	// 指定配置文件路径
	cfgPath := findConfigFile()
	if cfgPath == "" {
		panic("读取配置失败：未找到 config/config.yaml（已搜索当前目录及可执行文件目录的各级上级目录）")
	}
	v.SetConfigFile(cfgPath)
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
