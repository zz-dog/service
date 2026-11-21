package merchantpo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MemberRepository 成员仓储:N:N 归属的事实源。
type MemberRepository struct {
	db *gorm.DB
}

func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Save 新增成员(开店时的 owner / 邀请)。Create 前转 PO;Create 后回填自增主键。
// 撞 (user_id, merchant_id) 唯一索引时,TranslateError 会把 MySQL 1062 翻译成
// gorm.ErrDuplicatedKey,调用方 errors.Is 判重后给出业务错误(与 global/gorm.go 约定一致)。
func (r MemberRepository) Save(ctx context.Context, m *domainmerchant.Member) error {
	po := topoMember(m)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	m.ID = po.ID
	return nil
}

// FindByUserAndMerchant 查成员关系;没有返回 (nil, nil)——非成员是正常业务态。
func (r MemberRepository) FindByUserAndMerchant(ctx context.Context, userID, merchantID uint) (*domainmerchant.Member, error) {
	var po MemberPO
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND merchant_id = ?", userID, merchantID).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return todomainMember(&po), nil
}

// ListByMerchant 某店铺的成员列表:店主在前,同级按加入时间正序——
// 人事列表天然按"老板→管理员→员工"排,谁在店里的时间最长一眼可见。
func (r MemberRepository) ListByMerchant(ctx context.Context, merchantID uint) ([]domainmerchant.Member, error) {
	var pos []MemberPO
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("role ASC, created_at ASC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	return todomainMembers(pos), nil
}

// ListByUser 某用户加入的所有店铺("我的店铺"列表),时间正序。
func (r MemberRepository) ListByUser(ctx context.Context, userID uint) ([]domainmerchant.Member, error) {
	var pos []MemberPO
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	return todomainMembers(pos), nil
}

// CountByMerchant 某店铺的成员数(邀请上限检查)。
func (r MemberRepository) CountByMerchant(ctx context.Context, merchantID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&MemberPO{}).
		Where("merchant_id = ?", merchantID).
		Count(&count).Error
	return count, err
}

// Delete 移除成员。返回是否有行被删:目标本来就不是成员 = false,
// 调用方决定报什么错(通常是 ErrMemberNotFound)。
func (r MemberRepository) Delete(ctx context.Context, merchantID, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("merchant_id = ? AND user_id = ?", merchantID, userID).
		Delete(&MemberPO{})
	return result.RowsAffected == 1, result.Error
}

// UpdateRole 调整成员角色(admin↔staff)。返回是否生效:目标不是成员 = false。
func (r MemberRepository) UpdateRole(ctx context.Context, merchantID, userID uint, role domainmerchant.MemberRole) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&MemberPO{}).
		Where("merchant_id = ? AND user_id = ?", merchantID, userID).
		Update("role", role)
	return result.RowsAffected == 1, result.Error
}
