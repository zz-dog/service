package spec

import "context"

type SpecRepository interface {
	FindByID(ctx context.Context, id uint) (*Spec, error)
	FindByName(ctx context.Context, name string) (*Spec, error)
	// FindByIDs 批量按ID查询规格（含规格值），ids 为空返回空切片
	FindByIDs(ctx context.Context, ids []uint) ([]*Spec, error)
	Save(ctx context.Context, spec *Spec) error
	Delete(ctx context.Context, id uint) error
	SpecList(ctx context.Context, q SpecListQuery) ([]*Spec, int, error)
}

type SpecListQuery struct {
	Page     int
	PageSize int
	SpecID   uint
	Name     string
}
