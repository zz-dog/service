package category

import "errors"

var (
	ErrEmptyCategoryName  = errors.New("分类名称不能为空")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrCategoryHasProduct = errors.New("分类下还有商品，不能删除")
)
