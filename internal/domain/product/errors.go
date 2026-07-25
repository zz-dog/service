package product

import "errors"

var (
	ErrProductNotFound    = errors.New("商品不存在")
	ErrSKUNotFound        = errors.New("商品规格不存在")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrInsufficientStock  = errors.New("库存不足")
	ErrProductOffShelf    = errors.New("商品已下架")
	ErrEmptySKUs          = errors.New("商品至少需要一个规格")
	ErrDuplicateSKUCode   = errors.New("商品规格编码重复")
	ErrInvalidPrice       = errors.New("价格必须大于0")
	ErrInvalidStock       = errors.New("库存不能为负数")
	ErrEmptyProductName   = errors.New("商品名称不能为空")
	ErrEmptyCategoryName  = errors.New("分类名称不能为空")
	ErrCategoryHasProduct = errors.New("分类下还有商品，不能删除")
)
