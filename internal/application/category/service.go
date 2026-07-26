package categoryapp

import (
	"context"
	"errors"

	domain "github.com/wsc-zz/service/internal/domain/category"
)

type Service struct {
	repo domain.CategoryRepository
}

func NewService(repo domain.CategoryRepository) *Service {
	return &Service{repo: repo}
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

// Delete 删除分类
func (s *Service) Delete(ctx context.Context, id uint) error {

	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, c.CategoryID)
}

// List 获取分类列表

func (s *Service) List(ctx context.Context) ([]*CategoryDto, error) {
	categories, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return toCategoryDTOs(categories), nil
}

// toCategoryDTO 将分类聚合根转换为视图对象
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
