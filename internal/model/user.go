package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	// 主键
	UserID uint `json:"userId" gorm:"primaryKey;autoIncrement;comment:主键"`
	// 时间戳（替代 gorm.Model）
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 登录核心字段
	Username string `json:"username" gorm:"size:32;not null;unique;comment:登录账号"`
	Password string `json:"-" gorm:"size:100;not null;comment:加密密码"`
	// 基础信息
	Nickname string     `json:"nickname" gorm:"size:32;default:'';comment:昵称"`
	Phone    string     `json:"phone" gorm:"size:11;unique;comment:手机号"`
	Email    string     `json:"email" gorm:"size:64;comment:邮箱"`
	Avatar   string     `json:"avatar" gorm:"size:255;default:'';comment:头像url"`
	Gender   int8       `json:"gender" gorm:"default:0;comment:0未知 1男 2女"`
	Birthday *time.Time `json:"birthday,omitempty" gorm:"comment:生日"`
	// 账号状态
	Status int8 `json:"status" gorm:"not null;default:1;comment:1正常 0禁用"`
	// 登录记录
	LastLoginIP string     `json:"lastLoginIp" gorm:"size:64;default:'';comment:最后登录IP"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty" gorm:"comment:最后登录时间"`
}
