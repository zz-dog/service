package categorypo

import (
	"context"
	"errors"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
	"gorm.io/gorm"
)

// CategorySpecPO 分类-规格绑定表：联合主键 (category_id, spec_id)，
// sort/required 是关系本身的属性。
type CategorySpecPO struct {
	CategoryID uint `gorm:"primaryKey;comment:分类ID"`
	SpecID     uint `gorm:"primaryKey;comment:规格维度ID"`
	Sort       int  `gorm:"default:0;comment:规格在该分类下的排序"`
	Required   bool `gorm:"default:false;comment:商品发布时是否必选"`
}

func (CategorySpecPO) TableName() string {
	return "category_specs"
}

func (po *CategorySpecPO) toDomain() *domaincategory.CategorySpec {
	return &domaincategory.CategorySpec{
		CategoryID: po.CategoryID,
		SpecID:     po.SpecID,
		Sort:       po.Sort,
		Required:   po.Required,
	}
}

func toCategorySpecPO(cs *domaincategory.CategorySpec) *CategorySpecPO {
	return &CategorySpecPO{
		CategoryID: cs.CategoryID,
		SpecID:     cs.SpecID,
		Sort:       cs.Sort,
		Required:   cs.Required,
	}
}

type CategorySpecRepository struct {
	db *gorm.DB
}

func NewCategorySpecRepository(db *gorm.DB) *CategorySpecRepository {
	return &CategorySpecRepository{db: db}
}

func (r *CategorySpecRepository) Find(ctx context.Context, categoryID, specID uint) (*domaincategory.CategorySpec, error) {
	var po CategorySpecPO
	err := r.db.WithContext(ctx).
		Where("category_id = ? AND spec_id = ?", categoryID, specID).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincategory.ErrCategorySpecNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *CategorySpecRepository) FindByCategoryID(ctx context.Context, categoryID uint) ([]*domaincategory.CategorySpec, error) {
	var pos []*CategorySpecPO
	if err := r.db.WithContext(ctx).
		Where("category_id = ?", categoryID).
		Order("sort ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	bindings := make([]*domaincategory.CategorySpec, 0, len(pos))
	for _, po := range pos {
		bindings = append(bindings, po.toDomain())
	}
	return bindings, nil
}

func (r *CategorySpecRepository) FindBySpecID(ctx context.Context, specID uint) ([]*domaincategory.CategorySpec, error) {
	var pos []*CategorySpecPO
	if err := r.db.WithContext(ctx).
		Where("spec_id = ?", specID).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	bindings := make([]*domaincategory.CategorySpec, 0, len(pos))
	for _, po := range pos {
		bindings = append(bindings, po.toDomain())
	}
	return bindings, nil
}

func (r *CategorySpecRepository) Save(ctx context.Context, cs *domaincategory.CategorySpec) error {
	return r.db.WithContext(ctx).Save(toCategorySpecPO(cs)).Error
}

func (r *CategorySpecRepository) Delete(ctx context.Context, categoryID, specID uint) error {
	res := r.db.WithContext(ctx).
		Where("category_id = ? AND spec_id = ?", categoryID, specID).
		Delete(&CategorySpecPO{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domaincategory.ErrCategorySpecNotFound
	}
	return nil
}

func (r *CategorySpecRepository) DeleteByCategoryID(ctx context.Context, categoryID uint) error {
	return r.db.WithContext(ctx).
		Where("category_id = ?", categoryID).
		Delete(&CategorySpecPO{}).Error
}
