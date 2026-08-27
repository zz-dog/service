package category

// CategorySpec 分类-规格绑定（多对多关联）：
// 表达"某分类启用了哪些规格维度"。Sort/Required 是属于关系本身的属性，
// 而不属于分类或规格任何一方，所以独立成一张绑定表。
// 跨聚合只持ID：这里仅引用 CategoryID / SpecID，不持有对方对象。
type CategorySpec struct {
	CategoryID uint // 分类ID（联合主键）
	SpecID     uint // 规格维度ID（联合主键）
	Sort       int  // 该规格在此分类下的展示排序，越小越靠前
	Required   bool // 商品发布到该分类时是否必选该规格
}

// NewCategorySpec 创建绑定关系。
func NewCategorySpec(categoryID, specID uint, sort int, required bool) (*CategorySpec, error) {
	if categoryID == 0 || specID == 0 {
		return nil, ErrInvalidCategorySpec
	}
	return &CategorySpec{
		CategoryID: categoryID,
		SpecID:     specID,
		Sort:       sort,
		Required:   required,
	}, nil
}

// SetSort 设置该规格在分类下的排序。
func (cs *CategorySpec) SetSort(sort int) { cs.Sort = sort }

// SetRequired 设置是否必选。
func (cs *CategorySpec) SetRequired(required bool) { cs.Required = required }
