package transaction

import (
	"context"

	addressdomain "github.com/wsc-zz/service/internal/domain/address"
	merchantdomain "github.com/wsc-zz/service/internal/domain/merchant"
	orderdomain "github.com/wsc-zz/service/internal/domain/order"
	productdomain "github.com/wsc-zz/service/internal/domain/product"
	addresspo "github.com/wsc-zz/service/internal/infrastructure/persistence/address"
	merchantpo "github.com/wsc-zz/service/internal/infrastructure/persistence/merchant"
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

// RunMerchant 在同一事务内执行 fn，fn 收到与事务绑定的申请单/商户仓储。
// 审核通过 = 申请单迁移 + 建商户 + 回填,三步必须同生共死:
// 撞店铺名唯一索引时整个回滚,申请单回到"待审核",不会出现"已通过但没商户"的孤儿状态。
func (r *Runner) RunMerchant(ctx context.Context, fn func(merchantdomain.ApplicationRepository, merchantdomain.MerchantRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(merchantpo.NewMerchantApplicationRepository(tx), merchantpo.NewMerchantRepository(tx))
	})
}
