package payment

type PaymentStatus int

const (
	StatusProcessing    PaymentStatus = iota // 支付中
	StatusPaid                               // 已支付
	StatusClosed                             // 已关闭(重复发起预支付时关旧单)
	StatusRefundPending                      // 待退款(支付成功但订单已取消,阶段四写入)
	StatusRefunded                           // 已退款(阶段五)
)

type Channel int

const (
	ChannelWechat Channel = iota // 微信支付
	ChannelAlipay         = 1    // 支付宝
)
