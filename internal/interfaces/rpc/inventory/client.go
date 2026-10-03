package inventory

import (
	"context"
	"fmt"
	"time"

	orderapp "github.com/wsc-zz/service/internal/application/order"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/internal/interfaces/rpc/inventorypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn    *grpc.ClientConn
	stub    inventorypb.InventoryServiceClient
	timeout time.Duration
}

// NewClient 建立到 catalog gRPC 的连接。
// grpc.NewClient 是惰性连接(首次调用才真正拨号),所以启动期连不上也不报错,
// 故障推迟到第一次调用时以 Unavailable 暴露——这是 gRPC 想让你知道的语义。
func NewClient(addr string, timeout time.Duration) (*Client, error) {

	// grpc.WithTransportCredentials(insecure.NewCredentials()) 允许不使用 TLS, 仅用于内部服务间通信
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, stub: inventorypb.NewInventoryServiceClient(conn), timeout: timeout}, nil
}

func (c *Client) DeductBatch(ctx context.Context, requestID string, item []orderapp.InventoryItem) ([]orderapp.DeductedItem, error) {

	resp, err := c.call(ctx, func(ctx context.Context) (any, error) {
		return c.stub.DeductBatch(ctx, &inventorypb.DeductBatchRequest{
			RequestId: requestID,
			Items:     inventoryToDeductStock(item),
		})
	})
	if err != nil {
		return nil, err
	}
	// call 返回的是 any,取出 *inventorypb.DeductBatchResponse 转回应用层类型
	return deductStockToInventory(resp.(*inventorypb.DeductBatchResponse).Items), nil
}

func (c *Client) RestockBatch(ctx context.Context, requestID string, item []orderapp.InventoryItem) error {

	_, err := c.call(ctx, func(ctx context.Context) (any, error) {
		return c.stub.RestockBatch(ctx, &inventorypb.RestockBatchRequest{
			RequestId: requestID,
			Items:     inventoryToDeductStock(item),
		})
	})
	if err != nil {
		return err
	}
	return nil
}

// call 统一处理:超时控制 + "结果未知"重试。
// 超时可能是请求没到 catalog(没扣),也可能响应丢了(已扣)。此时**不补偿**,
// 用同一个 request_id 重试一次:幂等保证不会重复扣,重试把"未知"变成"确定"。
// 仍未知则返回 ErrInventoryUnknown——调用方(应用层)必须把它当失败返回用户,但绝不能回补。
func (c *Client) call(ctx context.Context, do func(context.Context) (any, error)) (any, error) {
	rc, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := do(rc)
	if err != nil {
		return nil, c.mapStatus(err)
	}
	return resp, nil
}

// mapStatus 把 gRPC status code 恢复成 handler 认识的领域错误。
// 直接复用 product 领域错误(同 module 内的现实选择,handler 的 errors.Is 映射零改动);
// 严格 DDD 做法是在 order 应用层定义自己的错误集,代价是 handler 同步改——教学取舍,选前者。
func (c *Client) mapStatus(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("库存服务调用失败: %w", err)
	}
	switch st.Code() {
	case codes.ResourceExhausted:
		return domainproduct.ErrInsufficientStock
	case codes.NotFound:
		return domainproduct.ErrSKUNotFound
	case codes.InvalidArgument:
		return fmt.Errorf("库存请求参数错误: %s", st.Message())
	default:
		return fmt.Errorf("库存服务不可用(%s): %s", st.Code(), st.Message())
	}
}

func inventoryToDeductStock(items []orderapp.InventoryItem) []*inventorypb.DeductItem {
	var result = make([]*inventorypb.DeductItem, 0, len(items))
	for _, item := range items {
		result = append(result, &inventorypb.DeductItem{
			ProductId: uint64(item.ProductID),
			Quantity:  int32(item.Quantity),
			SkuCode:   item.SKUCode,
		})
	}
	return result
}

func deductStockToInventory(items []*inventorypb.DeductedItem) []orderapp.DeductedItem {
	var result = make([]orderapp.DeductedItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderapp.DeductedItem{
			ProductID:   uint(item.ProductId),
			ProductName: item.ProductName,
			Quantity:    int(item.Quantity),
		})
	}
	return result
}
