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
	"github.com/wsc-zz/service/internal/infrastructure/discovery/nacos"
	"github.com/wsc-zz/service/internal/interfaces/http/gateway"
)

func main() {
	// 网关只做转发，不连数据库
	global.InitViper()
	global.InitZap()

	// 端口默认 9000
	port := global.Conf.Gateway.Port
	if port == 0 {
		port = 9000
	}
	global.Conf.Service.Name = "gateway"
	global.Conf.Service.Port = port

	// 服务发现客户端：Nacos 未启用时为 nil，解析地址时走兜底配置
	discovery, err := nacos.NewDiscovery()
	if err != nil {
		global.Logger.Error("Nacos 服务发现客户端创建失败", zap.Error(err))
		panic(err)
	}

	// 从配置构建路由表 + 兜底地址表
	routes := make([]gateway.Route, 0, len(global.Conf.Gateway.Routes))
	fallback := make(map[string]string)
	for _, rc := range global.Conf.Gateway.Routes {
		routes = append(routes, gateway.Route{Prefix: rc.Prefix, ServiceName: rc.ServiceName})
		if rc.FallbackAddr != "" {
			fallback[rc.ServiceName] = rc.FallbackAddr
		}
	}
	if len(routes) == 0 {
		panic("网关路由为空：请在 config.yaml 的 gateway.routes 中配置转发规则")
	}
	handler := gateway.New(routes, nacos.NewResolver(discovery, fallback))

	// 先监听端口，服务可达后再注册 Nacos
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		global.Logger.Error("端口监听失败", zap.Int("port", port), zap.Error(err))
		panic("端口监听失败: " + err.Error())
	}
	srv := &http.Server{Handler: handler}

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
