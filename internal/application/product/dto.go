package productapp

import (
	"time"
)

// SpecItemInput SKU 规格项输入：只收 ID 引用。
// 名称快照由应用层从规格库取权威值填充，客户端传名称也不采信（防脏数据/注入）。
type SpecItemInput struct {
	SpecID  uint `json:"specId"`  // 规格维度ID
	ValueID uint `json:"valueId"` // 规格值ID
}

type SKUInput struct {
	// SKUCode 不收客户端值，由服务端从规格组合派生（值ID按维度ID排序拼接）
	SpecItems []SpecItemInput `json:"specItems"` // 规格组合
	Price     int64           `json:"price"`     // 价格
	Stock     int             `json:"stock"`     // 库存
}
type CreateProductInput struct {
	CategoryID uint       `json:"categoryId"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	SKUs       []SKUInput `json:"skus"`
}
type UpdateProductInput struct {
	ProductID  uint     `json:"productId"`
	CategoryID uint     `json:"categoryId"`
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Urls       []string `json:"urls"`
}
type DeductStockInput struct {
	SKUCode string `json:"skuCode"`
	Qty     int    `json:"qty"`
}
type SearchProductInput struct {
	CategoryID uint
	Keyword    string
	Page       int
	PageSize   int
}

type SpecItemDTO struct {
	SpecID    uint   `json:"specId"`
	ValueID   uint   `json:"valueId"`
	SpecName  string `json:"specName"`
	ValueName string `json:"valueName"`
}

type SKUDTO struct {
	SKUCode   string        `json:"skuCode"`
	SpecItems []SpecItemDTO `json:"specItems"`
	SpecDesc  string        `json:"specDesc"` // 展示文案，如 "红色 / L码"
	Price     int64         `json:"price"`
	Stock     int           `json:"stock"`
}
type ProductDTO struct {
	ProductID  uint      `json:"productId"`
	CategoryID uint      `json:"categoryId"`
	Name       string    `json:"name"`
	Desc       string    `json:"desc"`
	Status     int       `json:"status"`
	SKUs       []SKUDTO  `json:"skus"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ProductListResult struct {
	List  []ProductDTO `json:"list"`
	Total int          `json:"total"`
}
