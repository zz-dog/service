package category

import "context"

// CategoryRepository 分类聚合根仓储端口
type CategoryRepository interface {
	FindByID(ctx context.Context, id uint) (*Category, error)
	FindByName(ctx context.Context, name string) (*Category, error)
	// FindAll 查询全部分类（分类数量通常不多，不分页），可按 sort 排序
	FindAll(ctx context.Context) ([]*Category, error)
	Save(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id uint) error
}

// CategorySpecRepository 分类-规格绑定仓储端口
type CategorySpecRepository interface {
	// Find 查询一条绑定关系，不存在返回 ErrCategorySpecNotFound
	Find(ctx context.Context, categoryID, specID uint) (*CategorySpec, error)
	// FindByCategoryID 查询分类下全部绑定，按 sort 排序
	FindByCategoryID(ctx context.Context, categoryID uint) ([]*CategorySpec, error)
	// FindBySpecID 查询绑定了某规格的全部分类（规格删除前做引用检查用）
	FindBySpecID(ctx context.Context, specID uint) ([]*CategorySpec, error)
	Save(ctx context.Context, cs *CategorySpec) error
	// Delete 解除一条绑定，不存在返回 ErrCategorySpecNotFound
	Delete(ctx context.Context, categoryID, specID uint) error
	// DeleteByCategoryID 分类删除时级联解绑其全部规格
	DeleteByCategoryID(ctx context.Context, categoryID uint) error
}
