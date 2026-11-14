package product

import "time"

// 商品状态

type Status int

const (
	StatusOnShelf       Status = iota + 1 // 上架
	StatusOffShelf                        // 下架
	StatusPendingReview                   // 待审核
	StatusRejected                        // 被拒绝
	StatusBanned                          // 禁用
)

// Product 是商品聚合根，管理 SKU 列表。
type Product struct {
	ProductID    uint      // 商品ID
	CategoryID   uint      // 分类ID
	MerchantID   uint      // 归属商户;0 = 平台自营。水平授权的锚点:商户写操作先比对这个字段
	Name         string    // 商品名称
	Desc         string    // 商品描述
	Status       Status    // 商品状态(1上架 2下架 3待审核 4已驳回 5违规下架)
	SKUs         []SKU     // 商品规格
	Medias       []Media   // 商品媒体资源（图片/视频/3D模型）
	RejectReason string    // 平台否决理由(驳回/违规下架共用一列;进阶上流水表再拆开)
	CreatedAt    time.Time // 创建时间
	UpdatedAt    time.Time // 更新时间
}

// NewProduct 创建商品(管理端:自营/代商户铺货,可信,出生即上架)。
// merchantID 允许 0 = 平台自营。
// 充血构造函数：校验名称、SKU 列表、SKU 编码与规格组合一致且不重复。
func NewProduct(merchantID uint, categoryID uint, name, desc string, skus []SKU) (*Product, error) {
	if name == "" {
		return nil, ErrEmptyProductName
	}
	if err := validateSKUs(skus); err != nil {
		return nil, err
	}
	return &Product{
		MerchantID: merchantID,
		CategoryID: categoryID,
		Name:       name,
		Desc:       desc,
		Status:     StatusOnShelf,
		SKUs:       skus,
	}, nil
}

// NewMerchantProduct 商户提交商品:出生即待审核,merchantID 必须 >0。
// "初态"是商品不变量的一部分:靠调用方事后补状态,就是在领域外养第二个真相。
// 与 NewProduct 的差异只有归属和初态——同一个 Create 用例,按调用者身份分流到
// 两个构造函数,状态裁决不散落在应用层的 if 里。
func NewMerchantProduct(merchantID uint, categoryID uint, name, desc string, skus []SKU) (*Product, error) {
	if merchantID == 0 {
		return nil, ErrInvalidMerchantID
	}
	if name == "" {
		return nil, ErrEmptyProductName
	}
	if err := validateSKUs(skus); err != nil {
		return nil, err
	}
	return &Product{
		MerchantID: merchantID,
		CategoryID: categoryID,
		Name:       name,
		Desc:       desc,
		Status:     StatusPendingReview,
		SKUs:       skus,
	}, nil
}

// UpdateInfo 修改商品基础信息（名称、描述、分类）。
func (p *Product) UpdateInfo(categoryID uint, name, desc string) error {
	if name == "" {
		return ErrEmptyProductName
	}
	p.CategoryID = categoryID
	p.Name = name
	p.Desc = desc

	return nil
}

// ReplaceMedias 替换全部媒体资源（管理端重新设置商品图片/视频/模型）。
func (p *Product) ReplaceMedias(medias []Media) error {
	if err := validateMedias(medias); err != nil {
		return err
	}
	p.Medias = medias
	return nil
}

// ReplaceSKUs 替换全部 SKU（管理端重新设置规格）。
// 校验编码与规格组合一致且不重复。
func (p *Product) ReplaceSKUs(skus []SKU) error {
	if err := validateSKUs(skus); err != nil {
		return err
	}
	p.SKUs = skus
	return nil
}

// UpdateSKU 更新某 SKU 的价格和图片（管理端调价）。
// 编码由规格组合派生，定位已有 SKU 只认 skuCode；规格组合和库存不动。
func (p *Product) UpdateSKU(skuCode string, price int64, picture string) error {
	if price <= 0 {
		return ErrInvalidPrice
	}
	idx, ok := p.FindSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	p.SKUs[idx].Price = price
	p.SKUs[idx].Picture = picture
	return nil
}

// RemoveSKU 删除某 SKU。
// 仅已下架商品允许(与删除商品同门槛)——修正:原守卫写反了(写成"仅上架允许"),
// 与注释意图相反,此处按注释意图落地。已下架商品回架前必然重审,删规格不必再回队列。
// 订单只引用 SKUCode 快照，删除不影响历史订单展示。
func (p *Product) RemoveSKU(skuCode string) error {
	if p.Status != StatusOffShelf {
		return ErrProductOnShelf
	}
	if len(p.SKUs) <= 1 {
		return ErrEmptySKUs
	}
	idx, ok := p.FindSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	p.SKUs = append(p.SKUs[:idx], p.SKUs[idx+1:]...)
	return nil
}

// validateSKUs 校验 SKU 列表：非空、字段合法、编码与规格组合一一对应、组合不重复。
// 编码由 DeriveSKUCode 派生，组合重复等价于编码重复，一并拒绝。
func validateSKUs(skus []SKU) error {
	if len(skus) == 0 {
		return ErrEmptySKUs
	}
	seen := make(map[string]bool, len(skus))
	for _, s := range skus {
		if err := s.validate(); err != nil {
			return err
		}
		if s.SKUCode != DeriveSKUCode(s.SpecItems) {
			return ErrSKUCodeMismatch
		}
		if seen[s.SKUCode] {
			return ErrDuplicateSpecCombination
		}
		seen[s.SKUCode] = true
	}
	return nil
}

// SubmitForReview 内容变更 → 回待审核。允许来自 已上架/已下架/已驳回;
// 已在待审核时不要调(原地改材料即可,状态不动)——由应用层判断。
// 这就是"驳回后重新提交"的形态:同一个商品回队列,不新建。
// 与入驻申请单相反:申请单是只读凭证(重提新建),商品是活数据(生命周期一站)。
func (p *Product) SubmitForReview() error {
	switch p.Status {
	case StatusOnShelf, StatusOffShelf, StatusRejected:
		p.Status = StatusPendingReview
		return nil
	default:
		return ErrInvalidStatusTransition
	}
}

// OffShelf 商户下架:已上架 → 已下架,非法迁移报错。
func (p *Product) OffShelf() error {
	if p.Status != StatusOnShelf {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusOffShelf
	return nil
}

// OnShelf 商户上架:已下架 → 已上架,免重审。
// 下架是商户的经营决策,不是合规否决——自己下的架自己回;
// 待审核/已驳回的商品没有"上架"动作,它们还没获得上过架的资格。
func (p *Product) OnShelf() error {
	if p.Status != StatusOffShelf {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusOnShelf
	return nil
}

// IsOnShelf 是否上架
func (p *Product) IsOnShelf() bool { return p.Status == StatusOnShelf }

// FindSKU 按 code 找 SKU的索引
func (p *Product) FindSKU(code string) (int, bool) {
	for i, s := range p.SKUs {
		if s.SKUCode == code {
			return i, true
		}
	}
	return 0, false
}

// DeductStock 扣减某 SKU 的库存（下单时调用）。
// 在聚合根内校验：库存不足返回 ErrInsufficientStock，下架商品返回 ErrProductOffShelf。
// 注意：内存层校验后，仓储层还要用条件更新防并发超卖。
func (p *Product) DeductStock(skuCode string, qty int) error {
	if !p.IsOnShelf() {
		return ErrProductOffShelf
	}
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.FindSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	if p.SKUs[idx].Stock < qty {
		return ErrInsufficientStock
	}
	p.SKUs[idx].Stock -= qty
	return nil
}

// Restock 增加某 SKU 库存（管理端补货）
func (p *Product) Restock(skuCode string, qty int) error {
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.FindSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	p.SKUs[idx].Stock += qty
	return nil
}

func (p *Product) CanDelete() bool {
	// 已下架(商户自主弃)与已驳回(未过审弃)都可删;
	// 待审核还在队列里,违规下架是处罚凭证,都不允许删
	return p.Status == StatusOffShelf || p.Status == StatusRejected
}

// Approve 待审核 → 已上架。通过即上架,不需要商户再手动点——
// 商户提交的就是"申请上架",通过即上架;想撤自己下架,下架免重审。
// 这里只裁决状态合法性;并发下"两个管理员同时审"由仓储条件更新兜底(阶段二),
// 双层各管一段:领域管规则,仓储管并发。
func (p *Product) Approve() error {
	if p.Status != StatusPendingReview {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusOnShelf
	return nil
}

// Reject 待审核 → 已拒绝。
func (p *Product) Reject(reason string) error {
	if p.Status != StatusPendingReview {
		return ErrInvalidStatusTransition
	}
	if reason == "" {
		return ErrRejectReasonRequired
	}
	p.Status = StatusRejected

	p.RejectReason = reason
	return nil
}

// Takedown 违规下架:已上架 → 违规下架,理由必填并落进 RejectReason。
// (原 Bann 校验了理由却没存,理由被丢掉——处置凭证缺了"为什么",已修正。)
// 状态 5 是终态:教学版无申诉,不需要任何出边方法。
// 与商户自主下架(OffShelf)结果都不可见,但语义不同:这是处罚,不可自行恢复。
func (p *Product) Takedown(reason string) error {
	if p.Status != StatusOnShelf {
		return ErrInvalidStatusTransition
	}
	if reason == "" {
		return ErrRejectReasonRequired
	}
	p.Status = StatusBanned
	p.RejectReason = reason
	return nil
}
