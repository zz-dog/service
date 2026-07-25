package productapp

import (
	"context"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo domainproduct.ProductRepository
}

func NewService(repo domainproduct.ProductRepository) *Service {
	return &Service{repo: repo}
}

// Create 创建商品
func (s *Service) Create(ctx context.Context, in CreateProductInput) (*ProductDTO, error) {
	skus := toDomainSKUs(in.SKUs)
	p, err := domainproduct.NewProduct(in.CategoryID, in.Name, in.Desc, skus)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}
func toDomainSKUs(in []SKUInput) []domainproduct.SKU {
	skus := make([]domainproduct.SKU, 0, len(in))
	for _, s := range in {
		skus = append(skus, domainproduct.SKU{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return skus
}
func toProductDTO(p *domainproduct.Product) ProductDTO {
	skus := make([]SKUDTO, 0, len(p.SKUs))
	for _, s := range p.SKUs {
		skus = append(skus, SKUDTO{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return ProductDTO{
		ProductID:  p.ProductID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Desc:       p.Desc,
		Status:     p.Status,
		SKUs:       skus,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}
