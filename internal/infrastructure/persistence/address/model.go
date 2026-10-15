package addresspo

import (
	"time"

	"gorm.io/gorm"

	"github.com/wsc-zz/service/internal/domain/address"
)

// AddressPO 是收货地址的持久化对象（Persistence Object），对应数据库表。
// 它带有 GORM 标签，与领域实体 address.Address 分离，避免领域层耦合 ORM。
type AddressPO struct {
	AddressID uint `gorm:"primaryKey;autoIncrement;comment:地址ID"`
	UserID    uint `gorm:"index;not null;comment:所属用户ID"`

	Receiver string `gorm:"size:32;not null;comment:收货人姓名"`
	Phone    string `gorm:"size:16;not null;comment:联系电话"`
	Province string `gorm:"size:32;not null;comment:省"`
	City     string `gorm:"size:32;not null;comment:市"`
	District string `gorm:"size:32;not null;comment:区/县"`
	Detail   string `gorm:"size:255;not null;comment:详细地址"`

	// 是否默认地址。不加 uniqueIndex(user_id, is_default) 硬约束：
	// 软删除的行也占唯一索引，业务唯一由"先清零再置一"的事务保证
	IsDefault bool `gorm:"default:false;comment:是否默认地址"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (AddressPO) TableName() string { return "user_addresses" }

// toDomain 将持久化对象转为领域实体。
func toDomain(po *AddressPO) *address.Address {
	return &address.Address{
		AddressID: po.AddressID,
		UserID:    po.UserID,
		Receiver:  po.Receiver,
		Phone:     po.Phone,
		Province:  po.Province,
		City:      po.City,
		District:  po.District,
		Detail:    po.Detail,
		IsDefault: po.IsDefault,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// toPO 将领域实体转为持久化对象。
func toPO(a *address.Address) *AddressPO {
	return &AddressPO{
		AddressID: a.AddressID,
		UserID:    a.UserID,
		Receiver:  a.Receiver,
		Phone:     a.Phone,
		Province:  a.Province,
		City:      a.City,
		District:  a.District,
		Detail:    a.Detail,
		IsDefault: a.IsDefault,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
