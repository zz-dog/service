package orderapp

import "time"

type OrderItemInput struct {
	ProductID   uint
	ProductName string
	Quantity    int
	Price       int64 // 单位：分
}

type CreateOrderInput struct {
	UserID           uint
	Items            []OrderItemInput
	ConsigneeName    string
	ConsigneePhone   string
	ConsigneeAddress string
}

// PayOrderInput 模拟支付用例输入
type PayOrderInput struct {
	OrderID uint
	UserID  uint // 用于校验订单归属
}

// CancelOrderInput 取消订单用例输入
type CancelOrderInput struct {
	OrderID uint
	UserID  uint
}

// GetOrderInput 查询订单详情用例输入
type GetOrderInput struct {
	OrderID uint
	UserID  uint
}

// QueryOrdersInput 分页查询订单用例输入
type QueryOrdersInput struct {
	UserID   uint
	Page     int
	PageSize int
}

// OrderItemDTO 订单明细视图
type OrderItemDTO struct {
	ProductID   uint   `json:"productId"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`    // 单位：分
	Subtotal    int64  `json:"subtotal"` // 小计，单位：分
}

// OrderDTO 订单视图
type OrderDTO struct {
	OrderID          uint           `json:"orderId"`
	UserID           uint           `json:"userId"`
	OrderNo          string         `json:"orderNo"`
	Status           int            `json:"status"`
	StatusName       string         `json:"statusName"`
	TotalAmount      int64          `json:"totalAmount"` // 单位：分
	ConsigneeName    string         `json:"consigneeName"`
	ConsigneePhone   string         `json:"consigneePhone"`
	ConsigneeAddress string         `json:"consigneeAddress"`
	LogisticsNo      string         `json:"logisticsNo"`
	Items            []OrderItemDTO `json:"items"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	PaidAt           *time.Time     `json:"paidAt,omitempty"`
	CancelledAt      *time.Time     `json:"cancelledAt,omitempty"`
}

// OrderListResult 订单分页结果
type OrderListResult struct {
	List     []OrderDTO `json:"list"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}
