package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	alipaychannel "github.com/wsc-zz/service/internal/infrastructure/channel/alipay"
	"github.com/wsc-zz/service/internal/infrastructure/channel/mock"
	"github.com/wsc-zz/service/internal/infrastructure/idgen"
	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
	paymentpo "github.com/wsc-zz/service/internal/infrastructure/persistence/payment"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
	orderpay "github.com/wsc-zz/service/internal/interfaces/rpc/orderpay"
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

	orderClient, err := orderpay.NewClient(global.Conf.Order.RpcAddr, 3*time.Second)
	if err != nil {
		panic(err)
	}
	payNoGen, err := idgen.NewSnowflakeGenerator(1) // 单实例固定 1;多实例必须各不相同
	if err != nil {
		panic(err)
	}

	// 渠道装配:payment.channel 选择"心脏",mock 永远是 fallback(沙箱有维护窗口、隧道会断)
	var channel paymentapp.Channel
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
		channelType = domainpayment.ChannelAlipay
	default:
		// mock(含未配置的情况):宁可用假渠道起步,也别带着半截配置上真渠道
		mockCh = mock.NewMockChannel(global.Conf.Payment.PayBaseURL, global.Conf.Payment.MockChannelSecret)
		channel = mockCh
		channelType = domainpayment.ChannelMock
	}

	paymentSvc := paymentapp.NewService(repo, channel, orderClient, payNoGen, channelType)
	h := handler.NewPaymentHandler(paymentSvc, mockCh, alipayCh)

	// 用户操作,走 JWT:userID 从 token 来,金额归属全以订单为准
	blacklist := NewTokenBlacklist()
	payApi := api.Group("/payment", middleware.JWTAuth(blacklist))
	payApi.POST("/prepay", h.Prepay)

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
