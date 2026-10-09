package payment

import (
	"context"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *paymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) FindByPayNo(ctx context.Context, payNo string) (*domainpayment.Payment, error) {
	var po PaymentPO
	err := r.db.WithContext(ctx).First(&po, "pay_no = ?", payNo).Error
	if err != nil {
		return nil, err
	}
	return toDomain(po), nil
}

func (r *paymentRepository) Save(ctx context.Context, p *domainpayment.Payment) error {
	po := toPO(*p)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *paymentRepository) FindProcessingByOrderNo(ctx context.Context, orderNo string) (*domainpayment.Payment, error) {
	var po PaymentPO
	err := r.db.WithContext(ctx).First(&po, "order_no = ? and status = ?", orderNo, domainpayment.StatusProcessing).Error
	if err != nil {
		return nil, err
	}
	return toDomain(po), nil
}

// MarkPaid 更新支付状态为已支付
func (r *paymentRepository) MarkPaid(ctx context.Context, payNo string) (bool, error) {
	po := PaymentPO{PayNo: payNo}
	err := r.db.WithContext(ctx).Model(&po).Update("status", domainpayment.StatusPaid).Error
	if err != nil {
		return false, err
	}
	return po.Status == domainpayment.StatusPaid, nil
}
func toDomain(po PaymentPO) *domainpayment.Payment {
	return &domainpayment.Payment{
		PayNo:   po.PayNo,
		OrderNo: po.OrderNo,
		UserID:  po.UserID,
		Amount:  po.Amount,
		Channel: domainpayment.Channel(po.Channel),
		Status:  domainpayment.PaymentStatus(po.Status),
		PaidAt:  po.PaidAt,
	}
}

func toPO(p domainpayment.Payment) PaymentPO {
	return PaymentPO{
		PayNo:   p.PayNo,
		OrderNo: p.OrderNo,
		UserID:  p.UserID,
		Amount:  p.Amount,
		Channel: domainpayment.Channel(p.Channel),
		Status:  domainpayment.PaymentStatus(p.Status),
		PaidAt:  p.PaidAt,
	}
}
