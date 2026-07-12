package userapp

import "time"

// ---- 输入 DTO：用例契约，不带 gin binding 标签，保持应用层与 Web 框架解耦 ----

// RegisterInput 注册用例输入
type RegisterInput struct {
	Username string
	Password string
	Phone    string
	Nickname string
}

// LoginInput 登录用例输入
type LoginInput struct {
	Username string
	Password string
	IP       string
}

// ---- 输出 DTO ----

// UserDTO 是对外暴露的用户视图，不含密码等敏感字段。
type UserDTO struct {
	UserID       uint       `json:"userId"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Username     string     `json:"username"`
	LoginChannel int        `json:"loginChannel"`
	UnionID      string     `json:"unionId"`
	OpenID       string     `json:"openId"`
	AlipayUID    string     `json:"alipayUid"`
	Nickname     string     `json:"nickname"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	Avatar       string     `json:"avatar"`
	Gender       int8       `json:"gender"`
	Birthday     *time.Time `json:"birthday,omitempty"`
	Status       int8       `json:"status"`
	LastLoginIP  string     `json:"lastLoginIp"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
}

// LoginResult 登录用例输出
type LoginResult struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}
