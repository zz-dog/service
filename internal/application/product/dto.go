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

// MediaInput 商品媒体资源输入：类型 1图片 2视频 3模型。
type MediaInput struct {
	Type int    `json:"type"` // 资源类型
	URL  string `json:"url"`  // 资源地址
}

type SKUInput struct {
	// SKUCode 不收客户端值，派生（值ID按维度ID排序拼接）
	SpecItems []SpecItemInput // 规格组合
	Price     int64           // 价格
	Stock     int             // 库存
	Picture   string
}
type CreateProductInput struct {
	CategoryID uint
	MerchantID uint // 管理端语义:代谁铺货;0 = 平台自营。商户端发布不收这个字段(服务端推导)
	Name       string
	Desc       string
	SKUs       []SKUInput
	Medias     []MediaInput // 商品媒体资源，按顺序展示
}
type UpdateProductInput struct {
	ProductID  uint
	CategoryID uint
	Name       string
	Desc       string
	Medias     []MediaInput // 整体替换：不传则清空
}
type DeductStockInput struct {
	SKUCode string
	Qty     int
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

type MediaDTO struct {
	Type int    `json:"type"` // 类型 1图片 2视频 3模型
	URL  string `json:"url"`  // 资源地址
}

type SKUDTO struct {
	SKUCode   string        `json:"skuCode"`
	SpecItems []SpecItemDTO `json:"specItems"`
	SpecDesc  string        `json:"specDesc"` // 展示文案，如 "红色 / L码"
	Price     int64         `json:"price"`
	Stock     int           `json:"stock"`
}
type ProductDTO struct {
	ProductID    uint       `json:"productId"`
	CategoryID   uint       `json:"categoryId"`
	MerchantID   uint       `json:"merchantId"`   // 归属商户;0 = 平台自营
	Name         string     `json:"name"`
	Desc         string     `json:"desc"`
	Status       int        `json:"status"`       // 1上架 2下架 3待审核 4已驳回 5违规下架
	RejectReason string     `json:"rejectReason"` // 平台否决理由;商户在"我的商品"里看
	SKUs         []SKUDTO   `json:"skus"`
	Medias       []MediaDTO `json:"medias"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type ProductListResult struct {
	List  []ProductDTO `json:"list"`
	Total int          `json:"total"`
}
