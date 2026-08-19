package productpo

import (
	"time"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type SKUPO struct {
	ProductID uint      `gorm:"index;comment:商品ID"`
	SKUCode   string    `gorm:"primaryKey;comment:规格编码"`
	Spec      string    `gorm:"size:128;comment:规格"` // 规格类如：颜色，尺寸等
	Price     int64     `gorm:"comment:单价(分)"`       // 单价(分)
	Stock     int       `gorm:"comment:库存"`          // 库存
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
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

func (ProductPO) TableName() string { return "products" }
func (SKUPO) TableName() string     { return "product_skus" }
