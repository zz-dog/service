package specapp

import "time"

type SpecInput struct {
	Name string `json:"name"` // 规格名称
	Sort int    `json:"sort"` // 排序
}

type SpecDTO struct {
	SpecID uint   `json:"specId"`
	Name   string `json:"name"` // 规格名称
	Sort   int    `json:"sort"`
	Values []SpecValueDTO

	CreatedAt time.Time
	UpdatedAt time.Time
}

type SpecValueDTO struct {
	SpecValueID uint   `json:"specValueId"`
	Name        string `json:"name"`
	Sort        int    `json:"sort"`
	SpecID      uint   `json:"specId"`
}
