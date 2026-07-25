package product

import "time"

// 商品状态

type Status int

const (
	StatusOnShelf  Status = iota // 上架
	StatusOffShelf               // 下架
)

// Product 是商品聚合根，管理 SKU 列表。
type Product struct {
	ProductID  uint      // 商品ID
	CategoryID uint      // 分类ID
	Name       string    // 商品名称
	Desc       string    // 商品描述
	Status     Status    // 上下架状态
	SKUs       []SKU     // 商品规格
	Urls       []string  // 商品图片
	CreatedAt  time.Time // 创建时间
	UpdatedAt  time.Time // 更新时间
}

// NewProduct 创建商品（默认上架）。
// 充血构造函数：校验名称、SKU 列表、SKU 编码不重复。
func NewProduct(categoryID uint, name, desc string, skus []SKU) (*Product, error) {
	if name == "" {
		return nil, ErrEmptyProductName
	}
	if len(skus) == 0 {
		return nil, ErrEmptySKUs
	}
	seen := make(map[string]bool, len(skus))
	for _, s := range skus {
		if err := s.validate(); err != nil {
			return nil, err
		}
		if seen[s.SKUCode] {
			return nil, ErrDuplicateSKUCode
		}
		seen[s.SKUCode] = true
	}
	return &Product{
		CategoryID: categoryID,
		Name:       name,
		Desc:       desc,
		Status:     StatusOnShelf,
		SKUs:       skus,
	}, nil
}

// UpdateInfo 修改商品基础信息（名称、描述、分类）。
func (p *Product) UpdateInfo(categoryID uint, name, desc string) error {
	if name == "" {
		return ErrEmptyProductName
	}
	p.CategoryID = categoryID
	p.Name = name
	p.Desc = desc
	return nil
}

// ReplaceSKUs 替换全部 SKU（管理端重新设置规格）。
// 校验编码不重复。
func (p *Product) ReplaceSKUs(skus []SKU) error {
	if len(skus) == 0 {
		return ErrEmptySKUs
	}
	seen := make(map[string]bool, len(skus))
	for _, s := range skus {
		if err := s.validate(); err != nil {
			return nil
		}
		if seen[s.SKUCode] {
			return ErrDuplicateSKUCode
		}
		seen[s.SKUCode] = true
	}
	p.SKUs = skus
	return nil
}

// OnShelf 上架
func (p *Product) OnShelf() { p.Status = StatusOnShelf }

// OffShelf 下架
func (p *Product) OffShelf() { p.Status = StatusOffShelf }

// IsOnShelf 是否上架
func (p *Product) IsOnShelf() bool { return p.Status == StatusOnShelf }

// findSKU 按 code 找 SKU
func (p *Product) findSKU(code string) (int, bool) {
	for i, s := range p.SKUs {
		if s.SKUCode == code {
			return i, true
		}
	}
	return 0, false
}

// DeductStock 扣减某 SKU 的库存（下单时调用）。
// 在聚合根内校验：库存不足返回 ErrInsufficientStock，下架商品返回 ErrProductOffShelf。
// 注意：内存层校验后，仓储层还要用条件更新防并发超卖。
func (p *Product) DeductStock(skuCode string, qty int) error {
	if !p.IsOnShelf() {
		return ErrProductOffShelf
	}
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.findSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	if p.SKUs[idx].Stock < qty {
		return ErrInsufficientStock
	}
	p.SKUs[idx].Stock -= qty
	return nil
}

// Restock 增加某 SKU 库存（管理端补货）
func (p *Product) Restock(skuCode string, qty int) error {
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.findSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	p.SKUs[idx].Stock += qty
	return nil
}
