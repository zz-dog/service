package productapp

import (
	"context"
	"fmt"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo    domainproduct.ProductRepository
	specDir SpecDirectory
}

func NewService(repo domainproduct.ProductRepository, specDir SpecDirectory) *Service {
	return &Service{repo: repo, specDir: specDir}
}

// Create 创建商品：
// ① 校验 SKU 规格项 (SpecID, ValueID) 属于分类绑定的规格集；
// ② 快照名称一律取规格库权威值，不采信客户端传入的名称；
// ③ 落库走 Create（纯插入）并回填 ProductID，返回的 DTO 带真实 ID 和时间戳。
func (s *Service) Create(ctx context.Context, in CreateProductInput) (*ProductDTO, error) {
	specs, err := s.specDir.SpecsForCategory(ctx, in.CategoryID)
	if err != nil {
		return nil, err
	}
	skus, err := buildSKUs(in.SKUs, specs)
	if err != nil {
		return nil, err
	}
	p, err := domainproduct.NewProduct(in.CategoryID, in.Name, in.Desc, skus)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

func (s *Service) List(ctx context.Context, in domainproduct.ListQuery) (ProductListResult, error) {
	products, total, err := s.repo.List(ctx, in)
	return ProductListResult{
		List:  toProductDTOs(products),
		Total: total,
	}, err
}

// Update 更新商品：
// ① 校验 SKU 规格项 (SpecID, ValueID) 属于分类绑定的规格集；
// ② 快照名称一律取规格库权威值，不采信客户端传入的名称；
// ③ 落库走 Save（更新）并返回的 DTO 带真实 ID 和时间戳。
func (s *Service) Update(ctx context.Context, in UpdateProductInput) (*ProductDTO, error) {

	p, err := s.repo.FindByID(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	p.UpdateInfo(in.CategoryID, in.Name, in.Desc, in.Urls)
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// buildSKUs 把输入 SKU 转为领域 SKU：
// 规格项必须在分类绑定的规格集内，名称快照取权威值填充；
// SKUCode 不收客户端值，由规格组合派生（DeriveSKUCode）。
func buildSKUs(inputs []SKUInput, specs []*CategorySpecView) ([]domainproduct.SKU, error) {
	specByID := make(map[uint]*CategorySpecView, len(specs))
	for _, sp := range specs {
		specByID[sp.SpecID] = sp
	}
	skus := make([]domainproduct.SKU, 0, len(inputs))
	for _, in := range inputs {
		items := make([]domainproduct.SpecItem, 0, len(in.SpecItems))
		for _, item := range in.SpecItems {
			spec, ok := specByID[item.SpecID]
			if !ok {
				return nil, fmt.Errorf("%w：specId=%d", ErrSpecNotBoundToCategory, item.SpecID)
			}
			valueName, ok := spec.valueName(item.ValueID)
			if !ok {
				return nil, fmt.Errorf("%w：specId=%d valueId=%d", ErrSpecValueNotInSpec, item.SpecID, item.ValueID)
			}
			items = append(items, domainproduct.SpecItem{
				SpecID:    item.SpecID,
				ValueID:   item.ValueID,
				SpecName:  spec.SpecName,
				ValueName: valueName,
			})
		}
		skus = append(skus, domainproduct.SKU{
			SKUCode:   domainproduct.DeriveSKUCode(items),
			SpecItems: items,
			Price:     in.Price,
			Stock:     in.Stock,
		})
	}
	return skus, nil
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
		Status:     int(p.Status),
		SKUs:       skus,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func toProductDTOs(ps []*domainproduct.Product) []ProductDTO {
	dtos := make([]ProductDTO, 0, len(ps))
	for _, p := range ps {
		dtos = append(dtos, toProductDTO(p))
	}
	return dtos
}
