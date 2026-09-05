package specpo

import (
	"context"
	"errors"

	domainSpec "github.com/wsc-zz/service/internal/domain/spec"
	"gorm.io/gorm"
)

type SpecRepository struct {
	db *gorm.DB
}

func NewSpecRepository(db *gorm.DB) *SpecRepository {
	return &SpecRepository{db: db}
}

func (r *SpecRepository) FindByID(ctx context.Context, id uint) (*domainSpec.Spec, error) {

	var po SpecPO
	err := r.db.WithContext(ctx).Where("spec_id = ?", id).First(&po).Error
	// 处理未找到记录的情况
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainSpec.ErrSpecNotFound
		}
		return nil, err
	}
	return toSpec(&po), nil
}

// FindByIDs 批量按ID查询规格，Preload 一次带出全部规格值（两条 SQL）。
func (r *SpecRepository) FindByIDs(ctx context.Context, ids []uint) ([]*domainSpec.Spec, error) {
	if len(ids) == 0 {
		return []*domainSpec.Spec{}, nil
	}
	var pos []SpecPO
	err := r.db.WithContext(ctx).
		Where("spec_id IN ?", ids).
		Preload("Values").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domainSpec.Spec, len(pos))
	for i := range pos {
		result[i] = toSpec(&pos[i])
	}
	return result, nil
}

func (r *SpecRepository) FindByName(ctx context.Context, name string) (*domainSpec.Spec, error) {

	var po SpecPO
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&po).Error
	// 处理未找到记录的情况
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainSpec.ErrSpecNotFound
		}
		return nil, err
	}
	return toSpec(&po), nil
}

// Save 保存规格聚合（含规格值），单事务内完成：
// 主记录插入/更新；规格值与库中对账——带主键的更新、新值插入并回填ID、缺席的删除。
// 不依赖 GORM 的关联级联保存，行为全部显式。
func (r *SpecRepository) Save(ctx context.Context, spec *domainSpec.Spec) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		po := toSpecPO(spec)

		// 主记录：插入或只更新名称与排序（不级联保存 Values）
		if po.SpecID == 0 {
			if err := tx.Omit("Values").Create(po).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&SpecPO{}).
				Where("spec_id = ?", po.SpecID).
				Updates(map[string]interface{}{"name": po.Name, "sort": po.Sort}).Error; err != nil {
				return err
			}
		}
		spec.SpecID = po.SpecID

		// 库中已有的规格值，用于对账
		var existing []SpecValuePO
		if err := tx.Where("spec_id = ?", po.SpecID).Find(&existing).Error; err != nil {
			return err
		}
		existingIDs := make(map[uint]bool, len(existing))
		for _, v := range existing {
			existingIDs[v.ValueID] = true
		}

		keepIDs := make(map[uint]bool, len(po.Values))
		for i := range po.Values {
			v := &po.Values[i]
			v.SpecID = po.SpecID
			if existingIDs[v.ValueID] {
				if err := tx.Save(v).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Create(v).Error; err != nil {
					return err
				}
			}
			keepIDs[v.ValueID] = true
			// 回填到领域对象，供上层返回
			spec.Values[i].ValueID = v.ValueID
			spec.Values[i].SpecID = po.SpecID
		}

		// 删除本次提交中缺席的值
		ids := make([]uint, 0, len(existing))
		for _, v := range existing {
			if !keepIDs[v.ValueID] {
				ids = append(ids, v.ValueID)
			}
		}
		if len(ids) > 0 {
			if err := tx.Where("value_id IN ?", ids).Delete(&SpecValuePO{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SpecRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("spec_id = ?", id).Delete(&SpecPO{}).Error
}

func (r *SpecRepository) SpecList(ctx context.Context, q domainSpec.SpecListQuery) ([]*domainSpec.Spec, int, error) {
	if q.Page < 1 {
		q.Page = 1

	}
	if q.PageSize < 1 {
		q.PageSize = 10
	}
	// 过滤条件收敛到闭包里，分别构造两条干净的查询链：
	// 复用执行过 finisher（Find/Count）的语句会继承脏子句，GORM 不保证安全。
	applyFilters := func(tx *gorm.DB) *gorm.DB {
		if q.Name != "" {
			tx = tx.Where("name LIKE ?", "%"+q.Name+"%")
		}
		if q.SpecID > 0 {
			tx = tx.Where("spec_id = ?", q.SpecID)
		}
		return tx
	}

	var total int64
	if err := applyFilters(r.db.WithContext(ctx).Model(&SpecPO{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Preload 用第二条查询加载规格值（非 JOIN），避免一对多 JOIN 的行膨胀
	var pos []SpecPO
	err := applyFilters(r.db.WithContext(ctx).Model(&SpecPO{})).
		Preload("Values").
		Order("spec_id").
		Limit(q.PageSize).
		Offset((q.Page - 1) * q.PageSize).
		Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}
	result := make([]*domainSpec.Spec, len(pos))
	for i := range pos {
		result[i] = toSpec(&pos[i])
	}
	return result, int(total), nil
}
func toSpecValues(vaules []SpecValuePO) []domainSpec.SpecValue {
	value := make([]domainSpec.SpecValue, 0, len(vaules))
	for _, v := range vaules {
		value = append(value, domainSpec.SpecValue{
			ValueID: v.ValueID,
			SpecID:  v.SpecID,
			Name:    v.Name,
			Sort:    v.Sort,
		})
	}
	return value
}

func toSpec(po *SpecPO) *domainSpec.Spec {

	return &domainSpec.Spec{
		SpecID:    po.SpecID,
		Name:      po.Name,
		Sort:      po.Sort,
		Values:    toSpecValues(po.Values),
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

func toSpecPO(spec *domainSpec.Spec) *SpecPO {

	return &SpecPO{
		SpecID: spec.SpecID,
		Name:   spec.Name,
		Sort:   spec.Sort,
		Values: toSpecValuePO(spec.Values),
	}
}

func toSpecValuePO(values []domainSpec.SpecValue) []SpecValuePO {

	value := make([]SpecValuePO, 0, len(values))
	for _, v := range values {
		value = append(value, SpecValuePO{
			ValueID: v.ValueID,
			SpecID:  v.SpecID,
			Name:    v.Name,
			Sort:    v.Sort,
		})
	}
	return value
}
