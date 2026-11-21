package merchantapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MyShops 我的店铺列表:N:N 下一个用户可归属多家店,每家带"我在它是谁"(角色)。
// 归属即过滤条件:先查成员记录,再批量取店铺信息——没有成员关系的店绝不会出现。
func (s *Service) MyShops(ctx context.Context, userID uint) ([]ShopDTO, error) {
	members, err := s.memberRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(members))
	for i := range members {
		ids = append(ids, members[i].MerchantID)
	}
	merchants, err := s.mRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]*domainmerchant.Merchant, len(merchants))
	for i := range merchants {
		byID[merchants[i].MerchantID] = &merchants[i]
	}
	dtos := make([]ShopDTO, 0, len(members))
	for i := range members {
		m, ok := byID[members[i].MerchantID]
		if !ok {
			continue // 防御:成员记录理应随店存亡,店没了就跳过,不让列表 500
		}
		dtos = append(dtos, toShopDTO(m, members[i].Role))
	}
	return dtos, nil
}

// UpdateShop 更新店铺资料:先过成员身份(非成员与店不存在同错误),
// 角色权限裁决(owner),领域方法校验规则,仓储按 ID 精确更新列。
func (s *Service) UpdateShop(ctx context.Context, userID, merchantID uint, in UpdateShopInput) (*ShopDTO, error) {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageShop); err != nil {
		return nil, err
	}
	m, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if err := m.UpdateInfo(in.Name, in.Logo, in.Desc); err != nil {
		return nil, err
	}
	if err := s.mRepo.UpdateInfo(ctx, m.MerchantID, in.Name, in.Logo, in.Desc); err != nil {
		return nil, err
	}
	dto := toShopDTO(m, domainmerchant.RoleOwner)
	return &dto, nil
}

// Suspend 停业:领域方法裁决"营业中才能停",仓储条件更新兜并发。
func (s *Service) Suspend(ctx context.Context, userID, merchantID uint) error {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageShop); err != nil {
		return err
	}
	m, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		return err
	}
	return s.applyTransition(ctx, m, m.Suspend)
}

// Resume 恢复营业:已停业才能恢复(封禁的店平台说了算,店主 Resume 不动它)。
func (s *Service) Resume(ctx context.Context, userID, merchantID uint) error {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageShop); err != nil {
		return err
	}
	m, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		return err
	}
	return s.applyTransition(ctx, m, m.Resume)
}

// Ban 封禁商户(管理端):同 Suspend/Resume 的双层结构,只是取数换成按 ID(管理员语义)。
func (s *Service) Ban(ctx context.Context, merchantID uint) error {
	m, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		return err
	}
	return s.applyTransition(ctx, m, m.Ban)
}

// memberAt 当前用户在该店的成员记录,并按 need 裁决角色权限:
// ① 非成员(无成员记录)与店不存在同错误 ErrMerchantNotFound——不泄露"这家店存在";
// ② 是成员但角色不够 → ErrPermissionDenied——他本来就是这家店的员工,可以明确告知。
// need 传领域权限方法(MemberRole.CanManageShop 等),"什么角色能做什么"不在这里展开。
func (s *Service) memberAt(ctx context.Context, userID, merchantID uint,
	need func(domainmerchant.MemberRole) bool) (*domainmerchant.Member, error) {
	if userID == 0 || merchantID == 0 {
		return nil, domainmerchant.ErrMerchantNotFound
	}
	m, err := s.memberRepo.FindByUserAndMerchant(ctx, userID, merchantID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, domainmerchant.ErrMerchantNotFound
	}
	if !need(m.Role) {
		return nil, domainmerchant.ErrPermissionDenied
	}
	return m, nil
}

// applyTransition 店铺状态迁移的双层落库:领域方法 migrate 裁决"这个迁移合法吗",
// 条件更新(WHERE status=迁移前状态)裁决"并发下是不是我先改的"。
func (s *Service) applyTransition(ctx context.Context, m *domainmerchant.Merchant, migrate func() error) error {
	from := int(m.Status)
	if err := migrate(); err != nil {
		return err
	}
	changed, err := s.mRepo.UpdateStatus(ctx, m.MerchantID, from, int(m.Status))
	if err != nil {
		return err
	}
	if !changed {
		return domainmerchant.ErrInvalidStatusTransition // 状态刚被并发改掉(如刚被封禁)
	}
	return nil
}
