package payment

import (
	"time"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
)

// PaymentPO 支付单。一张表就是一条流水:幂等锚点(pay_no)与资金状态同聚合,
// 单行条件更新天然原子,不需要库存那样的旁路流水表。
type PaymentPO struct {
	ID        uint   `gorm:"primaryKey"`
	PayNo     string `gorm:"type:varchar(64);uniqueIndex"`
	OrderNo   string `gorm:"type:varchar(64);index"` // 对账配对键:按订单号找支付记录
	UserID    uint
	Channel   domainpayment.Channel `gorm:"type:varchar(16)"`
	Amount    int64                 // 分
	Status    domainpayment.PaymentStatus
	CreatedAt time.Time
	PaidAt    *time.Time
}

func (PaymentPO) TableName() string { return "payments" }
