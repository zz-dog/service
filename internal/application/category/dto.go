package categoryapp

import "time"

type CreateCategoryInput struct {
	Name     string `json:"name"`
	ParentID uint   `json:"parent_id"`
	Sort     int    `json:"sort"`
}

type UpdateCategoryInput struct {
	CategoryID uint    `json:"category_id"`
	Name       *string `json:"name"`      // 可选，nil 表示不修改
	ParentID   *uint   `json:"parent_id"` // 可选，nil 表示不修改
	Sort       *int    `json:"sort"`      // 可选，nil 表示不修改
}
type DeleteCategoryInput struct {
	CategoryID uint `json:"category_id"`
}
type CategoryDto struct {
	CategoryID uint      `json:"category_id"`
	Name       string    `json:"name"`
	ParentID   uint      `json:"parent_id"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BindSpecInput 绑定规格到分类
type BindSpecInput struct {
	CategoryID uint `json:"category_id"`
	SpecID     uint `json:"spec_id"`
	Sort       int  `json:"sort"`
	Required   bool `json:"required"`
}

// UnbindSpecInput 解除绑定
type UnbindSpecInput struct {
	CategoryID uint `json:"category_id"`
	SpecID     uint `json:"spec_id"`
}

// CategorySpecDto 分类下规格视图：绑定属性 + 规格维度本身的信息
type CategorySpecDto struct {
	CategoryID uint           `json:"category_id"`
	SpecID     uint           `json:"spec_id"`
	SpecName   string         `json:"spec_name"`
	Sort       int            `json:"sort"`
	Required   bool           `json:"required"`
	Values     []SpecValueDto `json:"values"`
}

// SpecValueDto 规格值视图（应用层局部定义，避免依赖 spec 应用层）
type SpecValueDto struct {
	ValueID uint   `json:"value_id"`
	Name    string `json:"name"`
	Sort    int    `json:"sort"`
}
