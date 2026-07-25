package product

import "context"

// ProductRepository 商品聚合根仓储端口
type ProductRepository interface {
	// FindByID 按主键查询商品（含 SKU）；未找到返回 ErrProductNotFound
	FindByID(ctx context.Context, id uint) (*Product, error)
	// FindBySKUCode 按 SKU 编码查询商品（用于下单扣库存），返回商品与 SKU 索引
	FindBySKUCode(ctx context.Context, skuCode string) (*Product, error)
	// Search 分页查询商品，支持按分类/名称筛选，返回列表与总数
	Search(ctx context.Context, categoryID uint, keyword string, page, pageSize int) ([]*Product, int64, error)
	// Save 新增或更新商品（含 SKU）
	Save(ctx context.Context, p *Product) error
	// DeductStock 原子扣减某 SKU 库存，并发安全；库存不足返回 ErrInsufficientStock
	DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error
	// CountByCategory 统计某分类下的商品数量（删除分类前校验）
	CountByCategory(ctx context.Context, categoryID uint) (int64, error)
}
