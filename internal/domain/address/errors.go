package address

import "errors"

// 领域错误：用哨兵错误表达业务语义。
// 仓储实现把底层错误（如 gorm.ErrRecordNotFound）翻译成这些领域错误，
// 上层据此映射 HTTP 响应，无需感知任何持久化细节。
var (
	ErrAddressNotFound = errors.New("收货地址不存在")
	ErrAddressLimit    = errors.New("收货地址数量已达上限（20条）")
	ErrInvalidPhone    = errors.New("手机号格式错误")
	ErrInvalidReceiver = errors.New("收货人姓名不能为空")
	ErrInvalidRegion   = errors.New("省市区不能为空")
	ErrInvalidDetail   = errors.New("详细地址不能为空")
	ErrAddressNotOwned = errors.New("无权操作该地址")
)
