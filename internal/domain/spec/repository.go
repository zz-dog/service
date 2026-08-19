package spec

import "context"

type SpecRepository interface {
	FindByID(ctx context.Context, id uint) (*Spec, error)
	FindByName(ctx context.Context, name string) (*Spec, error)
	Save(ctx context.Context, spec *Spec) error
	Delete(ctx context.Context, id uint) error
}
