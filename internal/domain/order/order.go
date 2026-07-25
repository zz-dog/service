package order

import (
	"fmt"
	"time"
)

type Order struct {
	OrderID     uint
	UserID      uint
	OrderNo     string
	Status      int
	TotalAmount int64 // 总金额，单位：分

	// 收货信息
	ConsigneeName    string
	ConsigneePhone   string
	ConsigneeAddress string

	// 物流
	LogisticsNo string

	// 订单明细（聚合内的值对象集合）
	Items []OrderItem

	CreatedAt   time.Time
	UpdatedAt   time.Time
	PaidAt      *time.Time
	CancelledAt *time.Time
}

// NewOrder 创建一个待支付订单。
// 作为充血模型的构造函数，集中封装不变量：校验明细、计算总额、生成订单号、初始状态。
func NewOrder(userID uint, items []OrderItem, consigneeName, consigneePhone, consigneeAddress string) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrEmptyOrderItems
	}
	var total int64
	for _, it := range items {
		if err := it.validate(); err != nil {
			return nil, err
		}
		total += it.Subtotal()
	}
	if consigneeName == "" || consigneePhone == "" || consigneeAddress == "" {
		return nil, ErrConsigneeIncomplete
	}

	return &Order{
		UserID:           userID,
		OrderNo:          generateOrderNo(),
		Status:           StatusPending,
		TotalAmount:      total,
		ConsigneeName:    consigneeName,
		ConsigneePhone:   consigneePhone,
		ConsigneeAddress: consigneeAddress,
		Items:            items,
	}, nil
}

// generateOrderNo 生成订单号：yyyyMMddHHmmss + 4 位纳秒后缀。
// 简单方案，足以演示；生产环境可换成雪花算法等，通过端口注入。
func generateOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)
}

// Pay 待支付 -> 已支付
func (o *Order) Pay() error {
	if o.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	o.Status = StatusPaid
	now := time.Now()
	o.PaidAt = &now
	return nil
}

// CanCancel 是否可取消（供展示层判断按钮可用性）
func (o *Order) CanCancel() bool {
	return o.Status == StatusPending || o.Status == StatusPaid
}
