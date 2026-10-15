package transaction

import (
	"context"

	addressdomain "github.com/wsc-zz/service/internal/domain/address"
	orderdomain "github.com/wsc-zz/service/internal/domain/order"
	productdomain "github.com/wsc-zz/service/internal/domain/product"
	addresspo "github.com/wsc-zz/service/internal/infrastructure/persistence/address"
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

// RunAddress 在同一事务内执行 fn，fn 收到与事务绑定的地址仓储。
// 典型用途：SetDefault 的"先清零其它默认标记，再置一条为默认"。
func (r *Runner) RunAddress(ctx context.Context, fn func(addressdomain.AddressRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(addresspo.NewAddressRepository(tx))
	})
}
