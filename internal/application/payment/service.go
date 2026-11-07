package paymentapp

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	settlementapp "github.com/wsc-zz/service/internal/application/settlement"
	domainorder "github.com/wsc-zz/service/internal/domain/order"
	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type Service struct {
	repo        domainpayment.Repository
	channel     Channel
	// channelType 本服务装配的渠道类型,建支付单时写进流水——
	// 渠道由配置选择(payment.channel),不由前端每次请求指定
	channelType domainpayment.Channel
	order       OrderGateway
	payNoGen    domainpayment.PayNoGenerator
	// settle 分账应用服务:回调闭环的最后一环(建分账单)。
	// 依赖方向 paymentapp → settlementapp,settlementapp 不回头 import,无环
	settle *settlementapp.Service
}

func NewService(repo domainpayment.Repository, channel Channel, order OrderGateway, payNoGen domainpayment.PayNoGenerator, channelType domainpayment.Channel, settle *settlementapp.Service) *Service {
	return &Service{repo: repo, channel: channel, channelType: channelType, order: order, payNoGen: payNoGen, settle: settle}
}

// Prepay 预支付:生成(或复用)支付单,向渠道下单,返回收银台跳转地址。
// userID 来自 JWT;金额、归属全部以订单为准——收了前端的金额也不能信。
func (s *Service) Prepay(ctx context.Context, userID uint, orderNo string) (dto *PrepayDto, err error) {
	o, err := s.order.GetPayableOrder(ctx, orderNo)
	if err != nil {
		return nil, err // ErrOrderNotFound / 其他(适配器已翻译)
	}
	// 归属校验:不是你的订单,按"不存在"返回——不泄露它存在
	if o.UserID != userID {
		return nil, domainpayment.ErrOrderNotFound
	}
	// 可付策略在消费方裁决:只有待支付订单能发起支付
	if o.Status != domainorder.StatusPending {
		return nil, domainpayment.ErrOrderNotPayable
	}
	p, err := s.repo.FindProcessingByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	// 如果 payment 不存在，则创建;已存在则复用(重复点击/网络重试安全)
	if p == nil {
		p, err = domainpayment.NewPayment(s.payNoGen, orderNo, userID, o.TotalAmount, s.channelType)
		if err != nil {
			return nil, domainpayment.ErrInvalidPayment
		}
		if err := s.repo.Save(ctx, p); err != nil {
			return nil, err
		}
	}
	payUrl, err := s.channel.Prepay(ctx, PrepayRequest{
		Amount:  p.Amount,
		PayNo:   p.PayNo,
		Subject: "支付订单：" + orderNo,
	})
	if err != nil {
		return nil, err
	}
	return toDto(p, payUrl), nil
}

// HandleChannelNotify 渠道回调:验签已在 handler 完成,这里只管状态机。
// amount 是渠道通知的实付金额(分);0 表示该渠道无金额可核(mock 收银台按支付单
// 原额支付,身份证明就是验签)。真实渠道必须传,与支付单不符 → ErrAmountMismatch,
// 拒绝记账——这是金额防线的渠道侧那一半,挡的是通知被伪造/错配。
// 返回 nil = 通知已受理,渠道不再重发;返回错误 = 渠道会重试(mock 不重试,真实渠道按退避重发)。
//
// 闭环补全式:渠道重试不是异常,是常态。回调处理的心脏从"一次做完"变成
// "每次都把能推进的推进完"——记账、通知订单、建分账单三步各自幂等,
// 任何一步曾经失败,下一次重放都会把它补上,整体"重放到最远"。
func (s *Service) HandleChannelNotify(ctx context.Context, payNo string, amount int64) error {
	p, err := s.repo.FindByPayNo(ctx, payNo)
	if err != nil {
		return err
	}
	switch p.Status {
	case domainpayment.StatusRefundPending, domainpayment.StatusRefunded:
		return nil // 退款态:钱的事走退款流程,闭环到此为止,不再推进也不建分账
	case domainpayment.StatusClosed:
		global.Logger.Warn("收到已关闭支付单的回调", zap.String("pay_no", payNo))
		return nil // 止住渠道重发;真实系统这里要人工看一眼
	}
	// StatusProcessing(待记账)和 StatusPaid(记账过但闭环可能没走完的重放)都继续
	if p.Status == domainpayment.StatusProcessing {
		if amount != 0 && amount != p.Amount {
			global.Logger.Error("回调金额与支付单不符,拒绝记账",
				zap.String("pay_no", payNo),
				zap.Int64("notify_amount", amount), zap.Int64("pay_amount", p.Amount))
			return domainpayment.ErrAmountMismatch // 绝不记账,回 fail 人工看
		}
		// 条件更新 支付中→已支付:并发双通知/渠道重放只有一人生效(RowsAffected==1)。
		// 生效与否看 RowsAffected,不是看内存结构;参数是 payNo 不是 orderNo(仓储按 pay_no 过滤)。
		if _, err := s.repo.MarkPaid(ctx, payNo); err != nil {
			return err
		}
		// changed=false 不提前退出:抢到的那位可能死在记账之后的任何一步,
		// 重放要把没做完的闭环补完——这正是"闭环补全"式与"一次做完"式的分界
	}

	// 闭环两步,各自幂等,任一步失败返回 err → 渠道重试 → 重放再来:
	// NotifyPaid 服务端幂等(已支付重放返回成功);EnsureForPayment 幂等建单(pay_no 唯一索引)。
	// 顺序不能反:订单状态是用户可见的强状态,分账是后台资金动作——
	// "订单没支付、钱已分出"是事故,"订单已支付、分账单没建"只是无害中间态
	err = s.order.NotifyPaid(ctx, NotifyPaidInput{
		Amount:  p.Amount,
		OrderNo: p.OrderNo,
		PayNo:   p.PayNo,
	})
	switch {
	case err == nil:
		// 订单已推进,闭环最后一步:建分账单(纯自营单内部当"无事发生")
	case errors.Is(err, domainpayment.ErrOrderCancelled):
		// 竞态败北:支付成功但订单已被取消(自动取消或用户取消)。取消路径会回补库存,
		// 所以这里只剩钱的事——转待退款,阶段五的退款 job 处理。对渠道返回成功:
		// 通知已受理,钱的问题是我们自己的事,不能让渠道无限重发
		if _, merr := s.repo.MarkRefundPending(ctx, payNo); merr != nil {
			global.Logger.Error("转待退款失败", zap.String("pay_no", payNo), zap.Error(merr))
		}
		global.Logger.Warn("订单已取消,支付单转待退款",
			zap.String("pay_no", payNo), zap.String("order_no", p.OrderNo))
		return nil
	case errors.Is(err, domainpayment.ErrAmountMismatch):
		// 金额防线被击穿(理论上到不了这):支付单已标记已支付但必须人工介入。
		// 返回 err 招来重试也无济于事,回成功止住渠道重发
		global.Logger.Error("支付通知金额不符,人工介入", zap.String("pay_no", payNo), zap.Error(err))
		return nil
	default:
		// 结果未知或基础设施失败:返回 err 让渠道重试,重放会从"已支付"一路补完闭环
		// ——三步各自幂等,重试安全,这正是闭环补全式的底气(不再依赖对账兜底)
		global.Logger.Warn("通知订单支付结果失败,待渠道重试",
			zap.String("pay_no", payNo), zap.String("order_no", p.OrderNo), zap.Error(err))
		return err
	}
	// EnsureForPayment 失败(如 order RPC 拉结算视图失败)→ 返回 err → 渠道重试 →
	// 重放:MarkPaid 0 行、NotifyPaid 幂等成功、EnsureForPayment 重来
	return s.settle.EnsureForPayment(ctx, p.PayNo, p.OrderNo, p.Amount)
}
