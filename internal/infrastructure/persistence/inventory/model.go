package inventory

import (
	"time"

	domaininventory "github.com/wsc-zz/service/internal/domain/inventory"
)

// LedgerPO 库存流水:每一次扣减/回补都落一行,是幂等、配对、对账三件事的锚点。
// 与扣减库存的 UPDATE 处于同一事务,要么都成功要么都不存在。
type LedgerPO struct {
	ID          uint                   `gorm:"primaryKey"` //
	RequestID   string                 `gorm:"type:varchar(64);uniqueIndex:uniq_req_action_idx,priority:1"`
	Action      domaininventory.Action `gorm:"type:varchar(16);uniqueIndex:uniq_req_action_idx,priority:2"` // deduct / restock
	ItemIndex   int                    `gorm:"uniqueIndex:uniq_req_action_idx,priority:3"`
	ProductID   uint
	SKUCode     string `gorm:"type:varchar(64)"`
	ProductName string
	UnitPrice   int64 // 分
	Quantity    int
	CreatedAt   time.Time
}
