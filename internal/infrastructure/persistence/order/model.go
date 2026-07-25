package orderpo

import (
	"time"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
)

// OrderPO 订单持久化对象，对应 orders 表。
type OrderPO struct {
	OrderID     uint   `gorm:"primaryKey;autoIncrement;comment:订单主键"`
	OrderNo     string `gorm:"size:32;uniqueIndex;comment:订单号"`
	UserID      uint   `gorm:"index;comment:下单用户ID"`
	Status      int    `gorm:"tinyint;not null;default:1;comment:订单状态 1待支付 2已支付 3已发货 4已完成 5已取消"`
	TotalAmount int64  `gorm:"comment:总金额(分)"`

	ConsigneeName    string `gorm:"size:32;comment:收货人姓名"`
	ConsigneePhone   string `gorm:"size:20;comment:收货人电话"`
	ConsigneeAddress string `gorm:"size:255;comment:收货地址"`

	LogisticsNo string `gorm:"size:64;default:'';comment:物流单号"`

	CreatedAt   time.Time
	UpdatedAt   time.Time
	PaidAt      *time.Time `gorm:"comment:支付时间"`
	CancelledAt *time.Time `gorm:"comment:取消时间"`

	// 一对多：订单包含的明细行。GORM 通过外键 OrderID 关联。
	Items []OrderItemPO `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

// OrderItemPO 订单明细持久化对象，对应 order_items 表。
type OrderItemPO struct {
	ItemID      uint   `gorm:"primaryKey;autoIncrement;comment:明细主键"`
	OrderID     uint   `gorm:"index;comment:所属订单ID"`
	ProductID   uint   `gorm:"comment:商品ID"`
	ProductName string `gorm:"size:128;comment:商品名称"`
	Quantity    int    `gorm:"comment:购买数量"`
	Price       int64  `gorm:"comment:单价(分)"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 自定义表名（GORM 默认会把 OrderPO 推断成 order_p_o_s）
func (OrderPO) TableName() string     { return "orders" }
func (OrderItemPO) TableName() string { return "order_items" }

// toDomain 将订单 PO 转为领域聚合根（含明细）
func toDomain(po *OrderPO) *domainorder.Order {
	items := make([]domainorder.OrderItem, 0, len(po.Items))
	for _, ip := range po.Items {
		items = append(items, domainorder.OrderItem{
			ProductID:   ip.ProductID,
			ProductName: ip.ProductName,
			Quantity:    ip.Quantity,
			Price:       ip.Price,
		})
	}
	return &domainorder.Order{
		OrderID:          po.OrderID,
		UserID:           po.UserID,
		OrderNo:          po.OrderNo,
		Status:           po.Status,
		TotalAmount:      po.TotalAmount,
		ConsigneeName:    po.ConsigneeName,
		ConsigneePhone:   po.ConsigneePhone,
		ConsigneeAddress: po.ConsigneeAddress,
		LogisticsNo:      po.LogisticsNo,
		Items:            items,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
		PaidAt:           po.PaidAt,
		CancelledAt:      po.CancelledAt,
	}
}
func toPO(o *domainorder.Order) *OrderPO {
	return &OrderPO{
		OrderID:          o.OrderID,
		OrderNo:          o.OrderNo,
		UserID:           o.UserID,
		Status:           o.Status,
		TotalAmount:      o.TotalAmount,
		ConsigneeName:    o.ConsigneeName,
		ConsigneePhone:   o.ConsigneePhone,
		ConsigneeAddress: o.ConsigneeAddress,
		LogisticsNo:      o.LogisticsNo,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		PaidAt:           o.PaidAt,
		CancelledAt:      o.CancelledAt,
	}
}

// toItemsPO 将领域明细列表转为 PO 列表（新增时用，OrderID 由 GORM 级联创建回填）
func toItemsPO(items []domainorder.OrderItem) []OrderItemPO {
	result := make([]OrderItemPO, 0, len(items))
	for _, it := range items {
		result = append(result, OrderItemPO{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}
	return result
}
