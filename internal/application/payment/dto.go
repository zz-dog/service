package paymentapp

import (
	"time"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type PrepayDto struct {
	PayNo     string     `json:"payNo"`
	OrderNo   string     `json:"orderNo"`
	UserID    uint       `json:"userId"`
	Channel   int        `json:"channel"` // domainpayment.Channel 数值(mock=2 / alipay=1),swagger 可解析
	Amount    int64      `json:"amount"`
	Status    int        `json:"status"` // domainpayment.PaymentStatus 数值
	CreatedAt time.Time  `json:"createdAt"`
	PaidAt    *time.Time `json:"paidAt"`
	PayUrl    string     `json:"payUrl"`
}

func toDto(p *domainpayment.Payment, payUrl string) *PrepayDto {
	return &PrepayDto{
		PayNo:     p.PayNo,
		OrderNo:   p.OrderNo,
		UserID:    p.UserID,
		Channel:   int(p.Channel),
		Amount:    p.Amount,
		Status:    int(p.Status),
		CreatedAt: p.CreatedAt,
		PaidAt:    p.PaidAt,
		PayUrl:    payUrl,
	}
}
