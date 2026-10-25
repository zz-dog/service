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

// MerchantDTO 商户对外视图
type MerchantDTO struct {
	MerchantID uint      `json:"merchantId"`
	UserID     uint      `json:"userId"`
	Name       string    `json:"name"`
	Logo       string    `json:"logo"`
	Desc       string    `json:"desc"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
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

func toMerchantDTO(m *domainmerchant.Merchant) MerchantDTO {
	return MerchantDTO{
		MerchantID: m.MerchantID,
		UserID:     m.UserID,
		Name:       m.Name,
		Logo:       m.Logo,
		Desc:       m.Desc,
		Status:     int(m.Status),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
