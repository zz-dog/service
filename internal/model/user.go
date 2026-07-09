package model

import (
	"time"

	"gorm.io/gorm"
)

// LoginChannel 登录渠道枚举 建议常量定义
const (
	ChannelPassword = 1 // 账号密码登录
	ChannelWechat   = 2 // 微信登录(小程序/公众号/App)
	ChannelAlipay   = 3 // 支付宝快捷登录
)

type User struct {
	// 主键与时间
	UserID    uint           `json:"userId" gorm:"primaryKey;autoIncrement;comment:用户主键ID"`
	CreatedAt time.Time      `json:"createdAt" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`

	// 自有账号登录（可选，第三方登录可为空）
	Username string `json:"username" gorm:"size:32;default:'';index;comment:自定义登录账号，第三方登录可空"`
	Password string `json:"-" gorm:"size:100;default:'';comment:BCrypt加密密码，第三方登录无密码"`

	// 第三方登录核心关联字段（重点新增）
	LoginChannel int    `json:"loginChannel" gorm:"tinyint;not null;default:1;comment:登录渠道 1账号密码 2微信 3支付宝"`
	UnionID      string `json:"unionId" gorm:"size:128;index;default:'';comment:微信UnionID(多端统一标识)"`
	OpenID       string `json:"openId" gorm:"size:128;index;default:'';comment:微信OpenID(单端唯一)"`
	AlipayUID    string `json:"alipayUid" gorm:"size:128;index;default:'';comment:支付宝用户唯一ID"`

	// 基础用户信息
	Nickname string     `json:"nickname" gorm:"size:32;default:'';comment:用户展示昵称，优先第三方昵称"`
	Phone    string     `json:"phone" gorm:"size:11;index;default:'';comment:绑定手机号，不唯一"`
	Email    string     `json:"email" gorm:"size:64;default:'';comment:邮箱"`
	Avatar   string     `json:"avatar" gorm:"size:255;default:'';comment:头像地址"`
	Gender   int8       `json:"gender" gorm:"tinyint;default:0;comment:0未知 1男 2女"`
	Birthday *time.Time `json:"birthday,omitempty" gorm:"comment:生日，允许为空"`

	// 账号状态
	Status int8 `json:"status" gorm:"tinyint;not null;default:1;comment:账号状态 1正常 0禁用"`

	// 登录记录
	LastLoginIP string     `json:"lastLoginIp" gorm:"size:64;default:'';comment:最后登录IP"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty" gorm:"comment:最后登录时间"`
}
