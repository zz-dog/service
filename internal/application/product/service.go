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
		items := make([]domainproduct.SpecItem, 0, len(s.SpecItems))
		for _, item := range s.SpecItems {
			items = append(items, domainproduct.SpecItem{
				SpecID:    item.SpecID,
				ValueID:   item.ValueID,
				SpecName:  item.SpecName,
				ValueName: item.ValueName,
			})
		}
		skus = append(skus, domainproduct.SKU{
			SKUCode:   s.SKUCode,
			SpecItems: items,
			Price:     s.Price,
			Stock:     s.Stock,
		})
	}
	return skus
}

func toSpecItemDTOs(items []domainproduct.SpecItem) []SpecItemDTO {
	dtos := make([]SpecItemDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, SpecItemDTO{
			SpecID:    item.SpecID,
			ValueID:   item.ValueID,
			SpecName:  item.SpecName,
			ValueName: item.ValueName,
		})
	}
	return dtos
}

func toProductDTO(p *domainproduct.Product) ProductDTO {
	skus := make([]SKUDTO, 0, len(p.SKUs))
	for _, s := range p.SKUs {
		skus = append(skus, SKUDTO{
			SKUCode:   s.SKUCode,
			SpecItems: toSpecItemDTOs(s.SpecItems),
			SpecDesc:  s.SpecDesc(),
			Price:     s.Price,
			Stock:     s.Stock,
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
