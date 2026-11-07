package settlement

import (
	"slices"
	"time"
)

// SettleNoGenerator 结算单号生成端口(仿 PayNoGenerator;实现在 infrastructure/idgen,
// 复用同一颗雪花再开一个方法或包一层,单号格式自定、全局唯一即可)
type SettleNoGenerator interface {
	NextSettleNo() string
}

// SettlementSource 结算输入:一行订单明细的归属快照(阶段一的成果)
type SettlementSource struct {
	MerchantID uint  // 0 = 平台自营,不参与分账
	Amount     int64 // 行小计,分
}

// SettlementDetail 分账明细(值对象):分给一个接收方的一笔钱。
// 教学版接收方 = 商户;真实渠道里 receiver 是渠道侧账号,聚合结构不变。
type SettlementDetail struct {
	MerchantID uint  // 接收商户
	Amount     int64 // 商户应得 = 组内小计 − 佣金(减法,不许再乘一次)
}

// Settlement 分账单:一笔支付的资金二次分配记录,也是对账的流水。
// 与 payment 的关系:PayNo 一对一(一笔支付最多一张分账单,唯一索引落死)。
type Settlement struct {
	ID               uint   // 数据库自增主键,Save 回填;条件更新四件套靠它定位
	SettleNo         string // 渠道幂等号 out_request_no
	PayNo            string // 关联支付单,唯一索引:建单幂等的最终防线
	OrderNo          string
	TotalAmount      int64 // 订单实付,对账锚点
	SettleAmount     int64 // Σ明细,实际分出去的
	CommissionAmount int64 // 平台留成 = TotalAmount − SettleAmount(佣金+自营部分,冗余存,报表友好)
	RateBps          int   // 费率快照,基点(500 = 5.00%)。存整数不存 float:钱的计算全程整数,payment 教程的老规矩
	Status           SettleStatus
	RetryCount       int
	FailReason       string
	CreatedAt        time.Time
	SettledAt        *time.Time
	Details          []SettlementDetail
}

// NewSettlement 归组 + 算钱,纯函数可单测。
// 不变量:
//  1. Σ源金额 == total,对不上直接拒绝——上游给错数宁可炸在这里,也不建一张错的账;
//  2. merchantId=0 的行不进明细(自营留在平台,想清楚②);
//  3. 每个商户组:佣金 = 组内小计 × rateBps / 10000(整除截断),
//     应得 = 组内小计 − 佣金。减法保证 Σ明细 + 留成 == total 恒成立;
//  4. 所有行都是自营 → 无账可分,返回 ErrNothingToSettle,调用方捕获后当"无事发生"。
//     (选哨兵错误而非约定值:调用方要能区分"建了"和"没建",返回 nil 容易被当 bug)
func NewSettlement(gen SettleNoGenerator, payNo, orderNo string, total int64, sources []SettlementSource, rateBps int) (*Settlement, error) {
	// 第一遍:归组 + 对账。同商户行小计先合并,佣金作用在"组内小计"上——
	// 按行各算会因截断让平台少收:99+99 按行 4+4=8,按组 198*500/10000=9
	subtotals := make(map[uint]int64, len(sources))
	var sourceTotal int64
	for _, src := range sources {
		sourceTotal += src.Amount
		if src.MerchantID == 0 {
			continue // 自营行,不进明细
		}
		subtotals[src.MerchantID] += src.Amount
	}

	// 不变量①:Σ源金额 == total,对不上直接拒绝——上游给错数宁可炸在这里,也不建一张错的账
	if sourceTotal != total {
		return nil, ErrAmountMismatch
	}
	// 不变量④:全是自营 → 无账可分,返回哨兵错误;
	// 调用方捕获后当"无事发生"(选哨兵而非约定值:调用方要能区分"建了"和"没建",nil 容易被当 bug)
	if len(subtotals) == 0 {
		return nil, ErrNothingToSettle
	}

	// map 遍历无序,同一笔单子两次建单明细顺序会抖;按 merchantID 排序让结果确定,
	// 单测断言、落库、对账逐条比对都受益
	merchantIDs := make([]uint, 0, len(subtotals))
	for id := range subtotals {
		merchantIDs = append(merchantIDs, id)
	}
	slices.Sort(merchantIDs)

	// 第二遍:按组算钱,一组一条明细
	details := make([]SettlementDetail, 0, len(merchantIDs))
	settleAmount := int64(0) // Σ明细,实际分给商户的
	for _, merchantID := range merchantIDs {
		subtotal := subtotals[merchantID]
		commission := subtotal * int64(rateBps) / 10000 // 整除截断,零头归商户(平台让利)
		amount := subtotal - commission                 // 减法补另一侧,Σ明细 + 留成 == total 恒成立
		details = append(details, SettlementDetail{
			MerchantID: merchantID,
			Amount:     amount,
		})
		settleAmount += amount
	}

	return &Settlement{
		SettleNo:         gen.NextSettleNo(),
		PayNo:            payNo,
		OrderNo:          orderNo,
		TotalAmount:      total,
		SettleAmount:     settleAmount,
		CommissionAmount: total - settleAmount, // 平台留成 = 佣金 + 自营部分
		Status:           SettleStatusPending,
		RateBps:          rateBps,
		Details:          details,
	}, nil
}

// Start 待结算 → 结算中。
// 这里只裁决状态合法性;并发下"job 和手动重试同时抢"由仓储条件更新兜底——
// 双层各管一段:领域管规则,仓储管并发,和 Payment.Pay() 同款结构。
func (s *Settlement) Start() error {
	if s.Status != SettleStatusPending {
		return ErrInvalidStatusTransition
	}
	s.Status = SettleStatusExecuting
	return nil
}

// Succeed 结算中 → 结算成功,写 SettledAt(内存态;落库走 repo.MarkSettled 条件更新)。
func (s *Settlement) Succeed() error {
	if s.Status != SettleStatusExecuting {
		return ErrInvalidStatusTransition
	}
	now := time.Now()
	s.Status = SettleStatusSuccess
	s.SettledAt = &now
	return nil
}

// FailRequeue 结算中 → 渠道失败:RetryCount+1,未超限回待结算(下一轮 job 再来),
// 达到 maxRetry 转结算失败(终态)。返回是否转成终态,调用方据此定告警级别。
// 与仓储 FailRequeue 的 SQL 分支是同一条算术规则(retry_count+1 与 maxRetry 比大小),
// 两处结论必然一致;并发竞争以仓储条件更新为准,这里的裁决服务内存态与调用方判断。
func (s *Settlement) FailRequeue(maxRetry int) bool {
	s.RetryCount++
	if s.RetryCount < maxRetry {
		s.Status = SettleStatusPending
		return false
	}
	s.Status = SettleStatusFailed
	return true
}

// RequeueForRetry 结算失败 → 待结算。管理员手动重试,给终态单子第二次生命;
// 仓储侧 MarkRetry 同步清掉 fail_reason。
func (s *Settlement) RequeueForRetry() error {
	if s.Status != SettleStatusFailed {
		return ErrInvalidStatusTransition
	}
	s.Status = SettleStatusPending
	return nil
}
