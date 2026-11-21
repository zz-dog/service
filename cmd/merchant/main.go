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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/wsc-zz/service/global"
	nacosconfig "github.com/wsc-zz/service/internal/infrastructure/configcenter/nacos"
	"github.com/wsc-zz/service/internal/infrastructure/discovery/nacos"
	merchantpo "github.com/wsc-zz/service/internal/infrastructure/persistence/merchant"
	"github.com/wsc-zz/service/internal/interfaces/http/router"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantpb"
	merchantrpc "github.com/wsc-zz/service/internal/interfaces/rpc/merchantrpc"
)

//	@title						Merchant Service API
//	@version					1.0
//	@description				商户入驻服务（申请提交 / 人工审核 / 店铺管理）。host 为网关端口，经网关按前缀转发
//	@host						localhost:9000
//	@BasePath					/api
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization

func main() {
	// 1. 初始化基础设施：配置、日志、远程配置、数据库
	global.InitViper()
	global.InitZap()

	// 配置中心：拉取远程配置覆盖本地（远端键覆盖本地同名键，环境变量仍最高优先）；
	// 启用了 Nacos 但拉取失败时 fail-fast，避免拿着本地旧配置悄悄启动
	nacosCfg, err := nacosconfig.NewConfigClient()
	if err != nil {
		global.Logger.Error("Nacos 配置中心客户端创建失败", zap.Error(err))
		panic(err)
	}
	if err := nacosCfg.Load(); err != nil {
		global.Logger.Error("加载 Nacos 远程配置失败", zap.Error(err))
		panic(err)
	}

	global.InitMysql() // 连接数据库
	global.InitRedis() // 连接 Redis（JWT 黑名单要用）

	// 2. 自动迁移持久化对象，确保表已创建/更新
	if err := global.DB.AutoMigrate(
		&merchantpo.MerchantApplicationPO{},
		&merchantpo.MerchantPO{},
		&merchantpo.MemberPO{},
	); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	// 一人一店 → N:N 的存量迁移:老 merchants.user_id 搬进 merchant_members(role=owner),
	// 搬完删列;幂等,重复启动安全。迁移失败 fail-fast——半搬不搬的状态不能带病上线
	if err := merchantpo.MigrateLegacyOwnership(global.DB); err != nil {
		global.Logger.Error("存量归属迁移失败", zap.Error(err))
		panic("存量归属迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 3. 初始化路由
	r := router.InitMerchantRouter()

	// 4. 先监听端口，服务可达后再注册 Nacos，避免注册后请求打到尚未监听的实例
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(global.Conf.Merchant.ServicePort))
	if err != nil {
		global.Logger.Error("端口监听失败", zap.Int("port", global.Conf.Merchant.ServicePort), zap.Error(err))
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

	// gRPC:catalog 发布商品前按 userId 验商户身份(9085),与 HTTP 双端口共存(仿 cmd/catalog)
	grpcServer := grpc.NewServer()
	merchantpb.RegisterMerchantServiceServer(grpcServer, merchantrpc.NewServer(global.DB))
	reflection.Register(grpcServer)
	grpcListener, err := net.Listen("tcp", ":"+strconv.Itoa(global.Conf.Merchant.RpcPort))
	if err != nil {
		global.Logger.Error("GRPC 端口监听失败", zap.Int("port", global.Conf.Merchant.RpcPort), zap.Error(err))
		panic("GRPC 端口监听失败: " + err.Error())
	}
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			global.Logger.Error("GRPC 服务异常退出", zap.Error(err))
			panic(err)
		}
	}()
	global.Logger.Info("服务启动成功", zap.String("name", global.Conf.Merchant.ServiceName), zap.Int("port", global.Conf.Merchant.ServicePort))

	// 注册 Nacos:Register 返回后立刻判 err——注册失败意味着网关发现不了本服务,
	// 带着错误继续启动只会把"注册失败"埋成线上灵异问题(此坑 payment 阶段踩过)。
	// RPC 先注册(HTTP 注册是网关流量的开关,RPC 注册是 catalog 拨号的开关,缺一不可)
	if err := registry.RpcRegister(global.Conf.Merchant.RpcName, global.Conf.Merchant.RpcPort); err != nil {
		global.Logger.Error("Nacos RPC 服务注册失败", zap.Error(err))
		panic(err)
	}
	if err := registry.ServiceRegister(global.Conf.Merchant.ServiceName, global.Conf.Merchant.ServicePort); err != nil {
		global.Logger.Error("Nacos 服务注册失败", zap.Error(err))
		panic(err)
	}

	// 5. 等待退出信号：先注销 Nacos（停止接入新流量），再优雅关闭 HTTP（处理完存量请求）
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
	global.Logger.Info("服务已退出")
}
