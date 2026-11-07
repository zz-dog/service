package settlementapp

import (
	"time"

	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
)

// SettlementDetailDto 分账明细对外视图
type SettlementDetailDto struct {
	MerchantID uint  `json:"merchantId"`
	Amount     int64 `json:"amount"`
}

// SettlementDto 分账单对外视图,含明细(管理端列表/详情共用一张皮)
type SettlementDto struct {
	SettleNo         string                `json:"settleNo"`
	PayNo            string                `json:"payNo"`
	OrderNo          string                `json:"orderNo"`
	TotalAmount      int64                 `json:"totalAmount"`
	SettleAmount     int64                 `json:"settleAmount"`
	CommissionAmount int64                 `json:"commissionAmount"`
	RateBps          int                   `json:"rateBps"`
	Status           int                   `json:"status"`
	StatusName       string                `json:"statusName"`
	RetryCount       int                   `json:"retryCount"`
	FailReason       string                `json:"failReason"`
	CreatedAt        time.Time             `json:"createdAt"`
	SettledAt        *time.Time            `json:"settledAt"`
	Details          []SettlementDetailDto `json:"details"`
}

// SettlementListDto 分页列表:总数 + 当前页(对账先看 total,再翻页)
type SettlementListDto struct {
	Total int64           `json:"total"`
	List  []SettlementDto `json:"list"`
}

func toSettlementDto(st *domainsettlement.Settlement) SettlementDto {
	details := make([]SettlementDetailDto, 0, len(st.Details))
	for _, d := range st.Details {
		details = append(details, SettlementDetailDto{
			MerchantID: d.MerchantID,
			Amount:     d.Amount,
		})
	}
	return SettlementDto{
		SettleNo:         st.SettleNo,
		PayNo:            st.PayNo,
		OrderNo:          st.OrderNo,
		TotalAmount:      st.TotalAmount,
		SettleAmount:     st.SettleAmount,
		CommissionAmount: st.CommissionAmount,
		RateBps:          st.RateBps,
		Status:           int(st.Status),
		StatusName:       domainsettlement.StatusName(st.Status),
		RetryCount:       st.RetryCount,
		FailReason:       st.FailReason,
		CreatedAt:        st.CreatedAt,
		SettledAt:        st.SettledAt,
		Details:          details,
	}
}
