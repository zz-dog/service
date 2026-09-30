package inventory

import "errors"

var ErrDuplicateRequest = errors.New("重复的库存操作请求")

var ErrInsufficientInventory = errors.New("库存不足")
var ErrInvalidRequest = errors.New("无效的库存操作请求")

var ErrOrderNotFound = errors.New("订单不存在")
var ErrInvalidRequestID = errors.New("无效的请求ID")
var ErrSKUNotFound = errors.New("SKU不存在")
