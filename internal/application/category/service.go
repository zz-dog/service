package categoryapp

import (
	"context"
	"errors"

	domain "github.com/wsc-zz/service/internal/domain/category"
	domainSpec "github.com/wsc-zz/service/internal/domain/spec"
)

type Service struct {
	repo        domain.CategoryRepository
	specRepo    domainSpec.SpecRepository // 绑定时校验规格存在、列表时取规格详情
	bindingRepo domain.CategorySpecRepository
}

func NewService(repo domain.CategoryRepository, specRepo domainSpec.SpecRepository, bindingRepo domain.CategorySpecRepository) *Service {
	return &Service{repo: repo, specRepo: specRepo, bindingRepo: bindingRepo}
}

// 创建分类
func (s *Service) Create(ctx context.Context, in CreateCategoryInput) (*CategoryDto, error) {
	existing, err := s.repo.FindByName(ctx, in.Name)
	// 存在同名分类则报错
	if existing != nil {
		return nil, domain.ErrCategoryAlreadyExists
	}
	// 分类不存在 则创建
	if err != nil && !errors.Is(err, domain.ErrCategoryNotFound) {
		return nil, err
	}

	// 创建分类
	c, err := domain.NewCategory(in.Name, in.ParentID, in.Sort)
	if err != nil {
		return nil, err
	}

	// 保存
	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}

	dto := toCategoryDTO(c)
	return &dto, nil
}

// Update 更新分类
func (s *Service) Update(ctx context.Context, in UpdateCategoryInput) (*CategoryDto, error) {
	c, err := s.repo.FindByID(ctx, in.CategoryID)
	// 分类不存在则报错
	if err != nil {
		return nil, err
	}

	// 更新分类
	if in.Name != nil {
		if err := c.Rename(*in.Name); err != nil {
			return nil, err
		}
	}
	if in.ParentID != nil {
		c.SetParentID(*in.ParentID)
	}
	if in.Sort != nil {
		c.SetSort(*in.Sort)
	}

	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	dto := toCategoryDTO(c)
	return &dto, nil
}

// Delete 删除分类：先级联解绑其全部规格，再删除分类本身
func (s *Service) Delete(ctx context.Context, id uint) error {

	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	// 级联解绑：分类没了，绑定关系失去意义
	if err := s.bindingRepo.DeleteByCategoryID(ctx, c.CategoryID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, c.CategoryID)
}

// List 获取分类列表
func (s *Service) List(ctx context.Context) ([]*CategoryDto, error) {
	return s.FindAll(ctx)
}

func (s *Service) FindAll(ctx context.Context) ([]*CategoryDto, error) {

	cs, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return toCategoryDTOs(cs), nil
}

// BindSpec 绑定规格到分类：校验双方存在且未重复绑定后保存关系
func (s *Service) BindSpec(ctx context.Context, in BindSpecInput) (*CategorySpecDto, error) {
	// 分类必须存在
	if _, err := s.repo.FindByID(ctx, in.CategoryID); err != nil {
		return nil, err
	}
	// 规格必须存在
	spec, err := s.specRepo.FindByID(ctx, in.SpecID)
	if err != nil {
		return nil, err
	}
	// 已绑定则报错，避免重复
	if _, err := s.bindingRepo.Find(ctx, in.CategoryID, in.SpecID); err == nil {
		return nil, domain.ErrCategorySpecAlreadyBound
	} else if !errors.Is(err, domain.ErrCategorySpecNotFound) {
		return nil, err
	}

	binding, err := domain.NewCategorySpec(in.CategoryID, in.SpecID, in.Sort, in.Required)
	if err != nil {
		return nil, err
	}
	if err := s.bindingRepo.Save(ctx, binding); err != nil {
		return nil, err
	}
	return toCategorySpecDTO(binding, spec), nil
}

// UnbindSpec 解除分类与规格的绑定
func (s *Service) UnbindSpec(ctx context.Context, in UnbindSpecInput) error {
	return s.bindingRepo.Delete(ctx, in.CategoryID, in.SpecID)
}

// ListSpecs 查询分类下绑定的全部规格（含规格值，便于前端直接渲染选择器）
func (s *Service) ListSpecs(ctx context.Context, categoryID uint) ([]*CategorySpecDto, error) {
	// 分类必须存在
	if _, err := s.repo.FindByID(ctx, categoryID); err != nil {
		return nil, err
	}
	bindings, err := s.bindingRepo.FindByCategoryID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	// 批量取规格，避免循环内逐条查询的 N+1
	specIDs := make([]uint, 0, len(bindings))
	for _, b := range bindings {
		specIDs = append(specIDs, b.SpecID)
	}
	specs, err := s.specRepo.FindByIDs(ctx, specIDs)
	if err != nil {
		return nil, err
	}
	specByID := make(map[uint]*domainSpec.Spec, len(specs))
	for _, spec := range specs {
		specByID[spec.SpecID] = spec
	}
	dtos := make([]*CategorySpecDto, 0, len(bindings))
	for _, b := range bindings {
		spec, ok := specByID[b.SpecID]
		if !ok {
			return nil, domainSpec.ErrSpecNotFound
		}
		dtos = append(dtos, toCategorySpecDTO(b, spec))
	}
	return dtos, nil
}

func toCategorySpecDTO(b *domain.CategorySpec, spec *domainSpec.Spec) *CategorySpecDto {
	dto := &CategorySpecDto{
		CategoryID: b.CategoryID,
		SpecID:     b.SpecID,
		SpecName:   spec.Name,
		Sort:       b.Sort,
		Required:   b.Required,
		Values:     make([]SpecValueDto, 0, len(spec.Values)),
	}
	for _, v := range spec.Values {
		dto.Values = append(dto.Values, SpecValueDto{
			ValueID: v.ValueID,
			Name:    v.Name,
			Sort:    v.Sort,
		})
	}
	return dto
}
func toCategoryDTO(c *domain.Category) CategoryDto {
	return CategoryDto{
		CategoryID: c.CategoryID,
		Name:       c.Name,
		ParentID:   c.ParentID,
		Sort:       c.Sort,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

// toCategoryDTOs 批量将分类聚合根转换为视图对象
func toCategoryDTOs(cs []*domain.Category) []*CategoryDto {
	dtos := make([]*CategoryDto, 0, len(cs))
	for _, c := range cs {
		dto := toCategoryDTO(c)
		dtos = append(dtos, &dto)
	}
	return dtos
}
