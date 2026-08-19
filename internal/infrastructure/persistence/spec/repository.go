package specpo

import (
	"context"

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
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	// 处理未找到记录的情况
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainSpec.ErrSpecNotFound
		}
	}
	return toSpec(&po), nil
}

func (r *SpecRepository) FindByName(ctx context.Context, name string) (*domainSpec.Spec, error) {

	var po SpecPO
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&po).Error
	// 处理未找到记录的情况
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainSpec.ErrSpecNotFound
		}
	}
	return toSpec(&po), nil
}

func (r *SpecRepository) Save(ctx context.Context, spec *domainSpec.Spec) error {
	return r.db.WithContext(ctx).Save(toSpecPO(spec)).Error
}

func (r *SpecRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&SpecPO{}).Error
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
