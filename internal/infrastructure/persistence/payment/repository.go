package payment

import (
	"context"
	"time"

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

// MarkPaid 条件更新 支付中→已支付,和订单 MarkPaid 同骨架:
// Where status=? 让状态机成为并发裁决者——重复通知、并发双通知都只生效一次(RowsAffected==0)。
// 生效与否看 RowsAffected,不是看内存结构:Update 不会把新值写回 po。
// 注意 Model(&po) 的 po 若只有 PayNo、主键为零值,GORM 不会生成 WHERE,直接 ErrMissingWhereClause——
// 条件必须显式写出来。
func (r *paymentRepository) MarkPaid(ctx context.Context, payNo string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&PaymentPO{}).
		Where("pay_no = ? AND status = ?", payNo, domainpayment.StatusProcessing).
		Updates(map[string]interface{}{
			"status":  int(domainpayment.StatusPaid),
			"paid_at": time.Now(),
		})
	return result.RowsAffected == 1, result.Error
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
