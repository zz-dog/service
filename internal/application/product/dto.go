package productapp

import (
	"time"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type SKUInput struct {
	SKUCode string `json:"skuCode"`
	Spec    string `json:"spec"`
	Price   int64  `json:"price"` // 分
	Stock   int    `json:"stock"`
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

type SKUDTO struct {
	SKUCode string `json:"skuCode"`
	Spec    string `json:"spec"`
	Price   int64  `json:"price"`
	Stock   int    `json:"stock"`
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
