package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	settlementapp "github.com/wsc-zz/service/internal/application/settlement"
	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
	alipaychannel "github.com/wsc-zz/service/internal/infrastructure/channel/alipay"
	"github.com/wsc-zz/service/internal/infrastructure/channel/mock"
	"github.com/wsc-zz/service/internal/infrastructure/idgen"
	paymentpo "github.com/wsc-zz/service/internal/infrastructure/persistence/payment"
	settlepo "github.com/wsc-zz/service/internal/infrastructure/persistence/settle"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
	orderpay "github.com/wsc-zz/service/internal/interfaces/rpc/orderpay"
	"go.uber.org/zap"
)

// InitPaymentRouter payment 服务的路由入口。网关按 /api/payment 前缀转发到本服务,
// 路径必须带此前缀。
func InitPaymentRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	api := r.Group("/api")
	registerPaymentRoutes(api)
	return r
}

func registerPaymentRoutes(api *gin.RouterGroup) {
	repo := paymentpo.NewPaymentRepository(global.DB)
	settleRepo := settlepo.NewSettlementRepository(global.DB)

	orderClient, err := orderpay.NewClient(global.Conf.Order.RpcAddr, 3*time.Second)
	if err != nil {
		panic(err)
	}
	payNoGen, err := idgen.NewSnowflakeGenerator(1) // 单实例固定 1;多实例必须各不相同
	if err != nil {
		panic(err)
	}

	// 渠道装配:payment.channel 选择"心脏",mock 永远是 fallback(沙箱有维护窗口、隧道会断)。
	// 同一颗心脏,两个视角:预支付和分账是两种能力、两个端口,但适配器同时实现二者,
	// 装配处不用为分账再选一次渠道(mock/alipay 的静态类型各自满足两个接口)
	var channel paymentapp.Channel
	var settleChannel settlementapp.Channel
	var channelType domainpayment.Channel
	var mockCh *mock.MockChannel
	var alipayCh *alipaychannel.Channel
	switch global.Conf.Payment.Channel {
	case "alipay":
		alipayCh, err = alipaychannel.NewChannel(
			global.Conf.Payment.Alipay.AppID,
			global.Conf.Payment.Alipay.PrivateKey,
			global.Conf.Payment.Alipay.PublicKey,
			global.Conf.Payment.NotifyUrl,
		)
		if err != nil {
			panic(err)
		}
		channel = alipayCh
		settleChannel = alipayCh
		channelType = domainpayment.ChannelAlipay
	default:
		// mock(含未配置的情况):宁可用假渠道起步,也别带着半截配置上真渠道
		mockCh = mock.NewMockChannel(global.Conf.Payment.PayBaseURL, global.Conf.Payment.MockChannelSecret, global.Conf.Payment.Settle.MockFail)
		channel = mockCh
		settleChannel = mockCh
		channelType = domainpayment.ChannelMock
	}

	// 结算域与应用服务:结算视图走 order 的 gRPC client(同一个 client 两个端口),
	// 单号生成器复用同一颗雪花(NextPayNo/NextSettleNo 各发各的,全局唯一)
	settleSvc := settlementapp.NewService(
		settleRepo,
		settleChannel,
		orderClient,
		payNoGen,
		global.Conf.Payment.Settle.CommissionRateBps,
		global.Conf.Payment.Settle.MaxRetry,
	)
	paymentSvc := paymentapp.NewService(repo, channel, orderClient, payNoGen, channelType, settleSvc)
	h := handler.NewPaymentHandler(paymentSvc, mockCh, alipayCh)
	settleH := handler.NewSettlementHandler(settleSvc)

	// 结算 job 是后台任务不是路由,放在注册函数里(它属于这个服务的生命周期)
	interval := global.Conf.Payment.Settle.JobIntervalSeconds
	if interval <= 0 {
		interval = 30 // 配置缺失兜底:time.NewTicker 对非正数会 panic
	}
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			if err := settleSvc.RunPending(ctx); err != nil {
				global.Logger.Error("结算 job 异常", zap.Error(err))
			}
			cancel()
		}
	}()

	// 用户操作,走 JWT:userID 从 token 来,金额归属全以订单为准
	blacklist := NewTokenBlacklist()
	payApi := api.Group("/payment", middleware.JWTAuth(blacklist))
	payApi.POST("/prepay", h.Prepay)

	// 管理端:JWTAuth 在前(认证:你是谁),AdminAuth 在后(授权:你够不够格),
	// 鉴权在路由层一次做齐,handler 里不再人人自觉(merchant 教程 2.4 的老话)
	admin := payApi.Group("/admin", middleware.AdminAuth())
	{
		admin.GET("/settlements", settleH.List)                       // 分账单列表(含明细)
		admin.POST("/settlements/:settleNo/execute", settleH.Execute) // 立即执行(和 job 共用抢锁语义)
		admin.POST("/settlements/:settleNo/retry", settleH.Retry)     // 失败单重置回待结算
	}

	// 渠道回调:验签做身份证明,必须在 JWT 组之外——渠道服务器没有用户令牌。
	// 按装配的渠道只注册对应的通知端点
	if mockCh != nil {
		mockApi := api.Group("/payment/mock")
		mockApi.GET("/cashier", h.MockCashier) // 收银台:验签后渲染确认页
		mockApi.POST("/notify", h.MockNotify)  // 渠道异步通知
	} else {
		api.POST("/payment/notify", h.AlipayNotify) // 与 config payment.notify_url 一致
	}
}
