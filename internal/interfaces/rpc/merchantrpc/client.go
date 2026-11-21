package merchantrpc

import (
	"context"
	"fmt"
	"time"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	productapp "github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn    *grpc.ClientConn
	stub    merchantpb.MerchantServiceClient
	timeout time.Duration
}

// NewClient 建立到 merchant gRPC 的连接。
// grpc.NewClient 是惰性连接(首次调用才真正拨号),启动期连不上也不报错,
// 故障推迟到第一次调用时以 Unavailable 暴露(与 inventory client 同款)。
func NewClient(addr string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, stub: merchantpb.NewMerchantServiceClient(conn), timeout: timeout}, nil
}

// AuthorizeMember 实现 productapp.MerchantGateway 端口:协议翻译,不做策略。
// merchant_id=0 / role=0 原样返回,"能不能做"的裁决在应用层——client 层不做业务裁决。
func (c *Client) AuthorizeMember(ctx context.Context, userID, merchantID uint) (productapp.MemberInfo, error) {
	rc, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.stub.AuthorizeMember(rc, &merchantpb.AuthorizeMemberRequest{
		UserId:     uint64(userID),
		MerchantId: uint64(merchantID),
	})
	if err != nil {
		return productapp.MemberInfo{}, c.mapStatus(err)
	}
	return productapp.MemberInfo{
		MerchantID: uint(resp.GetMerchantId()),
		Name:       resp.GetName(),
		Status:     int(resp.GetStatus()),
		Role:       domainmerchant.MemberRole(resp.GetRole()),
	}, nil
}

// mapStatus gRPC 错误统一翻译。这里没有可映射的领域错误:
// 发布是低频写操作,merchant 服务不可用就拒绝发布,绝不降级成"无主商品"——
// 归属语义和资金语义一样,不能在故障时含糊。
func (c *Client) mapStatus(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("商户服务调用失败: %w", err)
	}
	if st.Code() == codes.Unavailable {
		return fmt.Errorf("商户服务不可用,请稍后重试")
	}
	return fmt.Errorf("商户服务调用失败(%s): %s", st.Code(), st.Message())
}
