package address

import "context"

// AddressRepository 是收货地址聚合根的仓储接口（端口）。
// 领域层定义契约，持久化实现由基础设施层（infrastructure/persistence）提供。
type AddressRepository interface {
	// Save 新增或更新地址：AddressID 为 0 时新增，否则更新全字段
	Save(ctx context.Context, a *Address) error
	// FindByID 按主键查询（自动过滤软删行）；未找到时返回 ErrAddressNotFound
	FindByID(ctx context.Context, id uint) (*Address, error)
	// FindByUserID 查询用户的地址列表，默认地址排最前
	FindByUserID(ctx context.Context, userID uint) ([]*Address, error)
	// CountByUserID 统计用户的地址数量（用于上限校验）
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	// ClearDefault 清除该用户所有默认标记，仅 SetDefault/Create 事务内调用
	ClearDefault(ctx context.Context, userID uint) error
	// Delete 软删除地址
	Delete(ctx context.Context, a *Address) error
}
