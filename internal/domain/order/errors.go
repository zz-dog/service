package order

import "errors"

var (
	ErrOrderNotFound           = errors.New("订单不存在")
	ErrInvalidStatusTransition = errors.New("订单当前状态不允许该操作")
	ErrOrderAlreadyCancelled   = errors.New("订单已取消，不可重复操作")
	ErrEmptyOrderItems         = errors.New("订单商品明细不能为空")
	ErrInvalidQuantity         = errors.New("商品数量必须大于0")
	ErrInvalidPrice            = errors.New("商品单价必须大于0")
	ErrConsigneeIncomplete     = errors.New("收货信息不完整")
	ErrEmptyLogisticsNo        = errors.New("物流单号不能为空")
)
