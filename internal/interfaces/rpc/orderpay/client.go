package orderpay

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/wsc-zz/service/global"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	domainpayment "github.com/wsc-zz/service/internal/domain/payment"
	"github.com/wsc-zz/service/internal/interfaces/rpc/orderpb"
)

type Client struct {
	conn    *grpc.ClientConn
	stub    orderpb.OrderPayServiceClient
	timeout time.Duration
}

func NewClient(addr string, timeout time.Duration) (*Client, error) {
	// grpc.WithTransportCredentials(insecure.NewCredentials()) 允许不使用 TLS, 仅用于内部服务间通信
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, stub: orderpb.NewOrderPayServiceClient(conn), timeout: timeout}, nil
}

// mapStatus 把 gRPC 状态码翻译回 payment 领域错误——应用层只认业务语义不认协议:
//   NotFound           → ErrOrderNotFound
//   FailedPrecondition → ErrOrderCancelled(订单已取消,取消竞态的信号)
//   InvalidArgument    → ErrAmountMismatch(金额与订单不符)
//   其他               → 原样包装(基础设施问题,不许静默)
func mapStatus(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return domainpayment.ErrOrderNotFound
	case codes.FailedPrecondition:
		return domainpayment.ErrOrderCancelled
	case codes.InvalidArgument:
		return domainpayment.ErrAmountMismatch
	default:
		return err
	}
}

// isUnknown 结果未知:请求可能没到(没生效),也可能到了但响应丢了(已生效)。
// NotifyPaid 订单侧条件更新幂等,所以这两种情况重试都安全——重试把它变成前两种之一。
func isUnknown(err error) bool {
	code := status.Code(err)
	return code == codes.DeadlineExceeded || code == codes.Unavailable
}

// NotifyPaid 通知订单支付结果。幂等(订单侧条件更新保证),所以"结果未知"时
// 同参重发一次是安全的——这段注释 inventory 的 client 里也有一份,但那边实现
// 把重试删掉了;这边是真实现,对比着看。
func (c *Client) NotifyPaid(ctx context.Context, req paymentapp.NotifyPaidInput) error {
	reqPB := &orderpb.NotifyPaidRequest{
		OrderNo: req.OrderNo,
		PayNo:   req.PayNo,
		Amount:  req.Amount,
	}
	rc, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err := c.stub.NotifyPaid(rc, reqPB)
	if isUnknown(err) {
		global.Logger.Warn("通知订单结果未知,重试一次",
			zap.String("pay_no", req.PayNo), zap.String("order_no", req.OrderNo), zap.Error(err))
		rc2, cancel2 := context.WithTimeout(ctx, c.timeout)
		defer cancel2()
		_, err = c.stub.NotifyPaid(rc2, reqPB)
	}
	if err != nil {
		return mapStatus(err)
	}
	return nil
}

func (c *Client) GetPayableOrder(ctx context.Context, orderNo string) (paymentapp.PayableOrder, error) {
	resp, err := c.call(ctx, func(ctx context.Context) (any, error) {
		return c.stub.GetPayableOrder(ctx, &orderpb.GetPayableOrderRequest{
			OrderNo: orderNo,
		})
	})
	if err != nil {
		return paymentapp.PayableOrder{}, mapStatus(err)
	}
	return toPayableOrder(resp.(*orderpb.GetPayableOrderResponse)), nil
}

func (c *Client) call(ctx context.Context, do func(context.Context) (any, error)) (any, error) {
	rc, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := do(rc)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func toPayableOrder(pb *orderpb.GetPayableOrderResponse) paymentapp.PayableOrder {
	uid, _ := strconv.ParseUint(pb.GetUserId(), 10, 64) // 服务端写的一定是数字,解析失败按 0 处理(归属校验会拦)
	return paymentapp.PayableOrder{OrderNo: pb.OrderNo, UserID: uint(uid), Status: int(pb.Status), TotalAmount: pb.TotalAmount}
}
