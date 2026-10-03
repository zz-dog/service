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
	servicePort uint64
	groupName   string
	RpcEnabled  bool
	RpcName     string
	RpcPort     uint64
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

func (r *Registry) ServiceRegister(serviceName string, port int) error {

	registered, err := r.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          r.ip,         // 默认使用本机 IP
		Port:        uint64(port), // 默认使用本服务端port
		Weight:      1,            // 默认权重
		Enable:      true,         // 默认启用
		Healthy:     true,         // 默认健康
		ServiceName: serviceName,  // 默认服务名
		GroupName:   r.groupName,  //	默认分组
		Ephemeral:   true,         // 默认临时实例
	})
	if err != nil || !registered {
		return fmt.Errorf("注册 Nacos 服务实例失败: %w", err)
	}
	if !registered {
		return fmt.Errorf("注册 Nacos 服务实例失败: Nacos 未确认注册结果")
	}

	return nil
}
func Register() (*Registry, error) {
	groupName := groupNameOrDefault()
	client, err := newNamingClient()
	if err != nil {
		return nil, fmt.Errorf("创建 Nacos 客户端失败: %w", err)
	}
	ip := global.Conf.Nacos.ServiceIP

	return &Registry{client: client, ip: ip, groupName: groupName}, nil

}
func (r *Registry) RpcRegister(RpcName string, RpcPort int) error {

	ip := global.Conf.Nacos.ServiceIP
	if ip == "" {
		var err error
		ip, err = localIPv4()
		if err != nil {
			return err
		}
	}
	groupName := groupNameOrDefault()

	registered, err := r.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,              // 默认使用本机 IP
		Port:        uint64(RpcPort), // 默认使用本服务端口
		Weight:      1,               // 默认权重
		Enable:      true,            // 默认启用
		Healthy:     true,            // 默认健康
		ServiceName: r.RpcName,       // 默认服务名
		GroupName:   groupName,       //	默认分组
		Ephemeral:   true,            // 默认临时实例
	})
	if err != nil || !registered {
		return fmt.Errorf("注册 Nacos 服务实例失败: %w", err)
	}
	if !registered {
		return fmt.Errorf("注册 Nacos 服务实例失败: Nacos 未确认注册结果")
	}
	r.RpcName = RpcName
	r.RpcPort = uint64(RpcPort)
	r.RpcEnabled = true
	return nil
}
func (r *Registry) Deregister() error {
	if r == nil {
		return nil
	}
	_, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          r.ip,
		Port:        r.servicePort,
		ServiceName: r.serviceName,
		GroupName:   r.groupName,
		Ephemeral:   true,
	})
	if err != nil {
		return err
	}
	if r.RpcEnabled {
		_, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          r.ip,
			Port:        r.RpcPort,
			ServiceName: r.RpcName,
			GroupName:   r.groupName,
			Ephemeral:   true,
		})
		return err
	}
	return nil
}

func localIPv4() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	// 优先私网地址（RFC1918），其次任意非链路本地地址。
	// 169.254.x.x 是 Windows 断开网卡/虚拟网卡（如 Wi-Fi Direct）的自动配置地址，
	// 注册出去后调用方会连接黑洞直至 TCP 超时，必须跳过
	var fallback string
	for _, networkInterface := range interfaces {
		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			text := ip.String()
			if ip.IsPrivate() {
				return text, nil
			}
			if fallback == "" {
				fallback = text
			}
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("未找到可注册到 Nacos 的本机 IPv4 地址")
}
