package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	nacosconfig "github.com/wsc-zz/service/internal/infrastructure/configcenter/nacos"
	"github.com/wsc-zz/service/internal/infrastructure/discovery/nacos"
	"github.com/wsc-zz/service/internal/interfaces/http/gateway"
)

// buildRoutes 把配置的路由规则转为网关路由表
func buildRoutes(rcs []global.GatewayRoute) []gateway.Route {
	routes := make([]gateway.Route, 0, len(rcs))
	for _, rc := range rcs {
		routes = append(routes, gateway.Route{Prefix: rc.Prefix, ServiceName: rc.ServiceName})
	}
	return routes
}

// routePrefixes 取路由前缀列表，用于日志输出
func routePrefixes(routes []gateway.Route) []string {
	prefixes := make([]string, 0, len(routes))
	for _, route := range routes {
		prefixes = append(prefixes, route.Prefix)
	}
	return prefixes
}

func main() {
	// 网关只做转发，不连数据库
	global.InitViper()
	global.InitZap()

	// 配置中心：拉取远程配置覆盖本地（路由表可迁至 Nacos）；拉取失败时 fail-fast
	nacosCfg, err := nacosconfig.NewConfigClient()
	if err != nil {
		global.Logger.Error("Nacos 配置中心客户端创建失败", zap.Error(err))
		panic(err)
	}
	if err := nacosCfg.Load(); err != nil {
		global.Logger.Error("加载 Nacos 远程配置失败", zap.Error(err))
		panic(err)
	}

	// 端口默认 9000
	port := global.Conf.Gateway.Port
	if port == 0 {
		port = 9000
	}
	global.Conf.Service.Name = "gateway"
	global.Conf.Service.Port = port

	// 服务发现客户端：网关转发完全依赖 Nacos 解析上游地址，未启用时 fail-fast
	discovery, err := nacos.NewDiscovery()
	if err != nil {
		global.Logger.Error("Nacos 服务发现客户端创建失败", zap.Error(err))
		panic(err)
	}
	if discovery == nil {
		panic("网关依赖 Nacos 服务发现：请设置 nacos.enabled=true")
	}

	// 从配置构建路由表
	routes := buildRoutes(global.Conf.Gateway.Routes)
	if len(routes) == 0 {
		panic("网关路由为空：请在 Nacos 配置中心或 config.yaml 的 gateway.routes 中配置转发规则")
	}
	resolver := nacos.NewResolver(discovery)
	gw := gateway.New(routes, resolver)

	// 监听远程配置变更：热更新路由表，无需重启网关。
	// 空内容/空路由的变更将被忽略，保护现有转发规则
	if err := nacosCfg.Watch(func(content string) {
		if content == "" {
			global.Logger.Warn("远程配置已删除或内容为空，忽略本次变更")
			return
		}
		if err := global.MergeRemoteConfig(content); err != nil {
			global.Logger.Warn("远程配置热更新失败", zap.Error(err))
			return
		}
		newRoutes := buildRoutes(global.Conf.Gateway.Routes)
		if len(newRoutes) == 0 {
			global.Logger.Warn("远程配置网关路由为空，保留现有路由")
			return
		}
		gw.UpdateRoutes(newRoutes)
		global.Logger.Info("网关路由已热更新",
			zap.Int("routes", len(newRoutes)),
			zap.Strings("prefixes", routePrefixes(newRoutes)))
	}); err != nil {
		global.Logger.Warn("注册 Nacos 配置变更监听失败（配置热更新不可用）", zap.Error(err))
	}

	// 先监听端口，服务可达后再注册 Nacos
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		global.Logger.Error("端口监听失败", zap.Int("port", port), zap.Error(err))
		panic("端口监听失败: " + err.Error())
	}
	srv := &http.Server{Handler: gw}

	registry, err := nacos.Register()
	if err != nil {
		global.Logger.Error("Nacos 注册失败", zap.Error(err))
		panic(err)
	}

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			global.Logger.Error("HTTP 服务异常退出", zap.Error(err))
			panic(err)
		}
	}()
	global.Logger.Info("gateway 服务启动成功", zap.Int("port", port))

	// 优雅退出：先注销 Nacos，再关闭 HTTP
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	global.Logger.Info("收到退出信号，开始优雅关停")

	if err := registry.Deregister(); err != nil {
		global.Logger.Warn("Nacos 注销失败", zap.Error(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		global.Logger.Warn("HTTP 服务关闭超时", zap.Error(err))
	}
	global.Logger.Info("gateway 服务已退出")
}
