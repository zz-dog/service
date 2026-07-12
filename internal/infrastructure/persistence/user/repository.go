package userpo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/wsc-zz/service/internal/domain/user"
)

// UserRepository 是 domain/user.UserRepository 的 GORM 实现。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造仓储实现，注入 *gorm.DB。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// 编译期断言：确保实现满足领域接口
var _ user.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	var po UserPO
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*user.User, error) {
	var po UserPO
	err := r.db.WithContext(ctx).First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	po := toPO(u)
	// 主键为 0 走新增，否则更新全字段，与原 Create/Save 行为一致
	if u.UserID == 0 {
		if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		// 回填自增主键与时间戳到领域实体，使调用方拿到持久化后的状态
		u.UserID = po.UserID
		u.CreatedAt = po.CreatedAt
		u.UpdatedAt = po.UpdatedAt
		return nil
	}
	return r.db.WithContext(ctx).Save(po).Error
}
