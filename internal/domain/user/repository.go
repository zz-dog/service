package user

import "context"

// UserRepository 是用户聚合根的仓储接口（端口）。
// 领域层定义契约，持久化实现由基础设施层（infrastructure/persistence）提供。
type UserRepository interface {
	// FindByUsername 按用户名查询；未找到时返回 ErrUserNotFound
	FindByUsername(ctx context.Context, username string) (*User, error)
	// FindByID 按主键查询；未找到时返回 ErrUserNotFound
	FindByID(ctx context.Context, id uint) (*User, error)
	// Save 新增或更新用户：主键为 0 时新增，否则更新全字段
	Save(ctx context.Context, u *User) error

	// GetUserList 按条件分页查询用户列表，返回符合条件总记录数
	GetUserList(ctx context.Context, q UserListQuery) ([]*User, int, error)
}

// UserListQuery 用户列表查询条件，零值字段表示该条件不参与过滤。
type UserListQuery struct {
	Page     int
	PageSize int
	UserID   uint
	Username string
}
