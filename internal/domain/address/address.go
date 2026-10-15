package address

import (
	"regexp"
	"time"
)

// AddressMaxCount 是每个用户允许拥有的收货地址上限，防止刷库。
const AddressMaxCount = 20

// phoneRegex 匹配 11 位大陆手机号：1 开头，第二位 3-9。
var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Address 是收货地址聚合根（领域实体）。
// 纯净结构：不带任何 GORM / JSON 标签，不依赖任何外部库。
// 持久化由 infrastructure/persistence/address.AddressPO 负责，二者通过映射函数转换。
type Address struct {
	AddressID uint
	UserID    uint
	Receiver  string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAddress 创建一条收货地址（充血模型：不变量集中在领域层校验）。
// 规则：收货人/省/市/区/详细地址非空，手机号 11 位大陆手机号，详情 ≤255 字。
func NewAddress(userID uint, receiver, phone, province, city, district, detail string) (*Address, error) {
	a := &Address{
		UserID:   userID,
		Receiver: receiver,
		Phone:    phone,
		Province: province,
		City:     city,
		District: district,
		Detail:   detail,
	}
	if err := a.validate(); err != nil {
		return nil, err
	}
	return a, nil
}

// validate 校验地址字段的不变量，Create 与 Update 共用。
func (a *Address) validate() error {
	if a.Receiver == "" {
		return ErrInvalidReceiver
	}
	if !phoneRegex.MatchString(a.Phone) {
		return ErrInvalidPhone
	}
	if a.Province == "" || a.City == "" || a.District == "" {
		return ErrInvalidRegion
	}
	if a.Detail == "" {
		return ErrInvalidDetail
	}
	return nil
}

// Update 更新地址字段，传 nil 表示不修改（部分更新）。
// 非法字段值返回错误，与 user.UpdateProfile 同风格。
func (a *Address) Update(receiver, phone, province, city, district, detail *string) error {
	if receiver != nil {
		a.Receiver = *receiver
	}
	if phone != nil {
		a.Phone = *phone
	}
	if province != nil {
		a.Province = *province
	}
	if city != nil {
		a.City = *city
	}
	if district != nil {
		a.District = *district
	}
	if detail != nil {
		a.Detail = *detail
	}
	return a.validate()
}

// BelongsTo 地址是否属于指定用户（越权判定在领域层提供，应用层调用）。
func (a *Address) BelongsTo(userID uint) bool {
	return a.UserID == userID
}
