package user

import "errors"

// 领域错误：用哨兵错误表达业务语义。
// 仓储实现把底层错误（如 gorm.ErrRecordNotFound）翻译成这些领域错误，
// 上层据此映射 HTTP 响应，无需感知任何持久化细节。
var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrUserAlreadyExists  = errors.New("用户已存在")
	ErrUserDisabled       = errors.New("账号已被禁用")
	ErrInvalidCredentials = errors.New("账号或密码错误")
)
