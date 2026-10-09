package paymentapp

import "context"

type Channel interface {
	Prepay(ctx context.Context, req PrepayRequest) (payUrl string, err error)
}

type PrepayRequest struct {
	PayNo   string // 支付单号
	Amount  int64  // 金额
	Subject string // 订单标题
}

type OrderGateway interface {
	GetPayableOrder(ctx context.Context, orderNo string) (PayableOrder, error)
	NotifyPaid(ctx context.Context, input NotifyPaidInput) error
}

type PayableOrder struct {
	OrderNo     string
	TotalAmount int64
	UserID      uint
	Status      int
}

type NotifyPaidInput struct {
	PayNo   string
	Amount  int64
	OrderNo string
}
