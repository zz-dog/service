package paymentapp

import (
	"context"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
)

type Service struct {
	repo     domainpayment.Repository
	channel  Channel
	order    OrderGateway
	payNoGen domainpayment.PayNoGenerator
}

func NewService(repo domainpayment.Repository, channel Channel, order OrderGateway, payNoGen domainpayment.PayNoGenerator) *Service {
	return &Service{repo: repo, channel: channel, order: order, payNoGen: payNoGen}
}

func (s *Service) Prepay(ctx context.Context, req PrepayReq) (dto *PrepayDto, err error) {
	o, err := s.order.GetPayableOrder(ctx, req.OrderNo)
	if err != nil {
		return nil, err
	}
	if o.UserID != req.UserID {
		return nil, domainpayment.ErrOrderNotFound
	}
	if o.Status != domainorder.StatusPending {
		return nil, domainpayment.ErrOrderNotPayable
	}
	p, err := s.repo.FindProcessingByOrderNo(ctx, req.OrderNo)
	if err != nil {
		return nil, err
	}
	// 如果 payment 不存在，则创建
	if p == nil {
		p, err = domainpayment.NewPayment(s.payNoGen, req.OrderNo, req.UserID, o.TotalAmount, req.Channel)
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
		Subject: "支付订单：" + req.OrderNo,
	})
	if err != nil {
		return nil, err
	}
	return toDto(p, payUrl), nil
}

func (s *Service) HandleChannelNotify(ctx context.Context, payNo string) error {
	p, err := s.repo.FindByPayNo(ctx, payNo)
	if err != nil {
		return err
	}
	if p.Status != domainpayment.StatusProcessing {
		return domainpayment.ErrInvalidStatusTransition
	}
	// 更新支付单状态为已支付
	_, err = s.repo.MarkPaid(ctx, p.OrderNo)
	if err != nil {
		return err
	}
	// 通知订单支付成功
	err = s.order.NotifyPaid(ctx, NotifyPaidInput{
		Amount:  p.Amount,
		OrderNo: p.OrderNo,
		PayNo:   p.PayNo,
	})
	if err != nil {
		return err
	}
	return nil
}
