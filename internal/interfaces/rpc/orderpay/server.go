package orderpay

import (
	"context"
	"fmt"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
	"github.com/wsc-zz/service/internal/interfaces/rpc/orderpb"

	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type Server struct {
	orderpb.UnimplementedOrderPayServiceServer
	repo domainorder.OrderRepository
}

func NewServer(repo domainorder.OrderRepository) *Server {
	return &Server{repo: repo}
}

func (s *Server) NotifyPaid(ctx context.Context, req *orderpb.NotifyPaidRequest) (*orderpb.NotifyPaidResponse, error) {

	if req.GetOrderNo() == "" || req.GetPayNo() == "" {
		return &orderpb.NotifyPaidResponse{}, fmt.Errorf("invalid request parameters")
	}
	o, err := s.repo.FindByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		return &orderpb.NotifyPaidResponse{}, fmt.Errorf("订单不存在: %w", err)
	}
	if o.TotalAmount != req.GetAmount() {
		return &orderpb.NotifyPaidResponse{}, fmt.Errorf("金额与订单不符")
	}
	changed, err := s.repo.MarkPaid(ctx, req.GetOrderNo())
	if err != nil {
		return &orderpb.NotifyPaidResponse{}, fmt.Errorf("订单状态更新失败: %w", err)
	}
	if !changed {
		return &orderpb.NotifyPaidResponse{}, fmt.Errorf("订单状态未改变")
	}
	// Implementation for handling paid notification
	return &orderpb.NotifyPaidResponse{}, nil
}

func (s *Server) GetPayableOrder(ctx context.Context, req *orderpb.GetPayableOrderRequest) (*orderpb.GetPayableOrderResponse, error) {
	o, err := s.repo.FindByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		return nil, fmt.Errorf("订单不存在: %w", err)
	}
	if o.CanPay() != true {
		return nil, fmt.Errorf("订单不可支付")
	}
	return &orderpb.GetPayableOrderResponse{
		OrderNo:     o.OrderNo,
		TotalAmount: o.TotalAmount,
		Status:      int64(domainpayment.StatusProcessing),
	}, nil
}
