package categoryapp

import "time"

type CreateCategoryInput struct {
	Name     string `json:"name"`
	ParentID uint   `json:"parent_id"`
	Sort     int    `json:"sort"`
}

type UpdateCategoryInput struct {
	Name     string `json:"name"`
	ParentID uint   `json:"parent_id"`
	Sort     int    `json:"sort"`
}

type CategoryDto struct {
	CategoryID uint      `json:"category_id"`
	Name       string    `json:"name"`
	ParentID   uint      `json:"parent_id"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
