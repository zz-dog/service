package spec

import "time"

// SpecValue 规格属性值（聚合内实体）：某维度下的一个标准可选值，如"红色"。
// 有独立身份（ValueID），因为商品 SKU 会通过 ValueID 引用它。
type SpecValue struct {
	ValueID uint   // 规格值ID
	SpecID  uint   // 规格维度ID
	Name    string // 如 "红色"
	Sort    int    // 排序
}

type Spec struct {
	SpecID    uint        // 规格ID
	Name      string      // 如 "颜色"
	Sort      int         // 排序
	Values    []SpecValue // 规格值列表
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewSpec 创建规格维度。
func NewSpec(name string, sort int) (*Spec, error) {
	if name == "" {
		return nil, ErrEmptySpecName
	}
	return &Spec{Name: name, Sort: sort}, nil
}

// Rename 重命名维度。
func (a *Spec) Rename(name string) error {
	if name == "" {
		return ErrEmptySpecName
	}
	a.Name = name
	return nil
}

// SetSort 设置排序权重，越小越靠前。
func (a *Spec) SetSort(sort int) { a.Sort = sort }

// findValueByName 按名称查找值索引（用于防重）。
func (a *Spec) findValueByName(name string) (int, bool) {
	for i, v := range a.Values {
		if v.Name == name {
			return i, true
		}
	}
	return 0, false
}

// AddValue 添加标准值；名称为空或重复则报错。
func (a *Spec) AddValue(name string) error {
	if name == "" {
		return ErrEmptySpecValueName
	}
	if _, ok := a.findValueByName(name); ok {
		return ErrDuplicateSpecValue
	}
	a.Values = append(a.Values, SpecValue{SpecID: a.SpecID, Name: name})
	return nil
}

// RemoveValue 按 ID 删除标准值；不存在则报错。
func (a *Spec) RemoveValue(valueID uint) error {
	for i, v := range a.Values {
		if v.ValueID == valueID {
			a.Values = append(a.Values[:i], a.Values[i+1:]...)
			return nil
		}
	}
	return ErrSpecValueNotFound
}
