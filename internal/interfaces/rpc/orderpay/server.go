package orderpay

import (
	"context"
	"fmt"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
	"github.com/wsc-zz/service/internal/interfaces/rpc/orderpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	orderpb.UnimplementedOrderPayServiceServer
	repo domainorder.OrderRepository
}

func NewServer(repo domainorder.OrderRepository) *Server {
	return &Server{repo: repo}
}

// NotifyPaid 通知支付结果。状态码即契约(proto 里的注释):
//   已支付重放       → OK(幂等:重复通知安全,渠道/重试器都靠它兜底)
//   订单已取消       → FailedPrecondition(payment 侧转"待退款",取消竞态的信号)
//   金额与订单不符   → InvalidArgument(金额防线:绝不自动放行,人工介入)
//   订单不存在       → NotFound
func (s *Server) NotifyPaid(ctx context.Context, req *orderpb.NotifyPaidRequest) (*orderpb.NotifyPaidResponse, error) {
	if req.GetOrderNo() == "" || req.GetPayNo() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no/pay_no 不能为空")
	}
	o, err := s.repo.FindByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		return nil, status.Error(codes.NotFound, "订单不存在")
	}
	if o.TotalAmount != req.GetAmount() {
		return nil, status.Error(codes.InvalidArgument, "金额与订单不符")
	}
	switch o.Status {
	case domainorder.StatusPaid:
		// 幂等重放:订单已支付,再通知一次也返回成功——重试器/渠道重发都靠这一句闭嘴
		return &orderpb.NotifyPaidResponse{}, nil
	case domainorder.StatusPending:
		// 继续:正常待支付订单
	default:
		// 已取消/已发货/已完成:状态机拒绝"支付"这个动作。
		// 已取消是取消竞态的信号(FailedPrecondition),其余状态照同一码返回——
		// payment 侧只区分"能记"和"不能记",细分留给日志
		return nil, status.Error(codes.FailedPrecondition, "订单已取消或状态不可支付")
	}
	// 条件更新(待支付→已支付)抢并发:取消 job 和我们同时到达,只有一人生效
	changed, err := s.repo.MarkPaid(ctx, req.GetOrderNo())
	if err != nil {
		return nil, status.Error(codes.Internal, "订单状态更新失败")
	}
	if !changed {
		// 读的时候还是待支付,更新时已被取消抢先——竞态败北,信号必须传回去
		return nil, status.Error(codes.FailedPrecondition, "订单已取消")
	}
	return &orderpb.NotifyPaidResponse{}, nil
}

// GetPayableOrder 查询订单(预支付前调用)。只做取数,归属与可付策略由 payment 裁决。
func (s *Server) GetPayableOrder(ctx context.Context, req *orderpb.GetPayableOrderRequest) (*orderpb.GetPayableOrderResponse, error) {
	o, err := s.repo.FindByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		return nil, status.Error(codes.NotFound, "订单不存在")
	}
	if !o.CanPay() {
		return nil, status.Error(codes.FailedPrecondition, "订单当前状态不可支付")
	}
	// 返回订单自己的状态,不是 payment 域的常量——跨域契约传的是订单状态机的事实
	return &orderpb.GetPayableOrderResponse{
		OrderNo:     o.OrderNo,
		UserId:      fmt.Sprintf("%d", o.UserID),
		TotalAmount: o.TotalAmount,
		Status:      int64(o.Status),
	}, nil
}
