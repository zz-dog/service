package user

import "time"

// 登录渠道
const (
	ChannelPassword = 1 // 账号密码登录
	ChannelWechat   = 2 // 微信登录(小程序/公众号/App)
	ChannelAlipay   = 3 // 支付宝快捷登录
)

// User 是用户聚合根（领域实体）。
// 纯净结构：不带任何 GORM / JSON 标签，不依赖任何外部库。
// 持久化由 infrastructure/persistence/user.UserPO 负责，二者通过映射函数转换。
type User struct {
	UserID    uint
	CreatedAt time.Time
	UpdatedAt time.Time

	// 自有账号登录（第三方登录可为空）
	Username string
	Password string

	// 第三方登录核心关联字段
	LoginChannel int
	UnionID      string
	OpenID       string
	AlipayUID    string

	// 基础用户信息
	Nickname string
	Phone    string
	Email    string
	Avatar   string
	Gender   int8
	Birthday *time.Time

	// 账号状态：1 正常，0 禁用
	Status int8

	// 登录记录
	LastLoginIP string
	LastLoginAt *time.Time
}

// NewUser 创建一个新注册的密码渠道用户。
// 作为充血模型的构造函数，集中封装不变量：默认密码渠道、正常状态。
func NewUser(username, hashedPassword, phone, nickname string) *User {
	return &User{
		Username:     username,
		Password:     hashedPassword,
		Phone:        phone,
		Nickname:     nickname,
		LoginChannel: ChannelPassword,
		Status:       1,
	}
}

// VerifyPassword 校验明文密码是否正确，具体的比对算法由 hasher 提供。
func (u *User) VerifyPassword(plain string, hasher PasswordHasher) bool {
	return hasher.Compare(u.Password, plain) == nil
}

// IsDisabled 账号是否被禁用。
func (u *User) IsDisabled() bool {
	return u.Status == 0
}

// RecordLogin 记录登录信息（更新最后登录 IP 与时间）。
func (u *User) RecordLogin(ip string, at time.Time) {
	u.LastLoginIP = ip
	u.LastLoginAt = &at
}
