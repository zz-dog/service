package merchantapp

import (
	"time"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// SubmitInput 提交入驻申请入参。材料是 OSS 上传后的 URL(oss-upload-guide 的产出)。
type SubmitInput struct {
	ShopName     string
	ContactName  string
	ContactPhone string
	LicenseURL   string
	IDCardURL    string
}

// UpdateShopInput 更新店铺资料入参
type UpdateShopInput struct {
	Name string
	Logo string
	Desc string
}

// ApplicationDTO 申请单对外视图。状态原样给 int,前端按状态机渲染。
type ApplicationDTO struct {
	ID           uint       `json:"id"`
	UserID       uint       `json:"userId"`
	ShopName     string     `json:"shopName"`
	ContactName  string     `json:"contactName"`
	ContactPhone string     `json:"contactPhone"`
	LicenseURL   string     `json:"licenseUrl"`
	IDCardURL    string     `json:"idCardUrl"`
	Status       int        `json:"status"`
	RejectReason string     `json:"rejectReason"`
	MerchantID   *uint      `json:"merchantId"`
	CreatedAt    time.Time  `json:"createdAt"`
	ReviewedAt   *time.Time `json:"reviewedAt"`
}

// ShopDTO 我的店铺视图:N:N 下"我的店铺"是列表,每家店带着"我在它是谁"(角色)。
type ShopDTO struct {
	MerchantID uint      `json:"merchantId"`
	Name       string    `json:"name"`
	Logo       string    `json:"logo"`
	Desc       string    `json:"desc"`
	Status     int       `json:"status"`
	MyRole     int       `json:"myRole"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// MemberDTO 店铺成员视图。教学版只回 userID 不回昵称——用户资料在 identity 服务,
// merchant 服务没有它的 RPC;真实系统应在此做用户信息聚合或由前端补查。
type MemberDTO struct {
	UserID    uint      `json:"userId"`
	Role      int       `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// InviteMemberInput 邀请成员入参:直接指定目标 userID 与角色(admin/staff)。
// 教学版不校验目标用户存在(identity 未对 merchant 开放 RPC),也不做邀请码/申请-同意流;
// 真实系统应换成邀请链接或工单,并把"查无此人"拦在邀请入口。
type InviteMemberInput struct {
	TargetUserID uint
	Role         int
}

// ChangeMemberRoleInput 调整成员角色入参:只能 admin↔staff 互换,owner 永远不可动。
type ChangeMemberRoleInput struct {
	Role int
}

func toApplicationDTO(a *domainmerchant.MerchantApplication) ApplicationDTO {
	return ApplicationDTO{
		ID:           a.ID,
		UserID:       a.UserID,
		ShopName:     a.ShopName,
		ContactName:  a.ContactName,
		ContactPhone: a.ContactPhone,
		LicenseURL:   a.LicenseURL,
		IDCardURL:    a.IDCardURL,
		Status:       int(a.Status),
		RejectReason: a.RejectReason,
		MerchantID:   a.MerchantID,
		CreatedAt:    a.CreatedAt,
		ReviewedAt:   a.ReviewedAt,
	}
}

func toShopDTO(m *domainmerchant.Merchant, role domainmerchant.MemberRole) ShopDTO {
	return ShopDTO{
		MerchantID: m.MerchantID,
		Name:       m.Name,
		Logo:       m.Logo,
		Desc:       m.Desc,
		Status:     int(m.Status),
		MyRole:     int(role),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func toMemberDTO(m *domainmerchant.Member) MemberDTO {
	return MemberDTO{
		UserID:    m.UserID,
		Role:      int(m.Role),
		CreatedAt: m.CreatedAt,
	}
}
