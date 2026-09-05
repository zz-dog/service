package productpo

import (
	"context"
	"errors"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
	}
}

// preloadSKUs 预加载 SKU 及其规格项：Preload("SKUs").Preload("SKUs.SpecItems")

func (r *ProductRepository) FindByID(ctx context.Context, id uint) (*domainproduct.Product, error) {
	var po ProductPO
	err := r.db.WithContext(ctx).
		Preload("SKUs").Preload("SKUs.SpecItems").
		Where("product_id = ?", id).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainproduct.ErrProductNotFound
		}
		return nil, err
	}
	return toProduct(po), nil
}

func (r *ProductRepository) FindByProductAndSKUCode(ctx context.Context, productID uint, skuCode string) (*domainproduct.Product, error) {
	// SKU 编码在 product_skus 上，先定位商品再加载聚合
	var sku SKUPO
	err := r.db.WithContext(ctx).
		Where("product_id = ? AND sku_code = ?", productID, skuCode).
		First(&sku).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainproduct.ErrSKUNotFound
		}
		return nil, err
	}
	return r.FindByID(ctx, sku.ProductID)
}

// Create 新增商品聚合：纯插入语义，落库后回填生成的 ProductID 和时间戳。
// 与 Save（Upsert）区分，创建流程不会误更新已有商品。
func (r *ProductRepository) Create(ctx context.Context, p *domainproduct.Product) error {
	po := toPO(p)
	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return err
	}
	p.ProductID = po.ProductID
	p.CreatedAt = po.CreatedAt
	p.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *ProductRepository) Save(ctx context.Context, p *domainproduct.Product) error {
	po := toPO(p)
	// FullSaveAssociations：更新时同步 Upsert 子表（SKU / 规格项）
	return r.db.WithContext(ctx).
		Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&po).Error
}

func (r *ProductRepository) CountByCategory(ctx context.Context, categoryID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ProductPO{}).Where("category_id = ?", categoryID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// DeductStock 原子扣减 SKU 库存：条件更新 stock >= qty 防并发超卖。
// 注意库存列在 product_skus 上，不在 products 上。
func (r *ProductRepository) DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error {
	if qty <= 0 {
		return domainproduct.ErrInvalidStock
	}
	res := r.db.WithContext(ctx).Model(&SKUPO{}).
		Where("product_id = ? AND sku_code = ?", productID, skuCode).
		Where("stock >= ?", qty).
		Update("stock", gorm.Expr("stock - ?", qty))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 未命中：要么 SKU 不存在，要么库存不足
		var sku SKUPO
		if err := r.db.WithContext(ctx).
			Where("product_id = ? AND sku_code = ?", productID, skuCode).
			First(&sku).Error; err != nil {
			return domainproduct.ErrSKUNotFound
		}
		return domainproduct.ErrInsufficientStock
	}
	return nil
}

// RestockStock 恢复 SKU 库存。
func (r *ProductRepository) RestockStock(ctx context.Context, productID uint, skuCode string, qty int) error {
	if qty <= 0 {
		return domainproduct.ErrInvalidStock
	}
	res := r.db.WithContext(ctx).Model(&SKUPO{}).
		Where("product_id = ? AND sku_code = ?", productID, skuCode).
		Update("stock", gorm.Expr("stock + ?", qty))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainproduct.ErrSKUNotFound
	}
	return nil
}

func (r *ProductRepository) List(ctx context.Context, q domainproduct.ListQuery) ([]*domainproduct.Product, int, error) {
	var (
		pos   []ProductPO
		total int64
	)
	db := r.db.WithContext(ctx).Model(&ProductPO{})
	if q.CategoryID > 0 {
		db = db.Where("category_id = ?", q.CategoryID)
	}
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.ProductID > 0 {
		db = db.Where("product_id = ?", q.ProductID)
	}
	if q.Status > 0 {
		db = db.Where("status = ?", q.Status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.PageSize
	if err := db.Preload("SKUs").Preload("SKUs.SpecItems").
		Offset(offset).Limit(q.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	products := make([]*domainproduct.Product, 0, len(pos))
	for _, po := range pos {
		products = append(products, toProduct(po))
	}
	return products, int(total), nil

}
func toProduct(p ProductPO) *domainproduct.Product {
	return &domainproduct.Product{
		ProductID:  p.ProductID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Desc:       p.Desc,
		Status:     p.Status,
		SKUs:       toSKUs(p.SKUs),
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func toSKUs(s []SKUPO) []domainproduct.SKU {
	var target = make([]domainproduct.SKU, len(s))
	for i, sku := range s {
		target[i] = domainproduct.SKU{
			SKUCode:   sku.SKUCode,
			SpecItems: toSpecItems(sku.SpecItems),
			Price:     sku.Price,
			Stock:     sku.Stock,
			CreatedAt: sku.CreatedAt,
			UpdatedAt: sku.UpdatedAt,
		}
	}
	return target
}

func toSpecItems(items []SKUSpecItemPO) []domainproduct.SpecItem {
	var target = make([]domainproduct.SpecItem, len(items))
	for i, item := range items {
		target[i] = domainproduct.SpecItem{
			SpecID:    item.SpecID,
			ValueID:   item.ValueID,
			SpecName:  item.SpecName,
			ValueName: item.ValueName,
		}
	}
	return target
}

func toPO(p *domainproduct.Product) ProductPO {
	return ProductPO{
		ProductID:  p.ProductID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Desc:       p.Desc,
		Status:     p.Status,
		SKUs:       toSKUPO(p.SKUs),
	}
}

func toSKUPO(s []domainproduct.SKU) []SKUPO {
	var target = make([]SKUPO, len(s))
	for i, sku := range s {
		target[i] = SKUPO{
			SKUCode:   sku.SKUCode,
			Price:     sku.Price,
			Stock:     sku.Stock,
			SpecItems: toSpecItemPO(sku.SpecItems),
		}
	}
	return target
}

func toSpecItemPO(items []domainproduct.SpecItem) []SKUSpecItemPO {
	var target = make([]SKUSpecItemPO, len(items))
	for i, item := range items {
		target[i] = SKUSpecItemPO{
			SpecID:    item.SpecID,
			ValueID:   item.ValueID,
			SpecName:  item.SpecName,
			ValueName: item.ValueName,
		}
	}
	return target
}
