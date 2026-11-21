package merchantpo

import (
	"time"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MerchantApplicationPO 入驻申请。reject_reason / merchant_id 都是凭证的一部分:
// 前者记录"为什么被拒",后者记录"这次申请产出了哪家伙"。
type MerchantApplicationPO struct {
	ID           uint                       `gorm:"primaryKey"`
	UserID       uint                       `gorm:"index"`            // 不加唯一索引:一个人可以有多张历史申请
	ShopName     string                     `gorm:"type:varchar(64)"` // 申请单里不唯一——重名申请都可以提交,唯一性在 merchants 表裁决
	ContactName  string                     `gorm:"type:varchar(32)"`
	ContactPhone string                     `gorm:"type:varchar(16)"`
	LicenseURL   string                     `gorm:"type:varchar(512)"`
	IDCardURL    string                     `gorm:"type:varchar(512)"`
	Status       domainmerchant.ApplyStatus `gorm:"index"`
	RejectReason string                     `gorm:"type:varchar(255)"`
	MerchantID   *uint                      // 可空:待审核/已驳回时没有指向任何商户
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ReviewedAt   *time.Time
}

func (MerchantApplicationPO) TableName() string { return "merchant_applications" }

// MerchantPO 商户。不再挂 user_id:N:N 归属的事实源在 merchant_members 表,
// 谁是店主 = role=owner 的那条成员记录——归属只有一个事实源,两处各说各话迟早漂移。
type MerchantPO struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"type:varchar(64);uniqueIndex"`
	Logo      string `gorm:"type:varchar(512)"`
	Desc      string `gorm:"type:varchar(512)"`
	Status    domainmerchant.MerchantStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (MerchantPO) TableName() string { return "merchants" }

// MemberPO 店铺成员。(user_id, merchant_id) 复合唯一索引是"一人一店一条成员记录"的最终防线:
// 应用层的重复邀请检查只是友好报错,并发下(两个入口同时邀请同一人)只有索引拦得住。
type MemberPO struct {
	ID         uint                       `gorm:"primaryKey"`
	UserID     uint                       `gorm:"uniqueIndex:uk_user_merchant"`
	MerchantID uint                       `gorm:"uniqueIndex:uk_user_merchant;index"`
	Role       domainmerchant.MemberRole  // 1=owner 2=admin 3=staff
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (MemberPO) TableName() string { return "merchant_members" }

// topo 领域申请单 → PO。落库前必须转换:领域结构和 PO 是两套结构,
// 现在字段名恰好一致是运气,一旦漂移(领域改字段/PO 加列)直接喂 GORM 就是静默坏表。
func topo(a *domainmerchant.MerchantApplication) *MerchantApplicationPO {
	return &MerchantApplicationPO{
		ID:           a.ID,
		UserID:       a.UserID,
		ShopName:     a.ShopName,
		ContactName:  a.ContactName,
		ContactPhone: a.ContactPhone,
		LicenseURL:   a.LicenseURL,
		IDCardURL:    a.IDCardURL,
		Status:       a.Status,
		RejectReason: a.RejectReason,
		MerchantID:   a.MerchantID,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		ReviewedAt:   a.ReviewedAt,
	}
}

// topoMerchant 领域商户 → PO。主键字段名对不上(领域叫 MerchantID,表列是 id),
// 显式安放在这里——这就是 Save 不许直接喂领域对象的原因。
func topoMerchant(m *domainmerchant.Merchant) *MerchantPO {
	return &MerchantPO{
		ID:        m.MerchantID,
		Name:      m.Name,
		Logo:      m.Logo,
		Desc:      m.Desc,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// topoMember 领域成员 → PO。成员是新聚合,同商户一样显式安放,不许直接喂 GORM。
func topoMember(m *domainmerchant.Member) *MemberPO {
	return &MemberPO{
		ID:         m.ID,
		UserID:     m.UserID,
		MerchantID: m.MerchantID,
		Role:       m.Role,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func todomain(po *MerchantApplicationPO) *domainmerchant.MerchantApplication {
	return &domainmerchant.MerchantApplication{
		ID:           po.ID,
		UserID:       po.UserID,
		ShopName:     po.ShopName,
		ContactName:  po.ContactName,
		ContactPhone: po.ContactPhone,
		LicenseURL:   po.LicenseURL,
		IDCardURL:    po.IDCardURL,
		Status:       po.Status,
		RejectReason: po.RejectReason,
		ReviewedAt:   po.ReviewedAt,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func todomains(pos []MerchantApplicationPO) []domainmerchant.MerchantApplication {
	var domains []domainmerchant.MerchantApplication
	domains = make([]domainmerchant.MerchantApplication, len(pos))
	for i, po := range pos {
		domains[i] = *todomain(&po)
	}
	return domains
}

func todomainMerchant(po *MerchantPO) *domainmerchant.Merchant {
	return &domainmerchant.Merchant{
		MerchantID: po.ID,
		Name:       po.Name,
		Logo:       po.Logo,
		Desc:       po.Desc,
		Status:     po.Status,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func todomainMember(po *MemberPO) *domainmerchant.Member {
	return &domainmerchant.Member{
		ID:         po.ID,
		UserID:     po.UserID,
		MerchantID: po.MerchantID,
		Role:       po.Role,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func todomainMembers(pos []MemberPO) []domainmerchant.Member {
	domains := make([]domainmerchant.Member, len(pos))
	for i := range pos {
		domains[i] = *todomainMember(&pos[i])
	}
	return domains
}
