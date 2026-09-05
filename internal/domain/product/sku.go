package product

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// SpecItem SKU 规格项（值对象）：引用规格库的维度和值。
// ID（SpecID/ValueID）用于结构化校验和关联，名称（SpecName/ValueName）是快照，
// 规格库后续改名或删值不影响历史数据展示（订单渲染靠快照）。
type SpecItem struct {
	SpecID    uint   // 规格维度ID，如 "颜色"
	ValueID   uint   // 规格值ID，如 "红色"
	SpecName  string // 维度名快照
	ValueName string // 值名快照
}

// SKU 是商品规格（值对象）。
// 一个商品有多个 SKU（如"红色 L 码"、"蓝色 M 码"），每个 SKU 独立价格和库存。
// 一个 SKU 由若干规格项组合而成（红色+L码 = 一个 SKU）。
// 没有独立身份，作为 Product 聚合根的一部分存在。
type SKU struct {
	// SKUCode 规格编码，商品内唯一。不收客户端自定义值，
	// 由规格组合派生（见 DeriveSKUCode），同组合必同码。
	SKUCode   string     // 规格编码，如 "10-20"（值ID按维度ID排序拼接）
	SpecItems []SpecItem // 规格组合，如 [{颜色:红色},{尺码:L}]
	Price     int64      // 单价，单位：分
	Stock     int        // 库存
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeriveSKUCode 从规格组合派生 SKU 编码：值ID 按维度ID升序拼接。
// 编码是组合的确定性函数——同组合必同码，组合唯一则编码唯一。
// 客户端不传编码，服务端在装配 SKU 时调用。
func DeriveSKUCode(items []SpecItem) string {
	sorted := make([]SpecItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SpecID < sorted[j].SpecID })
	ids := make([]string, 0, len(sorted))
	for _, item := range sorted {
		ids = append(ids, strconv.FormatUint(uint64(item.ValueID), 10))
	}
	return strings.Join(ids, "-")
}

// SpecDesc 拼接展示文案，如 "红色 / L码"（替代原 Spec 自由文本字段）。
func (s SKU) SpecDesc() string {
	names := make([]string, 0, len(s.SpecItems))
	for _, item := range s.SpecItems {
		names = append(names, item.ValueName)
	}
	return strings.Join(names, " / ")
}

// validate 校验 SKU 字段
func (s SKU) validate() error {
	if s.SKUCode == "" {
		return ErrEmptySKUCode
	}
	if s.Price <= 0 {
		return ErrInvalidPrice
	}
	if s.Stock < 0 {
		return ErrInvalidStock
	}
	// 规格组合：至少一项，字段齐全，同一维度不能出现两次（一个 SKU 不能既红色又蓝色）
	if len(s.SpecItems) == 0 {
		return ErrEmptySpecItems
	}
	seen := make(map[uint]bool, len(s.SpecItems))
	for _, item := range s.SpecItems {
		if item.SpecID == 0 || item.ValueID == 0 || item.SpecName == "" || item.ValueName == "" {
			return ErrInvalidSpecItem
		}
		if seen[item.SpecID] {
			return ErrDuplicateSpecInSKU
		}
		seen[item.SpecID] = true
	}
	return nil
}
