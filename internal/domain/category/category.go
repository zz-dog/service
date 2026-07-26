package category

import "time"

type Category struct {
	CategoryID uint
	Name       string
	ParentID   uint // 父分类ID，0 表示顶级分类
	Sort       int  // 排序权重，越小越靠前

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCategory 创建分类
func NewCategory(name string, parentID uint, sort int) (*Category, error) {
	if name == "" {
		return nil, ErrEmptyCategoryName
	}
	return &Category{
		Name:     name,
		ParentID: parentID,
		Sort:     sort,
	}, nil
}

// Rename 重命名分类
func (c *Category) Rename(name string) error {
	if name == "" {
		return ErrEmptyCategoryName
	}
	c.Name = name
	return nil
}

// SetParentID 设置父分类ID（0 表示顶级分类）
func (c *Category) SetParentID(parentID uint) {
	c.ParentID = parentID
}

// SetSort 设置排序权重，越小越靠前
func (c *Category) SetSort(sort int) {
	c.Sort = sort
}
