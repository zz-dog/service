package productapp

import (
	"context"
	"errors"
)

var (
	// ErrSpecNotBoundToCategory SKU 引用的规格维度未绑定到商品所属分类
	ErrSpecNotBoundToCategory = errors.New("规格维度未绑定到该分类")
	// ErrSpecValueNotInSpec SKU 引用的规格值不属于该规格维度
	ErrSpecValueNotInSpec = errors.New("规格值不属于该规格维度")
)

// CategorySpecView 分类绑定的规格视图：商品应用层做 SKU 归属校验所需的最小数据。
type CategorySpecView struct {
	SpecID   uint
	SpecName string
	Values   []SpecValueView
}

// SpecValueView 规格值视图
type SpecValueView struct {
	ValueID uint
	Name    string
}

// valueName 按值ID取规格库中的权威名称
func (v *CategorySpecView) valueName(valueID uint) (string, bool) {
	for _, val := range v.Values {
		if val.ValueID == valueID {
			return val.Name, true
		}
	}
	return "", false
}

// SpecDirectory 规格目录端口：查询分类绑定的规格集（含规格值）。
// SKU 规格项归属校验是跨聚合（商品×分类规格绑定）的一致性规则，
// 放在应用层；端口实现由组合根用分类模块适配，商品应用层不依赖分类模块。
type SpecDirectory interface {
	SpecsForCategory(ctx context.Context, categoryID uint) ([]*CategorySpecView, error)
}
