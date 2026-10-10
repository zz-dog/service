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

	"github.com/wsc-zz/service/global"
	nacosconfig "github.com/wsc-zz/service/internal/infrastructure/configcenter/nacos"
	"github.com/wsc-zz/service/internal/infrastructure/discovery/nacos"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	router "github.com/wsc-zz/service/internal/interfaces/http/router"
	"github.com/wsc-zz/service/internal/interfaces/rpc/inventory"
	"github.com/wsc-zz/service/internal/interfaces/rpc/inventorypb"
)

func main() {
	global.InitViper() // 初始化配置
	global.InitZap()   // 初始化日志
	// 配置中心：拉取远程配置覆盖本地；拉取失败时 fail-fast
	nacosCfg, err := nacosconfig.NewConfigClient()
	if err != nil {
		global.Logger.Error("Nacos 配置中心客户端创建失败", zap.Error(err))
		panic(err)
	}
	if err := nacosCfg.Load(); err != nil {
		global.Logger.Error("加载 Nacos 远程配置失败", zap.Error(err))
		panic(err)
	}

	global.InitMysql() //连接数据库
	global.InitRedis() //连接 Redis

	if err := global.DB.AutoMigrate(
		&categorypo.CategoryPO{},
		&categorypo.CategorySpecPO{},
		&specpo.SpecPO{},
		&specpo.SpecValuePO{},
		&productpo.ProductPO{},
		&productpo.SKUPO{},
		&productpo.SKUSpecItemPO{},
		&productpo.ProductMediaPO{},
	); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	r := router.InitCatalogRouter()

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(global.Conf.Catalog.ServicePort))
	if err != nil {
		global.Logger.Error("端口监听失败", zap.Int("port", global.Conf.Catalog.ServicePort), zap.Error(err))
		panic("端口监听失败: " + err.Error())
	}
	srv := &http.Server{Handler: r} // 服务

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			global.Logger.Error("HTTP 服务异常退出", zap.Error(err))
			panic(err)
		}
	}()

	grpcServer := grpc.NewServer()
	inventorypb.RegisterInventoryServiceServer(grpcServer, inventory.NewServer(global.DB))
	reflection.Register(grpcServer)
	grpcListener, err := net.Listen("tcp", ":"+strconv.Itoa(global.Conf.Catalog.RpcPort))

	if err != nil {
		global.Logger.Error("GRPC 端口监听失败", zap.Int("port", global.Conf.Catalog.RpcPort), zap.Error(err))
		panic("GRPC 端口监听失败: " + err.Error())
	}
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			global.Logger.Error("GRPC 服务异常退出", zap.Error(err))
			panic(err)
		}
	}()

	registry, err := nacos.Register()                                              // 注册 Nacos 服务实例
	registry.RpcRegister(global.Conf.Catalog.RpcName, global.Conf.Catalog.RpcPort) // 注册 Nacos RPC 服务实例
	registry.ServiceRegister(global.Conf.Catalog.ServiceName, global.Conf.Catalog.ServicePort)
	if err != nil {
		global.Logger.Error("Nacos 注册失败", zap.Error(err))
	}

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
