package nacos

import (
	"fmt"
	"net"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/wsc-zz/service/global"
)

type Registry struct {
	client      naming_client.INamingClient
	serviceName string
	ip          string
	port        uint64
	groupName   string
}

// newNamingClient 按配置创建 Nacos 命名客户端，服务注册与发现共用
func newNamingClient() (naming_client.INamingClient, error) {
	return clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig: &constant.ClientConfig{
			NamespaceId: global.Conf.Nacos.Namespace,
			Username:    global.Conf.Nacos.Username,
			Password:    global.Conf.Nacos.Password,
			TimeoutMs:   5000,
			LogLevel:    "error",
		},
		ServerConfigs: []constant.ServerConfig{*constant.NewServerConfig(
			global.Conf.Nacos.ServerAddr,
			global.Conf.Nacos.ServerPort,
		)},
	})
}

func groupNameOrDefault() string {
	if global.Conf.Nacos.GroupName == "" {
		return "DEFAULT_GROUP"
	}
	return global.Conf.Nacos.GroupName
}

func Register() (*Registry, error) {
	if !global.Conf.Nacos.Enabled {
		return nil, nil
	}

	ip := global.Conf.Nacos.ServiceIP
	if ip == "" {
		var err error
		ip, err = localIPv4()
		if err != nil {
			return nil, err
		}
	}
	groupName := groupNameOrDefault()

	client, err := newNamingClient()
	if err != nil {
		return nil, fmt.Errorf("创建 Nacos 客户端失败: %w", err)
	}

	port := global.Conf.Service.Port
	if port == 0 {
		return nil, fmt.Errorf("服务端口不能为空")
	}
	serviceName := global.Conf.Service.Name
	if serviceName == "" {
		return nil, fmt.Errorf("服务名不能为空")
	}

	registered, err := client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,           // 默认使用本机 IP
		Port:        uint64(port), // 默认使用本服务端口
		Weight:      1,            // 默认权重
		Enable:      true,         // 默认启用
		Healthy:     true,         // 默认健康
		ServiceName: serviceName,  // 默认服务名
		GroupName:   groupName,    //	默认分组
		Ephemeral:   true,         // 默认临时实例
	})
	if err != nil {
		return nil, fmt.Errorf("注册 Nacos 服务失败: %w", err)
	}
	if !registered {
		return nil, fmt.Errorf("注册 Nacos 服务失败: Nacos 未确认注册结果")
	}

	return &Registry{client: client, serviceName: serviceName, ip: ip, port: uint64(port), groupName: groupName}, nil
}

func (r *Registry) Deregister() error {
	if r == nil {
		return nil
	}
	_, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          r.ip,
		Port:        r.port,
		ServiceName: r.serviceName,
		GroupName:   r.groupName,
		Ephemeral:   true,
	})
	return err
}

func localIPv4() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, networkInterface := range interfaces {
		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
				return ipNet.IP.To4().String(), nil
			}
		}
	}
	return "", fmt.Errorf("未找到可注册到 Nacos 的本机 IPv4 地址")
}
