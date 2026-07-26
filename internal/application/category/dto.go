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
