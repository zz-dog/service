package settlepo

import "time"

// SettlementPO 分账单。pay_no 唯一索引是"一笔支付一张分账单"的最终防线:
// 应用层 FindByPayNo 检查只是友好路径,并发回调重放只有索引拦得住。
type SettlementPO struct {
	ID               uint   `gorm:"primaryKey"`
	SettleNo         string `gorm:"type:varchar(64);uniqueIndex"`
	PayNo            string `gorm:"type:varchar(64);uniqueIndex"`
	OrderNo          string `gorm:"type:varchar(64);index"`
	TotalAmount      int64  // 对账锚点
	SettleAmount     int64
	CommissionAmount int64
	RateBps          int
	Status           int `gorm:"index"` // 列表按状态过滤,索引值得给
	RetryCount       int
	FailReason       string `gorm:"type:varchar(255)"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	SettledAt        *time.Time

	Details []SettlementDetailPO `gorm:"foreignKey:SettlementID;constraint:OnDelete:CASCADE"`
}

// SettlementDetailPO 分账明细。教学版接收方=商户;真渠道要把 merchant_id 换成/加上渠道账号。
type SettlementDetailPO struct {
	ID           uint `gorm:"primaryKey"`
	SettlementID uint `gorm:"index"`
	MerchantID   uint `gorm:"index"` // "我的分账"查询键(进阶 2)
	Amount       int64
	CreatedAt    time.Time
}

func (SettlementPO) TableName() string       { return "settlements" }
func (SettlementDetailPO) TableName() string { return "settlement_details" }
