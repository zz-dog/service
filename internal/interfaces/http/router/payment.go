package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	"github.com/wsc-zz/service/internal/infrastructure/idgen"
	paymentpo "github.com/wsc-zz/service/internal/infrastructure/persistence/payment"

	alipay "github.com/wsc-zz/service/internal/infrastructure/channel/alipay"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	orderpay "github.com/wsc-zz/service/internal/interfaces/rpc/orderpay"
)

func InitPaymentRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	paymentGroup := r.Group("/api/payment")
	registerPaymentRoutes(paymentGroup)
	return r
}

func registerPaymentRoutes(api *gin.RouterGroup) {
	repo := paymentpo.NewPaymentRepository(global.DB)

	orderClient, err := orderpay.NewClient(global.Conf.Order.RpcAddr, 3*time.Second)
	if err != nil {
		panic(err)
	}
	payNoGen, err := idgen.NewSnowflakeGenerator(1)
	if err != nil {
		panic(err)
	}
	alipayChannel, err := alipay.NewChannel(global.Conf.Payment.Alipay.AppID, global.Conf.Payment.Alipay.PrivateKey, global.Conf.Payment.Alipay.PublicKey)

	paymentSvc := paymentapp.NewService(repo, alipayChannel, orderClient, payNoGen)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)
	api.POST("/prepay", paymentHandler.Prepay)
}
