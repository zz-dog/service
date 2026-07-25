package product

type SKU struct {
	SKUCode string `gorm:"primaryKey;autoIncrement;comment:分类组件"`
	Spec    string `gorm:"size:128;comment:规格"`
	Price   int64  `gorm:"comment:单价(分)"`
	Stock   int
}
