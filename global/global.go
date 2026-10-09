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
	Gateway  GatewayCfg    `yaml:"gateway"`
	Nacos    NacosCfg      `yaml:"nacos"`
	MySQL    MySQLCfg      `yaml:"mysql"`
	Jwt      JwtCfg        `yaml:"jwt"`
	Redis    RedisCfg      `yaml:"redis"`
	Catalog  ServiceConfig `yaml:"catalog"`
	Identity ServiceConfig `yaml:"identity"`
	Order    ServiceConfig `yaml:"order"`
	Payment  PaymentConfig `yaml:"payment"`
}
type PaymentConfig struct {
	NotifyUrl string    `yaml:"notify_url"`
	ReturnUrl string    `yaml:"return_url"`
	Alipay    payConfug `yaml:"alipay"`
}
type payConfug struct {
	AppID      string `yaml:"app_id" mapstructure:"app_id"`
	PrivateKey string `yaml:"private_key" mapstructure:"private_key"`
	PublicKey  string `yaml:"public_key" mapstructure:"public_key"`
}
type ServiceConfig struct {
	RpcAddr     string `yaml:"grpc_addr" mapstructure:"grpc_addr"`
	RpcPort     int    `yaml:"rpc_port" mapstructure:"rpc_port"`
	RpcName     string `yaml:"rpc_name" mapstructure:"rpc_name"`
	ServiceName string `yaml:"service_name" mapstructure:"service_name"`
	ServicePort int    `yaml:"service_port" mapstructure:"service_port"`
}

type RedisCfg struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
	PoolSize int    `yaml:"pool_size" mapstructure:"pool_size"` // 连接池最大连接数
	MinIdle  int    `yaml:"min_idle" mapstructure:"min_idle"`   // 连接池中空闲连接数
}

type ServerCfg struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	RpcPort    int    `yaml:"rpc_port" mapstructure:"rpc_port"`
	RpcName    string `yaml:"rpc_name" mapstructure:"rpc_name"`
	RpcEnabled bool   `yaml:"rpc_enabled" mapstructure:"rpc_enabled"`
}

// GatewayCfg 网关配置：路由表（路径前缀 → 服务名），目标地址由 Nacos 服务发现解析
type GatewayCfg struct {
	ServiceName string         `yaml:"service_name" mapstructure:"service_name"`
	ServicePort int            `yaml:"service_port" mapstructure:"service_port"`
	Routes      []GatewayRoute `yaml:"routes" mapstructure:"routes"`
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
	v := viper.New() // 创建一个 viper 实例
	confViper = v    // 绑定到全局变量 confViper
	// 指定配置文件路径
	cfgPath := findConfigFile() // 查找 config/config.yaml
	if cfgPath == "" {
		panic("读取配置失败：未找到 config/config.yaml（已搜索当前目录及可执行文件目录的各级上级目录）")
	}
	// 环境变量可覆盖同名配置键（仅对文件里已存在的键生效）：
	// catalog.grpc_addr → CATALOG_GRPC_ADDR。容器部署时用它给单个服务注入差异配置，
	// 例如 order 容器把 gRPC 地址指向 compose 服务名 catalog:9082
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

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
