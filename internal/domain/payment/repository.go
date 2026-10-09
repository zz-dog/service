package payment

import "context"

// Repository 支付仓储接口
type Repository interface {
	// Save 保存支付单
	Save(ctx context.Context, p *Payment) error
	// FindByPayNo 根据支付单号查询支付单
	FindByPayNo(ctx context.Context, payNo string) (*Payment, error)
	// FindProcessingByOrderNo 找同订单"支付中"的单;没有返回 (nil, nil)——预支付复用判定
	FindProcessingByOrderNo(ctx context.Context, orderNo string) (*Payment, error)

	MarkPaid(ctx context.Context, payNo string) (
		bool, error,
	)
}
