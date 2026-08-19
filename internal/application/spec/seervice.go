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
	if err := s.repo.Save(ctx, spec); err != nil {
		return nil, err
	}
	return toDto(spec), nil
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
