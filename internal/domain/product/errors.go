package product

import "errors"

var (
	ErrProductNotFound = errors.New("商品不存在")
	ErrSKUNotFound     = errors.New("商品规格不存在")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrInsufficientStock  = errors.New("库存不足")
	ErrProductOffShelf    = errors.New("商品已下架")
	ErrEmptySKUs          = errors.New("商品至少需要一个规格")
	// ErrSKUCodeMismatch SKU 编码与规格组合派生的编码不一致（编码应由服务端派生，不收自定义值）
	ErrSKUCodeMismatch = errors.New("SKU编码与规格组合不一致")
	// ErrDuplicateSpecCombination 多个 SKU 规格组合相同（编码为组合派生，等价于编码重复）
	ErrDuplicateSpecCombination = errors.New("SKU规格组合重复")
	ErrInvalidPrice       = errors.New("价格必须大于0")
	ErrInvalidStock       = errors.New("库存不能为负数")
	ErrEmptySKUCode       = errors.New("规格编码不能为空")
	ErrEmptySpecItems     = errors.New("SKU至少需要一个规格项")
	ErrInvalidSpecItem    = errors.New("规格项的维度ID、值ID、名称不能为空")
	ErrDuplicateSpecInSKU = errors.New("SKU内同一规格维度不能出现多次")
	ErrEmptyProductName   = errors.New("商品名称不能为空")
	ErrEmptyCategoryName  = errors.New("分类名称不能为空")
	ErrCategoryHasProduct = errors.New("分类下还有商品，不能删除")
)
