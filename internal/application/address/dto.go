package addressapp

import "time"

// CreateInput 新增收货地址的入参。
type CreateInput struct {
	UserID    uint
	Receiver  string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool // 可选；true 时清掉已有默认地址，与保存放同一事务
}

// UpdateInput 更新收货地址的入参。
// 字段全部用指针实现部分更新，对齐 categoryapp.UpdateCategoryInput。
type UpdateInput struct {
	AddressID uint
	UserID    uint
	Receiver  *string
	Phone     *string
	Province  *string
	City      *string
	District  *string
	Detail    *string
}

// AddressDto 地址视图，JSON 序列化给前端。
type AddressDto struct {
	AddressID uint      `json:"addressId"`
	Receiver  string    `json:"receiver"`
	Phone     string    `json:"phone"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	District  string    `json:"district"`
	Detail    string    `json:"detail"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
