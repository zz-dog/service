package product

import "context"

// ProductRepository 商品聚合根仓储端口
type ProductRepository interface {
	// FindByID 按主键查询商品（含 SKU）；未找到返回 ErrProductNotFound
	FindByID(ctx context.Context, id uint) (*Product, error)
	// FindByProductAndSKUCode 按商品ID+SKU编码查询商品（用于下单扣库存）
	// SKU 编码仅商品内唯一，必须配合商品ID定位
	FindByProductAndSKUCode(ctx context.Context, productID uint, skuCode string) (*Product, error)

	Create(ctx context.Context, p *Product) error
	// Save 新增或更新商品（含 SKU）
	Save(ctx context.Context, p *Product) error
	// DeductStock 原子扣减某 SKU 库存，并发安全；库存不足返回 ErrInsufficientStock
	DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error
	// CountByCategory 统计某分类下的商品数量（删除分类前校验）
	CountByCategory(ctx context.Context, categoryID uint) (int64, error)
	List(ctx context.Context, q ListQuery) ([]*Product, int, error)
}

type ListQuery struct {
	Page       int
	PageSize   int
	Status     Status
	Name       string
	CategoryID uint
	ProductID  uint
}
