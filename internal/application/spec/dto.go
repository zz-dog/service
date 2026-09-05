package specapp

import "time"

type SpecInput struct {
	Name   string           `json:"name"` // 规格名称
	Sort   int              `json:"sort"` // 排序
	Values []SpecValueInput `json:"values"`
}

type SpecValueInput struct {
	Name string `json:"name"` // 规格值名称
	Sort int    `json:"sort"` // 排序
}

// UpdateSpecInput 整包更新入参：Values 为该规格的完整值列表，
// 带ID的值会被更新、不带ID的会被新增、数据库中缺席的会被删除。
type UpdateSpecInput struct {
	Name   string                 `json:"name"` // 规格名称
	Sort   int                    `json:"sort"` // 排序
	Values []UpdateSpecValueInput `json:"values"`
}

type UpdateSpecValueInput struct {
	ValueID uint   `json:"specValueId"` // 规格值ID，0 表示新增
	Name    string `json:"name"`        // 规格值名称
	Sort    int    `json:"sort"`        // 排序
}

type SpecDTO struct {
	SpecID uint           `json:"specId"`
	Name   string         `json:"name"` // 规格名称
	Sort   int            `json:"sort"`
	Values []SpecValueDTO `json:"values"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SpecValueDTO struct {
	SpecValueID uint   `json:"specValueId"`
	Name        string `json:"name"`
	Sort        int    `json:"sort"`
	SpecID      uint   `json:"specId"`
}

type ListDto struct {
	List  []*SpecDTO `json:"list"`
	Total int        `json:"total"`
}
