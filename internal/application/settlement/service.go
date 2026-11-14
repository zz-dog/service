package settlementapp

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
)

// settleBatchSize job 每轮最多处理的分账单数:批太大单轮超时,太小扫不过来
const settleBatchSize = 50

type Service struct {
	repo     domainsettlement.Repository
	channel  Channel
	view     SettlementViewGateway
	gen      domainsettlement.SettleNoGenerator
	rateBps  int
	maxRetry int
}

func NewService(repo domainsettlement.Repository, channel Channel, view SettlementViewGateway, gen domainsettlement.SettleNoGenerator, rateBps int, maxRetry int) *Service {
	return &Service{
		repo:     repo,
		channel:  channel,
		view:     view,
		gen:      gen,
		rateBps:  rateBps,
		maxRetry: maxRetry,
	}
}

// EnsureForPayment 幂等建分账单:调一次和调 N 次,库里最多一张单。
// 回调路径的每一环都可能被渠道重放,本方法必须可重入。
// 流程:FindByPayNo 命中 → 直接 nil;未命中 → RPC 拉结算视图 → 金额核对
//
//	→ NewSettlement(纯自营 ErrNothingToSettle 捕获后当"无事发生")→ Save;
//	并发重放撞 pay_no 唯一索引 → 重查一次确认存在后当成功。
func (s *Service) EnsureForPayment(ctx context.Context, payNo, orderNo string, total int64) error {
	existing, err := s.repo.FindByPayNo(ctx, payNo)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // 幂等:单子已在,重放直接成功
	}
	sources, viewTotal, err := s.view.GetOrderSettlement(ctx, orderNo)
	if err != nil {
		return err
	}
	// 双账本对锚:订单侧总额必须等于支付侧总额,对不上宁可不建单
	// (渠道重试会补来这一环),也不建一张错的账。Σ明细行与 total 的核对在
	// NewSettlement 不变量①里还有一道,这里是跨系统的第一道
	if viewTotal != total {
		global.Logger.Error("结算视图总额与支付单不符,拒绝建分账单",
			zap.String("pay_no", payNo), zap.String("order_no", orderNo),
			zap.Int64("view_total", viewTotal), zap.Int64("pay_total", total))
		return domainsettlement.ErrAmountMismatch
	}
	st, err := domainsettlement.NewSettlement(s.gen, payNo, orderNo, total, sources, s.rateBps)
	if errors.Is(err, domainsettlement.ErrNothingToSettle) {
		return nil // 纯自营订单无账可分,不是错误——闭环照常走
	}
	if err != nil {
		return err
	}
	if err := s.repo.Save(ctx, st); err != nil {
		if !errors.Is(err, domainsettlement.ErrDuplicateSettlement) {
			return err
		}
		// 撞唯一索引 = 并发重放正在同时建单。重查一次确认"这张支付单的单子确实存在"
		// 才当幂等成功——不直接当成功:万一冲突来自 settle_no 撞号(不同支付单),
		// 那是要人工看的真异常,吞掉就埋雷了
		existing, ferr := s.repo.FindByPayNo(ctx, payNo)
		if ferr != nil || existing == nil {
			return err
		}
	}
	return nil
}

// Execute 执行一张分账单:抢占 → 渠道 → 终态。job 和管理员"立即执行"共用。
// 幂等/并发的分工:抢占靠 repo.Start 条件更新(谁抢到谁执行);
// 渠道靠 out_request_no(=SettleNo)幂等,重试不会重复分钱;
// 落库靠 MarkSettled/FailRequeue 条件更新,0 行说明单子被并发改走。
func (s *Service) Execute(ctx context.Context, settleNo string) error {
	st, err := s.repo.FindBySettleNo(ctx, settleNo)
	if err != nil {
		return err
	}
	// 双层抢占:领域方法裁决"只有待结算能开跑",仓储条件更新裁决"谁抢到"
	if err := st.Start(); err != nil {
		return err
	}
	won, err := s.repo.Start(ctx, st.ID)
	if err != nil {
		return err
	}
	if !won {
		return domainsettlement.ErrSettleExecuting // 不是失败,是"别人在跑"
	}
	// 渠道侧分账:失败 → FailRequeue 定夺走向(未超限回待结算/超限转失败),仓储落库
	err = s.channel.Settle(ctx, SettleRequest{
		SettleNo:  st.SettleNo,
		PayNo:     st.PayNo,
		Receivers: receiversFrom(st),
	})
	if err != nil {
		toFailed := st.FailRequeue(s.maxRetry)
		if _, rerr := s.repo.FailRequeue(ctx, st.ID, err.Error(), s.maxRetry); rerr != nil {
			global.Logger.Error("分账失败回队落库失败,单子停在结算中",
				zap.String("settle_no", settleNo), zap.Error(rerr))
			return rerr
		}
		if toFailed {
			global.Logger.Error("分账重试超限,转结算失败(终态)",
				zap.String("settle_no", settleNo), zap.String("reason", err.Error()))
		}
		return err
	}
	// 成功落库。这步若失败,单子停在"结算中"而渠道已分了钱——不回滚也不重复调渠道,
	// 重试不会重复分钱(out_request_no 是渠道幂等锚点);卡死的单子人工处理
	if _, err := s.repo.MarkSettled(ctx, st.ID); err != nil {
		global.Logger.Error("分账成功落库失败,单子停在结算中待对账",
			zap.String("settle_no", settleNo), zap.Error(err))
		return err
	}
	return nil
}

// RunPending job 主循环:扫一批待结算单逐张 Execute。
// 单张失败记日志继续——一张卡死不能拖死整批(仿 AutoCancelExpired 的容错骨架)。
func (s *Service) RunPending(ctx context.Context) error {
	list, err := s.repo.FindPending(ctx, settleBatchSize)
	if err != nil {
		return err
	}
	for _, st := range list {
		if err := s.Execute(ctx, st.SettleNo); err != nil {
			if errors.Is(err, domainsettlement.ErrSettleExecuting) {
				continue // 被别的执行者抢走,正常竞争
			}
			global.Logger.Error("执行分账单失败",
				zap.String("settle_no", st.SettleNo), zap.Error(err))
		}
	}
	return nil
}

// Retry 管理员手动重试:结算失败(终态)→ 待结算,给单子第二次生命。
// 与 Execute 共用抢锁语义:领域方法裁决"只有终态能重试",仓储条件更新 4→1 裁决并发,
// 0 行说明单子已被别人动过(如 job 恰好也在处理),按状态非法处理。
// 重试不直接执行:回到待结算,下一轮 job 或管理员 execute 再跑——
// 终态单子重新上路要走正常入口,便于审计和限速。
func (s *Service) Retry(ctx context.Context, settleNo string) error {
	st, err := s.repo.FindBySettleNo(ctx, settleNo)
	if err != nil {
		return err
	}
	if err := st.RequeueForRetry(); err != nil {
		return err
	}
	changed, err := s.repo.MarkRetry(ctx, st.ID)
	if err != nil {
		return err
	}
	if !changed {
		return domainsettlement.ErrInvalidStatusTransition // 并发败北:单子状态已被别人改走
	}
	return nil
}

// List 分账单列表(含明细),管理端对账入口。
// Σ明细 + 留成 == 总额 是对账铁律,列表把三个金额都吐出去,让前端能直接核
func (s *Service) List(ctx context.Context, q domainsettlement.ListQuery) (*SettlementListDto, error) {
	list, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	dtos := make([]SettlementDto, 0, len(list))
	for _, st := range list {
		dtos = append(dtos, toSettlementDto(st))
	}
	return &SettlementListDto{Total: total, List: dtos}, nil
}

// receiversFrom 聚合明细 → 渠道分账接收方。教学版接收方=商户,
// 真渠道要把 MerchantID 换成/加上渠道侧收款账号,聚合结构不变。
func receiversFrom(st *domainsettlement.Settlement) []Receiver {
	receivers := make([]Receiver, 0, len(st.Details))
	for _, d := range st.Details {
		receivers = append(receivers, Receiver{
			MerchantID: d.MerchantID,
			Amount:     uint64(d.Amount),
			Desc:       fmt.Sprintf("订单分账:%s", st.OrderNo),
		})
	}
	return receivers
}
