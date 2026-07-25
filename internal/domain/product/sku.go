package product

import "time"

// SKU 是商品规格（值对象）。
// 一个商品有多个 SKU（如"红色 L 码"、"蓝色 M 码"），每个 SKU 独立价格和库存。
// 没有独立身份，作为 Product 聚合根的一部分存在。
type SKU struct {
	SKUCode   string // 规格编码，商品内唯一，如 "RED-L"
	Spec      string // 规格描述，如 "红色 / L码"
	Price     int64  // 单价，单位：分
	Stock     int    // 库存
	CreatedAt time.Time
	UpdatedAt time.Time
}

// validate 校验 SKU 字段
func (s SKU) validate() error {
	if s.SKUCode == "" {
		return ErrDuplicateSKUCode
	}
	if s.Price <= 0 {
		return ErrInvalidPrice
	}
	if s.Stock < 0 {
		return ErrInvalidStock
	}
	return nil
}
