package settlementapp

import (
	"context"

	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
)

type Channel interface {
	Settle(ctx context.Context, req SettleRequest) error
}

type SettleRequest struct {
	SettleNo  string // 渠道幂等号
	PayNo     string // 关联交易号
	Receivers []Receiver
}

type Receiver struct {
	MerchantID uint   //商户号
	Amount     uint64 // 分
	Desc       string //渠道账单里的分账备注
}

type SettlementViewGateway interface {
	GetOrderSettlement(ctx context.Context, orderNO string) ([]domainsettlement.SettlementSource, int64, error)
}
