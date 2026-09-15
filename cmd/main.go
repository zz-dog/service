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
	nacosconfig "github.com/wsc-zz/service/internal/infrastructure/configcenter/nacos"
	orderpo "github.com/wsc-zz/service/internal/infrastructure/persistence/order"

	userpo "github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/interfaces/http/router"

	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"

	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
)

// @title           Demo Service API
// @version         1.0
// @description     DDD 架构示例服务（用户 / 订单 / 分类）
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey  ApiKeyAuth
// @in                           header
// @name                         Authorization

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

	global.InitMysql()

	// 2. 自动迁移持久化对象，确保表已创建/更新
	if err := global.DB.AutoMigrate(
		&userpo.UserPO{},
		&orderpo.OrderPO{},
		&orderpo.OrderItemPO{},
		&categorypo.CategoryPO{},
		&categorypo.CategorySpecPO{},
		&specpo.SpecPO{},
		&specpo.SpecValuePO{},
		&productpo.ProductPO{},
		&productpo.SKUPO{},
		&productpo.SKUSpecItemPO{},
	); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 3. 初始化路由
	r := router.InitRouter()

	// 4. 先监听端口，服务可达后再注册 Nacos，避免注册后请求打到尚未监听的实例
	port := global.Conf.Service.Port
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
	global.Logger.Info("服务启动成功", zap.String("name", global.Conf.Service.Name), zap.Int("port", port))

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
