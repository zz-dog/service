package specapp

import (
	"context"

	domainSpec "github.com/wsc-zz/service/internal/domain/spec"
)

type Service struct {
	repo domainSpec.SpecRepository
}

func NewService(repo domainSpec.SpecRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in SpecInput) (*SpecDTO, error) {

	spec, err := domainSpec.NewSpec(in.Name, in.Sort)
	if err != nil {
		return nil, err
	}
	for _, v := range in.Values {
		if err := spec.AddValue(v.Name, v.Sort); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Save(ctx, spec); err != nil {
		return nil, err
	}
	return toDto(spec), nil
}

// Update 整包更新规格：重命名/调序，并对规格值对账——
// 带ID的值更新、不带ID的值新增、请求中缺席的值删除。
func (s *Service) Update(ctx context.Context, id uint, in UpdateSpecInput) (*SpecDTO, error) {
	spec, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := spec.Rename(in.Name); err != nil {
		return nil, err
	}
	spec.SetSort(in.Sort)

	keep := make(map[uint]struct{}, len(in.Values))
	for _, v := range in.Values {
		if v.ValueID > 0 {
			if err := spec.UpdateValue(v.ValueID, v.Name, v.Sort); err != nil {
				return nil, err
			}
			keep[v.ValueID] = struct{}{}
			continue
		}
		if err := spec.AddValue(v.Name, v.Sort); err != nil {
			return nil, err
		}
	}
	// 新增值(ValueID==0)不参与对账，不能在此循环中被删掉
	for _, v := range append([]domainSpec.SpecValue(nil), spec.Values...) {
		if v.ValueID == 0 {
			continue
		}
		if _, ok := keep[v.ValueID]; !ok {
			if err := spec.RemoveValue(v.ValueID); err != nil {
				return nil, err
			}
		}
	}

	if err := s.repo.Save(ctx, spec); err != nil {
		return nil, err
	}
	return toDto(spec), nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) FindByID(ctx context.Context, id uint) (*SpecDTO, error) {
	spec, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDto(spec), nil
}
func (s *Service) FindByName(ctx context.Context, name string) (*SpecDTO, error) {
	spec, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return toDto(spec), nil
}
func (s *Service) List(ctx context.Context, q domainSpec.SpecListQuery) (ListDto, error) {
	specs, total, err := s.repo.SpecList(ctx, q)
	if err != nil {
		return ListDto{}, err
	}
	dtos := make([]*SpecDTO, 0, len(specs))
	for _, spec := range specs {
		dtos = append(dtos, toDto(spec))
	}
	return ListDto{
		List:  dtos,
		Total: total,
	}, nil
}
func toDto(s *domainSpec.Spec) *SpecDTO {
	return &SpecDTO{
		SpecID:    s.SpecID,
		Name:      s.Name,
		Sort:      s.Sort,
		Values:    toValueDTOs(s.Values),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func toValueDTOs(values []domainSpec.SpecValue) []SpecValueDTO {
	var dtoValues []SpecValueDTO
	for _, v := range values {
		dtoValues = append(dtoValues, SpecValueDTO{
			SpecValueID: v.ValueID,
			Name:        v.Name,
			Sort:        v.Sort,
		})
	}
	return dtoValues
}
