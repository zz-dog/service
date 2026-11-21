package merchantpo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MerchantRepository 商户仓储。
type MerchantRepository struct {
	db *gorm.DB
}

func NewMerchantRepository(db *gorm.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

// Save 新建商户(审核通过时)。Create 前转 PO;Create 后把自增主键回填给领域对象——
// 调用方(reviewApprove)要拿 m.MerchantID 去做 BindMerchant,凭证与结果互相指认。
func (r MerchantRepository) Save(ctx context.Context, m *domainmerchant.Merchant) error {
	po := topoMerchant(m)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	m.MerchantID = po.ID
	return nil
}

func (r MerchantRepository) FindByID(ctx context.Context, id uint) (*domainmerchant.Merchant, error) {
	var po MerchantPO
	err := r.db.WithContext(ctx).First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainmerchant.ErrMerchantNotFound
		}
		return nil, err
	}
	return todomainMerchant(&po), nil
}

// ListByIDs 批量按 ID 查商户(我的店铺列表:成员记录翻译成店铺视图时用)。
// ids 为空直接返回空切片——GORM 的 Where("id IN ?", []uint{}) 会生成 1=2 恒假条件,
// 这里提前短路,语义也更直白。
func (r MerchantRepository) ListByIDs(ctx context.Context, ids []uint) ([]domainmerchant.Merchant, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var pos []MerchantPO
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&pos).Error
	if err != nil {
		return nil, err
	}
	merchants := make([]domainmerchant.Merchant, len(pos))
	for i := range pos {
		merchants[i] = *todomainMerchant(&pos[i])
	}
	return merchants, nil
}

// UpdateStatus 条件更新 from→to:停业/恢复/封禁共用。
// 迁移合法性由调用方先用领域方法裁决;这里只保证并发下只有一个赢家。
func (r MerchantRepository) UpdateStatus(ctx context.Context, id uint, from, to int) (bool, error) {
	result := r.db.WithContext(ctx).Model(&MerchantPO{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	return result.RowsAffected == 1, result.Error
}

// UpdateInfo 更新店铺资料。desc 是 MySQL 保留字,GORM 会加反引号,直接用列名即可。
func (r MerchantRepository) UpdateInfo(ctx context.Context, id uint, name, logo, desc string) error {
	return r.db.WithContext(ctx).Model(&MerchantPO{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"name": name,
			"logo": logo,
			"desc": desc,
		}).Error
}
