package merchantapp

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// InviteMember 邀请成员(仅店主):直接指定目标 userID 与角色(admin/staff)。
// 并发防线是 (user_id, merchant_id) 唯一索引——两个入口同时邀请同一人,
// 友好检查全过,落库只有一个成功,另一个拿到 ErrAlreadyMember。
func (s *Service) InviteMember(ctx context.Context, userID, merchantID uint, in InviteMemberInput) (*MemberDTO, error) {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageStaff); err != nil {
		return nil, err
	}
	count, err := s.memberRepo.CountByMerchant(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if count >= domainmerchant.MaxMembersPerShop {
		return nil, domainmerchant.ErrMemberLimitReached
	}
	existing, err := s.memberRepo.FindByUserAndMerchant(ctx, in.TargetUserID, merchantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domainmerchant.ErrAlreadyMember
	}
	m, err := domainmerchant.NewMember(in.TargetUserID, merchantID, domainmerchant.MemberRole(in.Role))
	if err != nil {
		return nil, err
	}
	if err := s.memberRepo.Save(ctx, m); err != nil {
		// 并发撞唯一索引:别人先邀进来了(与 global/gorm.go 的 TranslateError 约定一致)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, domainmerchant.ErrAlreadyMember
		}
		return nil, err
	}
	dto := toMemberDTO(m)
	return &dto, nil
}

// RemoveMember 移除成员(仅店主)。
// 店主不可被移除(含店主移除自己):店必须永远有主,
// "owner 转让"(降自己升别人)要两步跨行事务,是进阶,教学版先封死这个口子。
func (s *Service) RemoveMember(ctx context.Context, userID, merchantID, targetUserID uint) error {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageStaff); err != nil {
		return err
	}
	if targetUserID == userID {
		return domainmerchant.ErrCannotModifyOwner
	}
	target, err := s.memberRepo.FindByUserAndMerchant(ctx, targetUserID, merchantID)
	if err != nil {
		return err
	}
	if target == nil {
		return domainmerchant.ErrMemberNotFound
	}
	if target.Role == domainmerchant.RoleOwner {
		return domainmerchant.ErrCannotModifyOwner
	}
	changed, err := s.memberRepo.Delete(ctx, merchantID, targetUserID)
	if err != nil {
		return err
	}
	if !changed {
		return domainmerchant.ErrMemberNotFound // 刚被人移过,条件删除没抢到
	}
	return nil
}

// ChangeMemberRole 调整成员角色(仅店主):admin↔staff 互换。
// owner 不可被调整(含自己);调整成 owner 直接在参数校验层拒掉——
// 授予 owner 的唯一入口是审核通过事务(NewOwnerMember),不是这里。
func (s *Service) ChangeMemberRole(ctx context.Context, userID, merchantID, targetUserID uint, in ChangeMemberRoleInput) error {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageStaff); err != nil {
		return err
	}
	role := domainmerchant.MemberRole(in.Role)
	if role != domainmerchant.RoleAdmin && role != domainmerchant.RoleStaff {
		return domainmerchant.ErrInvalidMemberRole
	}
	if targetUserID == userID {
		return domainmerchant.ErrCannotModifyOwner
	}
	target, err := s.memberRepo.FindByUserAndMerchant(ctx, targetUserID, merchantID)
	if err != nil {
		return err
	}
	if target == nil {
		return domainmerchant.ErrMemberNotFound
	}
	if target.Role == domainmerchant.RoleOwner {
		return domainmerchant.ErrCannotModifyOwner
	}
	changed, err := s.memberRepo.UpdateRole(ctx, merchantID, targetUserID, role)
	if err != nil {
		return err
	}
	if !changed {
		return domainmerchant.ErrMemberNotFound // 刚被人移过,条件更新没抢到
	}
	return nil
}

// ListMembers 成员列表(仅店主)。
// 教学版人事权只给 owner:看成员名单是邀请/移除的前置,跟着人事权走。
func (s *Service) ListMembers(ctx context.Context, userID, merchantID uint) ([]MemberDTO, error) {
	if _, err := s.memberAt(ctx, userID, merchantID, domainmerchant.MemberRole.CanManageStaff); err != nil {
		return nil, err
	}
	members, err := s.memberRepo.ListByMerchant(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	dtos := make([]MemberDTO, 0, len(members))
	for i := range members {
		dtos = append(dtos, toMemberDTO(&members[i]))
	}
	return dtos, nil
}
