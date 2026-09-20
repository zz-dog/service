package categoryapp

import "context"

type CategoryCache interface {

	// GetCategories 获取所有分类缓存, 如果缓存不存在则返回 false
	GetCategories(ctx context.Context) ([]*CategoryDto, bool, error)
	SetCategories(ctx context.Context, categories []*CategoryDto) error

	// Invalidate 清除分类缓存
	Invalidate(ctx context.Context) error
}
