package nacos

import (
	"fmt"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
)

// Discovery 服务发现客户端：按服务名查询 Nacos 上的健康实例，
// 与 Registry（服务注册）相对，是消费方的"另一半"。
type Discovery struct {
	client naming_client.INamingClient
}

// NewDiscovery 创建服务发现客户端；Nacos 未启用时返回 (nil, nil)，此时 Resolve 走兜底地址
func NewDiscovery() (*Discovery, error) {
	if !global.Conf.Nacos.Enabled {
		return nil, nil
	}
	client, err := newNamingClient()
	if err != nil {
		return nil, fmt.Errorf("创建 Nacos 客户端失败: %w", err)
	}
	return &Discovery{client: client}, nil
}

// Select 返回服务在 Nacos 上的全部健康实例（自动过滤不健康、已下线、权重≤0 的实例）
func (d *Discovery) Select(serviceName string) ([]model.Instance, error) {
	if d == nil {
		return nil, nil
	}
	return d.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   groupNameOrDefault(),
		HealthyOnly: true,
	})
}

// Resolver 带轮询负载均衡的服务地址解析器：
// Nacos 可用时按服务名轮询健康实例，否则退回配置的兜底地址。
type Resolver struct {
	discovery *Discovery
	fallback  map[string]string // 服务名 → 兜底地址(ip:port)，Nacos 未启用或无可用实例时使用

	mu       sync.Mutex
	counters map[string]uint64 // 服务名 → 轮询计数
}

func NewResolver(discovery *Discovery, fallback map[string]string) *Resolver {
	return &Resolver{discovery: discovery, fallback: fallback, counters: make(map[string]uint64)}
}

// Resolve 解析出一个可用实例地址，无可用实例时返回错误
func (r *Resolver) Resolve(serviceName string) (string, error) {
	if instances, err := r.discovery.Select(serviceName); err != nil {
		// 发现失败不阻断请求，退回兜底地址
		global.Logger.Warn("Nacos 服务发现失败，尝试兜底地址", zap.String("service", serviceName), zap.Error(err))
	} else if len(instances) > 0 {
		return instanceAddr(instances[r.next(serviceName, len(instances))]), nil
	}

	if addr, ok := r.fallback[serviceName]; ok {
		return addr, nil
	}
	return "", fmt.Errorf("服务 %s 无可用实例", serviceName)
}

// next 轮询计数器：同一服务在多个实例间轮流分发
func (r *Resolver) next(serviceName string, n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[serviceName]++
	return int(r.counters[serviceName] % uint64(n))
}

func instanceAddr(inst model.Instance) string {
	return fmt.Sprintf("%s:%d", inst.Ip, inst.Port)
}
