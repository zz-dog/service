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

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/infrastructure/discovery/nacos"
	userpo "github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	identityrouter "github.com/wsc-zz/service/internal/interfaces/http/router"
)

func main() {
	global.InitViper()
	global.InitZap()
	global.InitMysql()

	// identity 服务默认端口 8081（config.yaml 的 service.port 属于 main 服务），可用 SERVICE_PORT 覆盖
	port := 8081
	if value := os.Getenv("SERVICE_PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			port = parsed
		}
	}
	global.Conf.Service.Name = "identity"
	global.Conf.Service.Port = port

	if err := global.DB.AutoMigrate(&userpo.UserPO{}); err != nil {
		global.Logger.Error("用户表迁移失败", zap.Error(err))
		panic("用户表迁移失败: " + err.Error())
	}

	r := gin.Default()
	identityrouter.RegisterIdentityRoutes(r.Group("/api"))

	// 先监听端口，服务可达后再注册 Nacos，避免注册后请求打到尚未监听的实例
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		global.Logger.Error("端口监听失败", zap.Int("port", port), zap.Error(err))
		panic("端口监听失败: " + err.Error())
	}
	srv := &http.Server{Handler: r}

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
	global.Logger.Info("identity 服务启动成功", zap.Int("port", port))

	// 等待退出信号：先注销 Nacos（停止接入新流量），再优雅关闭 HTTP（处理完存量请求）
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
	global.Logger.Info("identity 服务已退出")
}
