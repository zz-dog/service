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
	// Delete 软删除商品；未找到不报错（幂等）
	Delete(ctx context.Context, id uint) error
	// Save 新增或更新商品（含 SKU）
	Save(ctx context.Context, p *Product) error
	// DeductStock 原子扣减某 SKU 库存，并发安全；库存不足返回 ErrInsufficientStock
	DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error
	// RestockStock 恢复某 SKU 库存
	RestockStock(ctx context.Context, productID uint, skuCode string, qty int) error
	// CountByCategory 统计某分类下的商品数量（删除分类前校验）
	CountByCategory(ctx context.Context, categoryID uint) (int64, error)
	List(ctx context.Context, q ListQuery) ([]*Product, int, error)
	// UpdateStatus 条件更新 from→to,返回是否本请求生效——上下架共用
	UpdateStatus(ctx context.Context, id uint, from, to Status) (bool, error)
	// Reject 条件更新 待审核→已驳回,理由随行
	Reject(ctx context.Context, id uint, reason string) (bool, error)
	// Takedown 条件更新 已上架→违规下架,理由随行。
	// 与 Reject 分开命名:接口上读得出"这是处置不是审核"
	Takedown(ctx context.Context, id uint, reason string) (bool, error)
}

type ListQuery struct {
	Page       int
	PageSize   int
	Status     *Status // nil 不过滤；指向具体状态则只查该状态(1上架 2下架 3待审核 4已驳回 5违规下架)
	Name       string
	CategoryID uint
	ProductID  uint
	MerchantID uint // >0 按归属过滤——商户"我的商品"靠它,归属即过滤条件
}
