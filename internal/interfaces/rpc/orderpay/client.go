package orderpay

import (
	"context"
	"time"

	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	"github.com/wsc-zz/service/internal/interfaces/rpc/orderpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func (c *Client) NotifyPaid(ctx context.Context, req paymentapp.NotifyPaidInput) error {
	_, err := c.call(ctx, func(ctx context.Context) (any, error) {
		return c.stub.NotifyPaid(ctx, &orderpb.NotifyPaidRequest{
			OrderNo: req.OrderNo,
			PayNo:   req.PayNo,
			Amount:  req.Amount,
		})
	})
	if err != nil {
		return err
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
		return paymentapp.PayableOrder{}, err
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
	return paymentapp.PayableOrder{OrderNo: pb.OrderNo, Status: int(pb.Status), TotalAmount: pb.TotalAmount}
}
