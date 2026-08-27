package productapp

import (
	"time"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

// SpecItemInput SKU 规格项输入：引用规格库维度/值，名称可选（不传则由应用层从规格库补全）
type SpecItemInput struct {
	SpecID    uint   `json:"specId"`    // 规格维度ID
	ValueID   uint   `json:"valueId"`   // 规格值ID
	SpecName  string `json:"specName"`  // 维度名（快照）
	ValueName string `json:"valueName"` // 值名（快照）
}

type SKUInput struct {
	SKUCode   string          `json:"skuCode"`   // 商品规格编码
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
	ProductID  uint       `json:"-"`
	CategoryID uint       `json:"categoryId"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	SKUs       []SKUInput `json:"skus"` // 整体替换 SKU
}
type DeductStockInput struct {
	SKUCode string `json:"skuCode"`
	Qty     int    `json:"qty"`
}
type SearchProductInput struct {
	CategoryID uint   `json:"-"` // 从 query 取
	Keyword    string `json:"-"`
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
	ProductID  uint                 `json:"productId"`
	CategoryID uint                 `json:"categoryId"`
	Name       string               `json:"name"`
	Desc       string               `json:"desc"`
	Status     domainproduct.Status `json:"status"`
	SKUs       []SKUDTO             `json:"skus"`
	CreatedAt  time.Time            `json:"createdAt"`
	UpdatedAt  time.Time            `json:"updatedAt"`
}

type ProductListResult struct {
	List     []ProductDTO `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}
