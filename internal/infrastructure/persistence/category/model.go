package categorypo

import (
	"time"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
)

type CategoryPO struct {
	CategoryID uint   `gorm:"primaryKey;autoIncrement;comment:分类ID"`
	Name       string `gorm:"size:32;uniqueIndex;comment:分类名称"`
	ParentID   uint   `gorm:"index;default:0;comment:父级分类ID"`
	Sort       int    `gorm:"default:0;comment:排序"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CategoryPO) TableName(c *domaincategory.Category) string {
	return "categories"
}

func (po *CategoryPO) toDomain() *domaincategory.Category {
	return &domaincategory.Category{
		CategoryID: po.CategoryID,
		Name:       po.Name,
		ParentID:   po.ParentID,
		Sort:       po.Sort,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func toPO(c *domaincategory.Category) *CategoryPO {
	return &CategoryPO{
		CategoryID: c.CategoryID,
		Name:       c.Name,
		ParentID:   c.ParentID,
		Sort:       c.Sort,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
