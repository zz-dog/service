package payment

import "time"

// PayNoGenerator 支付单号生成端口。order.go:63 注释里预留的"生产换雪花,通过端口注入"在这里落地:
// domain 只依赖接口,雪花实现在 infrastructure/idgen,单测可注入假实现。
type PayNoGenerator interface {
	NextPayNo() string
}

// Payment 支付单:一次支付尝试的完整记录,也是对账的流水。
// 与 order 的关系:多对一(一次订单可多次尝试),靠 order_no 关联,互不持有对方的数据。
type Payment struct {
	PayNo     string  // 业务唯一,幂等锚点
	OrderNo   string  // 关联订单
	UserID    uint    // 归属,"我的支付单"与查单鉴权用
	Channel   Channel // mock / wechat / alipay
	Amount    int64   // 分,建单时已与订单总额核对
	Status    PaymentStatus
	CreatedAt time.Time
	PaidAt    *time.Time
}

// NewPayment 生成支付中状态的支付单。PayNo 由注入的雪花生成器产生(19 位数字,趋势递增,
// 在渠道单号 ≤32 位限制内);唯一索引兜底重复(概率极低但钱的事不留概率)。
func NewPayment(gen PayNoGenerator, orderNo string, userID uint, amount int64, channel Channel) (*Payment, error) {
	if gen == nil || orderNo == "" || userID == 0 || amount <= 0 {
		return nil, ErrInvalidPaymentInput
	}
	return &Payment{
		PayNo:     gen.NextPayNo(),
		OrderNo:   orderNo,
		UserID:    userID,
		Channel:   channel,
		Amount:    amount,
		Status:    StatusProcessing,
		CreatedAt: time.Now(),
		PaidAt:    nil,
	}, nil
}

// 状态机方法,非法迁移一律 ErrInvalidStatusTransition:
// Pay()              支付中 → 已支付
// Close()            支付中 → 已关闭
// MarkRefundPending() 已支付 → 待退款(阶段四)
// MarkRefunded()      待退款 → 已退款(阶段五)

// Pay() 支付中 → 已支付
func (p *Payment) Pay() error {
	if p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusPaid
	now := time.Now()
	p.PaidAt = &now
	return nil
}

// Close()            支付中 → 已关闭
func (p *Payment) Close() error {
	if p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusClosed
	return nil
}

// MarkRefundPending 标记支付单为"待退款"状态,用于阶段四:退款中。仅允许已支付的支付单进入该状态。
func (p *Payment) MarkRefundPending() error {
	if p.Status != StatusPaid {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusRefundPending
	return nil

}

// MarkRefunded 标记支付单为"已退款"状态,用于阶段五:退款成功。仅允许待退款的支付单进入该状态。
func (p *Payment) MarkRefunded() error {
	if p.Status != StatusRefundPending {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusRefunded
	return nil
}
