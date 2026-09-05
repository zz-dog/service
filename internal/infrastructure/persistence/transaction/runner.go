package transaction

import (
	"context"

	orderdomain "github.com/wsc-zz/service/internal/domain/order"
	productdomain "github.com/wsc-zz/service/internal/domain/product"
	orderpo "github.com/wsc-zz/service/internal/infrastructure/persistence/order"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	"gorm.io/gorm"
)

type Runner struct {
	db *gorm.DB
}

func NewRunner(db *gorm.DB) *Runner {
	return &Runner{db: db}
}

func (r *Runner) Run(ctx context.Context, fn func(orderdomain.OrderRepository, productdomain.ProductRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(orderpo.NewOrderRepository(tx), productpo.NewProductRepository(tx))
	})
}
