package merchantrpc

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	merchantpo "github.com/wsc-zz/service/internal/infrastructure/persistence/merchant"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantpb"
	"gorm.io/gorm"
)

type Server struct {
	merchantpb.UnimplementedMerchantServiceServer
	mRepo domainmerchant.MerchantRepository
}

// NewServer merchant gRPC 服务端:内部自建仓储,构造只收 db,与 inventory server 同款。
func NewServer(db *gorm.DB) *Server {
	return &Server{mRepo: merchantpo.NewMerchantRepository(db)}
}

// GetMerchantByUserID 按归属用户查商户。
// "没有商户"是正常业务态,不是错误:返回 OK + merchant_id=0,**不要**返回 NotFound——
// catalog 拿 0 去拒绝发布即可;把业务态当错误码传,调用方就没法区分"没店"和"服务坏了"。
// 只做取数不做策略:"能不能发布"由 catalog 裁决(和 GetPayableOrder 同一纪律)。
func (s *Server) GetMerchantByUserID(ctx context.Context, req *merchantpb.GetMerchantByUserIDRequest) (*merchantpb.GetMerchantByUserIDResponse, error) {
	m, err := s.mRepo.FindByUserID(ctx, uint(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if m == nil {
		return &merchantpb.GetMerchantByUserIDResponse{}, nil
	}
	return &merchantpb.GetMerchantByUserIDResponse{
		MerchantId: uint64(m.MerchantID),
		Name:       m.Name,
		Status:     int32(m.Status),
	}, nil
}
