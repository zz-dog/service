package payment

import "errors"

var (
	//支付单不存在
	ErrPaymentNotFound = errors.New("支付单不存在")
	// 订单不存在
	ErrOrderNotFound = errors.New("订单不存在")

	// 订单当前状态不可支付
	ErrOrderNotPayable = errors.New("订单当前状态不可支付")
	// 触发支付单取消的信号
	ErrOrderCancelled = errors.New("订单已取消")
	// 触发退款的信号
	ErrAmountMismatch          = errors.New("金额与订单不符")
	ErrInvalidPayment          = errors.New("支付单参数不合法")
	ErrInvalidStatusTransition = errors.New("支付单状态不允许该操作")
	//
	ErrInvalidSignature = errors.New("回调签名无效")
	// 对用户而言"支付单已存在"和"支付单不存在"是同一个错误
	ErrInvalidPaymentInput = errors.New("支付单输入参数不合法")
)
