package merchantapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MyShop 我的店铺。一人一店:FindByUserID 查出来的店就是你的,不需要再比 userID。
func (s *Service) MyShop(ctx context.Context, userID uint) (*MerchantDTO, error) {
	m, err := s.ownedShop(ctx, userID)
	if err != nil {
		return nil, err
	}
	dto := toMerchantDTO(m)
	return &dto, nil
}

// UpdateShop 更新店铺资料:领域方法校验规则(名字非空),仓储按 ID 精确更新列。
func (s *Service) UpdateShop(ctx context.Context, userID uint, in UpdateShopInput) (*MerchantDTO, error) {
	m, err := s.ownedShop(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := m.UpdateInfo(in.Name, in.Logo, in.Desc); err != nil {
		return nil, err
	}
	if err := s.mRepo.UpdateInfo(ctx, m.MerchantID, in.Name, in.Logo, in.Desc); err != nil {
		return nil, err
	}
	dto := toMerchantDTO(m)
	return &dto, nil
}

// Suspend 停业:领域方法裁决"营业中才能停",仓储条件更新兜并发。
func (s *Service) Suspend(ctx context.Context, userID uint) error {
	m, err := s.ownedShop(ctx, userID)
	if err != nil {
		return err
	}
	return s.applyTransition(ctx, m, m.Suspend)
}

// Resume 恢复营业:已停业才能恢复(封禁的店平台说了算,店主 Resume 不动它)。
func (s *Service) Resume(ctx context.Context, userID uint) error {
	m, err := s.ownedShop(ctx, userID)
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

// ownedShop 当前用户的店铺;没有 → ErrMerchantNotFound。
func (s *Service) ownedShop(ctx context.Context, userID uint) (*domainmerchant.Merchant, error) {
	m, err := s.mRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, domainmerchant.ErrMerchantNotFound
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
