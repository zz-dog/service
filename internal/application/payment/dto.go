package paymentapp

import (
	"time"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type PrepayReq struct {
	UserID  uint
	OrderNo string
	Channel domainpayment.Channel
}

type PrepayDto struct {
	PayNo     string                      `json:"payNo"`
	OrderNo   string                      `json:"orderNo"`
	UserID    uint                        `json:"userId"`
	Channel   domainpayment.Channel       `json:"channel"`
	Amount    int64                       `json:"amount"`
	Status    domainpayment.PaymentStatus `json:"status"`
	CreatedAt time.Time                   `json:"createdAt"`
	PaidAt    *time.Time                  `json:"paidAt"`
	PayUrl    string                      `json:"payUrl"`
}

func toDto(p *domainpayment.Payment, payUrl string) *PrepayDto {
	return &PrepayDto{
		PayNo:     p.PayNo,
		OrderNo:   p.OrderNo,
		UserID:    p.UserID,
		Channel:   p.Channel,
		Amount:    p.Amount,
		Status:    p.Status,
		CreatedAt: p.CreatedAt,
		PaidAt:    p.PaidAt,
		PayUrl:    payUrl,
	}
}
