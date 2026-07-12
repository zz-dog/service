package userpo

import (
	"time"

	"gorm.io/gorm"

	"github.com/wsc-zz/service/internal/domain/user"
)

// UserPO 是用户的持久化对象（Persistence Object），对应数据库表。
// 它带有 GORM 标签，与领域实体 user.User 分离，避免领域层耦合 ORM。
type UserPO struct {
	UserID    uint           `gorm:"primaryKey;autoIncrement;comment:用户主键ID"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:软删除时间"`

	// 自有账号登录（第三方登录可为空）
	Username string `gorm:"size:32;default:'';index;comment:自定义登录账号，第三方登录可空"`
	Password string `gorm:"size:100;default:'';comment:BCrypt加密密码，第三方登录无密码"`

	// 第三方登录核心关联字段
	LoginChannel int    `gorm:"tinyint;not null;default:1;comment:登录渠道 1账号密码 2微信 3支付宝"`
	UnionID      string `gorm:"size:128;index;default:'';comment:微信UnionID(多端统一标识)"`
	OpenID       string `gorm:"size:128;index;default:'';comment:微信OpenID(单端唯一)"`
	AlipayUID    string `gorm:"size:128;index;default:'';comment:支付宝用户唯一ID"`

	// 基础用户信息
	Nickname string     `gorm:"size:32;default:'';comment:用户展示昵称"`
	Phone    string     `gorm:"size:11;index;default:'';comment:绑定手机号，不唯一"`
	Email    string     `gorm:"size:64;default:'';comment:邮箱"`
	Avatar   string     `gorm:"size:255;default:'';comment:头像地址"`
	Gender   int8       `gorm:"tinyint;default:0;comment:0未知 1男 2女"`
	Birthday *time.Time `gorm:"comment:生日，允许为空"`

	// 账号状态
	Status int8 `gorm:"tinyint;not null;default:1;comment:账号状态 1正常 0禁用"`

	// 登录记录
	LastLoginIP string     `gorm:"size:64;default:'';comment:最后登录IP"`
	LastLoginAt *time.Time `gorm:"comment:最后登录时间"`
}

// toDomain 将持久化对象转为领域实体。
func toDomain(po *UserPO) *user.User {
	return &user.User{
		UserID:       po.UserID,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
		Username:     po.Username,
		Password:     po.Password,
		LoginChannel: po.LoginChannel,
		UnionID:      po.UnionID,
		OpenID:       po.OpenID,
		AlipayUID:    po.AlipayUID,
		Nickname:     po.Nickname,
		Phone:        po.Phone,
		Email:        po.Email,
		Avatar:       po.Avatar,
		Gender:       po.Gender,
		Birthday:     po.Birthday,
		Status:       po.Status,
		LastLoginIP:  po.LastLoginIP,
		LastLoginAt:  po.LastLoginAt,
	}
}

// toPO 将领域实体转为持久化对象。
func toPO(u *user.User) *UserPO {
	return &UserPO{
		UserID:       u.UserID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Username:     u.Username,
		Password:     u.Password,
		LoginChannel: u.LoginChannel,
		UnionID:      u.UnionID,
		OpenID:       u.OpenID,
		AlipayUID:    u.AlipayUID,
		Nickname:     u.Nickname,
		Phone:        u.Phone,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Gender:       u.Gender,
		Birthday:     u.Birthday,
		Status:       u.Status,
		LastLoginIP:  u.LastLoginIP,
		LastLoginAt:  u.LastLoginAt,
	}
}
