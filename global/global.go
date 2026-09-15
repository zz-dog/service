package global

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var Conf Config

// confViper 全局配置读写实例：InitViper 装载本地文件，
// MergeRemoteConfig 在此之上合并 Nacos 配置中心下发的远端配置
var confViper *viper.Viper

type Config struct {
	Service ServerCfg  `yaml:"service"`
	Gateway GatewayCfg `yaml:"gateway"`
	Nacos   NacosCfg   `yaml:"nacos"`
	MySQL   MySQLCfg   `yaml:"mysql"`
	Jwt     JwtCfg     `yaml:"jwt"`
}

type ServerCfg struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// GatewayCfg 网关配置：路由表（路径前缀 → 服务名），目标地址由 Nacos 服务发现解析
type GatewayCfg struct {
	Port   int            `yaml:"port" mapstructure:"port"`
	Routes []GatewayRoute `yaml:"routes" mapstructure:"routes"`
}

type GatewayRoute struct {
	Prefix      string `yaml:"prefix" mapstructure:"prefix"`   // 匹配的路径前缀，长前缀优先
	ServiceName string `yaml:"service" mapstructure:"service"` // 目标服务在 Nacos 中注册的名字
}

type NacosCfg struct {
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`
	ServerAddr string `yaml:"server_addr" mapstructure:"server_addr"`
	ServerPort uint64 `yaml:"server_port" mapstructure:"server_port"`
	Namespace  string `yaml:"namespace_id" mapstructure:"namespace_id"`
	Username   string `yaml:"username" mapstructure:"username"`
	Password   string `yaml:"password" mapstructure:"password"`
	GroupName  string `yaml:"group_name" mapstructure:"group_name"`
	// 配置中心 Data ID，默认 service-config.yaml；mysql/gateway/jwt 等配置可放此处，远端覆盖本地同名键
	ConfigDataId string `yaml:"config_data_id" mapstructure:"config_data_id"`
	ServiceIP    string `yaml:"service_ip" mapstructure:"service_ip"`
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
	Secret     string `yaml:"secret" mapstructure:"secret"`
	ExpireHour int    `yaml:"expire_hour" mapstructure:"expire_hour"`
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
	confViper = v
	// 指定配置文件路径
	cfgPath := findConfigFile()
	if cfgPath == "" {
		panic("读取配置失败：未找到 config/config.yaml（已搜索当前目录及可执行文件目录的各级上级目录）")
	}
	v.SetConfigFile(cfgPath)
	v.SetConfigType("yaml")
	if err := v.BindEnv("jwt.secret", "JWT_SECRET"); err != nil {
		panic("绑定 JWT_SECRET 失败：" + err.Error())
	}
	if err := v.BindEnv("jwt.expire_hour", "JWT_EXPIRE_HOUR"); err != nil {
		panic("绑定 JWT_EXPIRE_HOUR 失败：" + err.Error())
	}
	for key, env := range map[string]string{
		"service.name":       "SERVICE_NAME",
		"service.port":       "SERVICE_PORT",
		"nacos.enabled":      "NACOS_ENABLED",
		"nacos.server_addr":  "NACOS_SERVER_ADDR",
		"nacos.server_port":  "NACOS_SERVER_PORT",
		"nacos.namespace_id": "NACOS_NAMESPACE_ID",
		"nacos.username":     "NACOS_USERNAME",
		"nacos.password":     "NACOS_PASSWORD",
		"nacos.group_name":     "NACOS_GROUP_NAME",
		"nacos.config_data_id": "NACOS_CONFIG_DATA_ID",
		"nacos.service_ip":     "SERVICE_IP",
	} {
		if err := v.BindEnv(key, env); err != nil {
			panic("绑定环境变量失败：" + env + ": " + err.Error())
		}
	}
	// 读取文件
	if err := v.ReadInConfig(); err != nil {
		panic("读取配置失败：" + err.Error())
	}
	// 映射到结构体
	if err := v.Unmarshal(&Conf); err != nil {
		panic("解析配置失败：" + err.Error())
	}
}

// MergeRemoteConfig 把配置中心下发的 YAML 合并进当前配置：
// 远端键覆盖本地同名键（切片整体替换），环境变量绑定仍然最高优先。
// 供启动时一次性合并与网关路由热更新复用。
func MergeRemoteConfig(content string) error {
	if err := confViper.MergeConfig(strings.NewReader(content)); err != nil {
		return fmt.Errorf("合并远程配置失败: %w", err)
	}
	if err := confViper.Unmarshal(&Conf); err != nil {
		return fmt.Errorf("解析远程配置失败: %w", err)
	}
	return nil
}
