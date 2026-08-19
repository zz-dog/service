package productpo

import (
	"context"

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

func (r *ProductRepository) FindByID(ctx context.Context, id uint) (*domainproduct.Product, error) {

	var po ProductPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainproduct.ErrProductNotFound
		}
	}
	return toProduct(po), nil
}

func (r *ProductRepository) FindBySKUCode(ctx context.Context, skuCode string) (*domainproduct.Product, error) {
	var po ProductPO
	if err := r.db.WithContext(ctx).Where("sku_code = ?", skuCode).First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainproduct.ErrSKUNotFound
		}
	}
	return toProduct(po), nil
}
func (r *ProductRepository) Save(ctx context.Context, p *domainproduct.Product) error {
	po := toPO(p)
	return r.db.WithContext(ctx).Create(&po).Error
}
func (r *ProductRepository) Search(ctx context.Context, categoryID uint, keyword string, page, pageSize int) ([]*domainproduct.Product, int64, error) {

	var (
		pos   []ProductPO
		total int64
	)
	db := r.db.WithContext(ctx).Model(&ProductPO{}).Where("category_id = ?", categoryID)
	if keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	products := make([]*domainproduct.Product, 0, len(pos))
	for _, po := range pos {
		products = append(products, toProduct(po))
	}
	return products, total, nil
}

func (r *ProductRepository) CountByCategory(ctx context.Context, categoryID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ProductPO{}).Where("category_id = ?", categoryID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProductRepository) DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error {
	return r.db.WithContext(ctx).Model(&ProductPO{}).Where("id = ?", productID).Update("stock", gorm.Expr("stock - ?", qty)).Error
}

func toProduct(p ProductPO) *domainproduct.Product {
	return &domainproduct.Product{
		ProductID:  p.ProductID,    // 商品ID
		CategoryID: p.CategoryID,   // 分类ID
		Name:       p.Name,         // 商品名称
		Desc:       p.Desc,         // 商品描述
		Status:     p.Status,       // 上下架状态
		SKUs:       toSKUs(p.SKUs), // 商品规格
		CreatedAt:  p.CreatedAt,    // 创建时间
		UpdatedAt:  p.UpdatedAt,    // 更新时间
	}
}

func toSKUs(s []SKUPO) []domainproduct.SKU {
	var taget = make([]domainproduct.SKU, len(s))
	for i, sku := range s {
		taget[i] = domainproduct.SKU{
			SKUCode:   sku.SKUCode,
			Spec:      sku.Spec,
			Price:     sku.Price,
			Stock:     sku.Stock,
			CreatedAt: sku.CreatedAt,
			UpdatedAt: sku.UpdatedAt,
		}
	}
	return taget
}

func toPO(p *domainproduct.Product) ProductPO {
	return ProductPO{
		ProductID:  p.ProductID,     // 商品ID
		CategoryID: p.CategoryID,    // 分类ID
		Name:       p.Name,          // 商品名称
		Desc:       p.Desc,          // 商品描述
		Status:     p.Status,        // 上下架状态
		SKUs:       toSKUPO(p.SKUs), // 商品规格
	}
}

func toSKUPO(s []domainproduct.SKU) []SKUPO {
	var taget = make([]SKUPO, len(s))
	for i, sku := range s {
		taget[i] = SKUPO{
			SKUCode: sku.SKUCode,
			Spec:    sku.Spec,
			Price:   sku.Price,
			Stock:   sku.Stock,
		}
	}
	return taget
}
