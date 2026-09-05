package orderapp

import (
	"time"
)

type OrderItemInput struct {
	ProductID uint
	SKUCode   string
	Quantity  int
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
	Status   int // 可选，若不传则查询所有状态的订单
}

// OrderItemDTO 订单明细视图
type OrderItemDTO struct {
	ProductID   uint   `json:"productId"`
	SKUCode     string `json:"skuCode"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`    // 单位：分
	Subtotal    int64  `json:"subtotal"` // 小计，单位：分
}

// OrderDTO 订单视图
type OrderDTO struct {
	OrderID          uint           `json:"orderId"`               // 订单ID
	UserID           uint           `json:"userId"`                // 用户ID
	OrderNo          string         `json:"orderNo"`               // 订单编号
	Status           int            `json:"status"`                // 订单状态：0-待支付，1-已支付，2-已取消
	StatusName       string         `json:"statusName"`            // 订单状态名称
	TotalAmount      int64          `json:"totalAmount"`           // 单位：分
	ConsigneeName    string         `json:"consigneeName"`         // 收货人姓名
	ConsigneePhone   string         `json:"consigneePhone"`        // 收货人手机号
	ConsigneeAddress string         `json:"consigneeAddress"`      // 收货人地址
	LogisticsNo      string         `json:"logisticsNo"`           // 物流单号
	Items            []OrderItemDTO `json:"items"`                 // 订单明细
	CreatedAt        time.Time      `json:"createdAt"`             // 创建时间
	UpdatedAt        time.Time      `json:"updatedAt"`             // 更新时间
	PaidAt           *time.Time     `json:"paidAt,omitempty"`      // 支付时间
	CancelledAt      *time.Time     `json:"cancelledAt,omitempty"` // 取消时间
}

// OrderListResult 订单分页结果
type OrderListResult struct {
	List  []OrderDTO `json:"list"`
	Total int        `json:"total"`
}
