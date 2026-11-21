package merchantrpc

import (
	"context"
	"errors"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	merchantpo "github.com/wsc-zz/service/internal/infrastructure/persistence/merchant"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantpb"
	"gorm.io/gorm"
)

type Server struct {
	merchantpb.UnimplementedMerchantServiceServer
	mRepo      domainmerchant.MerchantRepository
	memberRepo domainmerchant.MemberRepository
}

// NewServer merchant gRPC 服务端:内部自建仓储,构造只收 db,与 inventory server 同款。
func NewServer(db *gorm.DB) *Server {
	return &Server{
		mRepo:      merchantpo.NewMerchantRepository(db),
		memberRepo: merchantpo.NewMemberRepository(db),
	}
}

// AuthorizeMember 校验 (userID, merchantID) 的成员关系并带回角色。
// "拒绝"统一口径:店不存在与非成员同返回(merchant_id=0, role=0),OK 不报错——
// 把业务态当错误码传,调用方就没法区分"没权限"和"服务坏了";
// 同时不给探测者"这家店存在"的信号(非成员看到的和店不存在一模一样)。
// 只做取数不做策略:"拿到角色后能做什么"由调用方裁决(和 GetPayableOrder 同一纪律)。
func (s *Server) AuthorizeMember(ctx context.Context, req *merchantpb.AuthorizeMemberRequest) (*merchantpb.AuthorizeMemberResponse, error) {
	empty := &merchantpb.AuthorizeMemberResponse{}
	userID, merchantID := uint(req.GetUserId()), uint(req.GetMerchantId())
	if userID == 0 || merchantID == 0 {
		return empty, nil
	}
	m, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		if errors.Is(err, domainmerchant.ErrMerchantNotFound) {
			return empty, nil
		}
		return nil, err
	}
	mb, err := s.memberRepo.FindByUserAndMerchant(ctx, userID, merchantID)
	if err != nil {
		return nil, err
	}
	if mb == nil {
		return empty, nil
	}
	return &merchantpb.AuthorizeMemberResponse{
		MerchantId: uint64(m.MerchantID),
		Name:       m.Name,
		Status:     int32(m.Status),
		Role:       int32(mb.Role),
	}, nil
}
