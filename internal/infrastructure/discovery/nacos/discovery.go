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

// NewDiscovery 创建服务发现客户端；Nacos 未启用时返回 (nil, nil)，由调用方决定是否 fail-fast
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

// Resolver 带轮询负载均衡的服务地址解析器：按服务名轮询 Nacos 上的健康实例。
type Resolver struct {
	discovery *Discovery

	mu       sync.Mutex
	counters map[string]uint64 // 服务名 → 轮询计数
}

func NewResolver(discovery *Discovery) *Resolver {
	return &Resolver{discovery: discovery, counters: make(map[string]uint64)}
}

// Resolve 在健康实例间轮询，解析出一个可用实例地址，无可用实例时返回错误
func (r *Resolver) Resolve(serviceName string) (string, error) {
	instances, err := r.discovery.Select(serviceName)
	if err != nil {
		global.Logger.Error("Nacos 服务发现失败", zap.String("service", serviceName), zap.Error(err))
		return "", fmt.Errorf("查询服务 %s 实例失败: %w", serviceName, err)
	}
	if len(instances) == 0 {
		return "", fmt.Errorf("服务 %s 无可用实例", serviceName)
	}
	return instanceAddr(instances[r.next(serviceName, len(instances))]), nil
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
