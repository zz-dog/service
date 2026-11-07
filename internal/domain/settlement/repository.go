package settlement

import "context"

type ListQuery struct {
	Status   int // 0 = 不过滤
	Page     int
	PageSize int
}

// Repository 分账仓储(infrastructure 实现,阶段四写的时候对着抄)
type Repository interface {
	// Save 聚合 + 明细同一事务落库(仿 order.Save 的 items 级联写法)
	Save(ctx context.Context, s *Settlement) error
	FindBySettleNo(ctx context.Context, settleNo string) (*Settlement, error)
	// FindByPayNo 按支付单找分账单;没有返回 (nil, nil)——幂等建单的判定依据
	FindByPayNo(ctx context.Context, payNo string) (*Settlement, error)
	List(ctx context.Context, q ListQuery) ([]*Settlement, int64, error)
	// 条件更新四件套:WHERE status=? 看 RowsAffected,审核/支付同骨架。
	// 状态机方法裁决合法性,仓储管并发——各管一段。
	Start(ctx context.Context, id uint) (bool, error)                                    // 1→2
	MarkSettled(ctx context.Context, id uint) (bool, error)                              // 2→3
	FailRequeue(ctx context.Context, id uint, reason string, maxRetry int) (bool, error) // 2→1 或 2→4
	MarkRetry(ctx context.Context, id uint) (bool, error)                                // 4→1
	FindPending(ctx context.Context, limit int) ([]*Settlement, error)                   // job 扫描待结算
}
