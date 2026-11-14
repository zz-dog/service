package settlement

import "errors"

var (
	ErrInvalidSettleInput      = errors.New("分账输入不合法")
	ErrAmountMismatch          = errors.New("明细金额与订单总额不符,拒绝结算") // 与 NotifyPaid 的金额校验同一精神
	ErrNothingToSettle         = errors.New("纯自营订单,无账可分")       // 若你在 2.2 选了哨兵错误方案
	ErrInvalidStatusTransition = errors.New("当前状态不允许该操作")
	ErrSettlementNotFound      = errors.New("分账单不存在")
	ErrSettleExecuting         = errors.New("分账单正在结算中") // 抢占失败:别人在跑,不是错误也不是成功
	ErrDuplicateSettlement     = errors.New("分账单已存在")   // pay_no/settle_no 唯一索引冲突(仓储翻译),幂等建单据此重查确认
)
