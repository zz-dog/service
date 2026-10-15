package addresspo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/wsc-zz/service/internal/domain/address"
)

// AddressRepository 是 domain/address.AddressRepository 的 GORM 实现。
type AddressRepository struct {
	db *gorm.DB
}

// NewAddressRepository 构造仓储实现，注入 *gorm.DB（可以是事务内的 tx）。
func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Save(ctx context.Context, a *address.Address) error {
	po := toPO(a)
	// 主键为 0 走新增，否则更新全字段，与 userpo.UserRepository.Save 行为一致
	if a.AddressID == 0 {
		if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		// 回填自增主键与时间戳到领域实体，使调用方拿到持久化后的状态
		a.AddressID = po.AddressID
		a.CreatedAt = po.CreatedAt
		a.UpdatedAt = po.UpdatedAt
		return nil
	}
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *AddressRepository) FindByID(ctx context.Context, id uint) (*address.Address, error) {
	var po AddressPO
	err := r.db.WithContext(ctx).First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, address.ErrAddressNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *AddressRepository) FindByUserID(ctx context.Context, userID uint) ([]*address.Address, error) {
	var pos []AddressPO
	// 默认地址排最前，其余按最近更新排序
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, updated_at DESC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	addrs := make([]*address.Address, len(pos))
	for i, po := range pos {
		addrs[i] = toDomain(&po)
	}
	return addrs, nil
}

func (r *AddressRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&AddressPO{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}

func (r *AddressRepository) ClearDefault(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&AddressPO{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error
}

func (r *AddressRepository) Delete(ctx context.Context, a *address.Address) error {
	// gorm.DeletedAt 软删除：只更新 deleted_at，不物理删行
	po := toPO(a)
	return r.db.WithContext(ctx).Delete(po).Error
}
