package productpo

import (
	"time"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type SKUPO struct {
	ProductID uint   `gorm:"index;comment:商品ID"`
	SKUCode   string `gorm:"primaryKey;size:64;comment:规格编码"`
	Price     int64  `gorm:"comment:单价(分)"`
	Stock     int    `gorm:"comment:库存"`
	// 规格项：引用规格库的维度+值（含名称快照）
	SpecItems []SKUSpecItemPO `gorm:"foreignKey:SKUCode;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:SKU规格组合"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SKUSpecItemPO SKU 规格项：联合主键 (sku_code, spec_id)。
// 一个 SKU 每个维度最多一项（领域层已校验）。
type SKUSpecItemPO struct {
	SKUCode   string `gorm:"primaryKey;size:64;comment:SKU编码"`
	SpecID    uint   `gorm:"primaryKey;comment:规格维度ID"`
	ValueID   uint   `gorm:"index;comment:规格值ID"`
	SpecName  string `gorm:"size:128;comment:维度名快照"`
	ValueName string `gorm:"size:128;comment:值名快照"`
}

type ProductPO struct {
	ProductID  uint    `gorm:"primaryKey;autoIncrement;comment:商品ID"`
	CategoryID uint    `gorm:"index;comment:分类ID"`
	Name       string  `gorm:"size:128;uniqueIndex;comment:商品名称"`
	Desc       string  `gorm:"size:255;comment:商品描述"`
	SKUs       []SKUPO `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:商品规格"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Status     domainproduct.Status `gorm:"tinyint;not null;default:1;comment:商品状态 1上架 0下架"`
}

func (ProductPO) TableName() string     { return "products" }
func (SKUPO) TableName() string         { return "product_skus" }
func (SKUSpecItemPO) TableName() string { return "sku_spec_items" }
