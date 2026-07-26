package categorypo

import (
	"context"
	"errors"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (c *CategoryRepository) FindByID(ctx context.Context, id uint) (*domaincategory.Category, error) {
	var po CategoryPO
	err := c.db.WithContext(ctx).Where("category_id = ?", id).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincategory.ErrCategoryNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (c *CategoryRepository) FindByName(ctx context.Context, name string) (*domaincategory.Category, error) {
	var po CategoryPO
	err := c.db.WithContext(ctx).Where("name = ?", name).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincategory.ErrCategoryNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}
func (c *CategoryRepository) FindAll(ctx context.Context) ([]*domaincategory.Category, error) {
	var pos []*CategoryPO
	if err := c.db.WithContext(ctx).Find(&pos).Error; err != nil {
		return nil, err
	}
	var categories = make([]*domaincategory.Category, 0, len(pos))
	for _, po := range pos {
		categories = append(categories, po.toDomain())
	}
	return categories, nil
}

func (c *CategoryRepository) Save(ctx context.Context, cate *domaincategory.Category) error {
	var po = toPO(cate)
	return c.db.WithContext(ctx).Save(po).Error
}

func (c *CategoryRepository) Delete(ctx context.Context, id uint) error {
	return c.db.WithContext(ctx).Where("category_id = ?", id).Delete(&CategoryPO{}).Error
}
