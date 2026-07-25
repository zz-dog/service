package category

import "context"

// CategoryRepository 分类聚合根仓储端口
type CategoryRepository interface {
	FindByID(ctx context.Context, id uint) (*Category, error)
	// FindAll 查询全部分类（分类数量通常不多，不分页），可按 sort 排序
	FindAll(ctx context.Context) ([]*Category, error)
	Save(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id uint) error
}
